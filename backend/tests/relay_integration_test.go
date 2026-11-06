package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
	"v2ray-dash/backend/pkg/database"
)

func TestRelayManagementWorkflow(t *testing.T) {
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("set TEST_DATABASE_URL to run the isolated relay workflow")
	}
	db, err := database.NewPostgres(connection)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	schema := pq.QuoteIdentifier("relay_" + strings.ReplaceAll(uuid.NewString(), "-", ""))
	_, err = db.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	_, err = db.Exec("SET search_path TO " + schema)
	require.NoError(t, err)
	require.NoError(t, database.InitSchema(db))
	require.NoError(t, database.InitSchema(db))
	router := newRouter(db)
	w := request(t, router, "POST", "/api/auth/setup", `{"username":"relay-admin","password":"relay-test-password"}`, "", 200)
	var auth struct{ Token string }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &auth))
	nodes := repository.NewServerRepository(db.DB)
	accounts := repository.NewAccountRepository(db.DB)
	jsonRequest := func(method, path string, body any, want int) *httptest.ResponseRecorder {
		encoded, e := json.Marshal(body)
		require.NoError(t, e)
		return request(t, router, method, path, string(encoded), auth.Token, want)
	}
	createNode := func(name, ip, role string, managed bool) *model.Server {
		w := jsonRequest("POST", "/api/servers", map[string]any{"name": name, "ip": ip, "role": role, "country_code": "US", "ssh_port": 9, "ssh_user": "root", "reality_enabled": true, "reality_server_name": "academy.nvidia.com", "reality_public_key": "test-public-key", "reality_port": 443}, 201)
		var server model.Server
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &server))
		require.Equal(t, role, server.Role)
		if managed {
			require.NoError(t, nodes.SaveNode(&model.ManagedNode{ServerID: server.ID, Role: role, Layout: role, InboundTag: "entry", ShortID: "abcdef1234", EgressIP: ip, ConfigHash: "fixture", SSHHostKey: "fixture"}))
		}
		return &server
	}
	entry := createNode("中转A", "192.0.2.1", "relay", true)
	entryB := createNode("中转B", "192.0.2.2", "relay", true)
	first := createNode("出口a", "192.0.2.10", "exit", true)
	second := createNode("出口b", "192.0.2.20", "exit", true)
	legacy := createNode("原有直连", "192.0.2.30", "exit", false)
	for id, country := range map[string]string{entry.ID: "HK", entryB.ID: "HK", second.ID: "JP"} {
		_, err := nodes.Update(id, &model.UpdateServerRequest{CountryCode: &country})
		require.NoError(t, err)
	}
	createCombination := func(name, relay, exit string) string {
		w := jsonRequest("POST", "/api/relay-combinations", model.CombinationRequest{Name: name, RelayServerID: relay, ExitServerID: exit}, 201)
		var result struct{ ID string }
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		return result.ID
	}
	comboA := createCombination("A到a", entry.ID, first.ID)
	comboB := createCombination("A到b", entry.ID, second.ID)
	comboC := createCombination("B到a", entryB.ID, first.ID)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE NOT transport`).Scan(&count))
	require.Zero(t, count, "creating a combination must never create customer accounts")
	require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT exit_account_id) FROM relay_combinations`).Scan(&count))
	require.Equal(t, 3, count, "each combination prepares a dedicated exit credential immediately")
	connectionFor := func(id string) model.RelayConnection {
		w := request(t, router, "GET", "/api/relay-combinations/"+id+"/connection", "", auth.Token, 200)
		require.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
		var connection model.RelayConnection
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &connection))
		return connection
	}
	request(t, router, "GET", "/api/relay-combinations/"+comboA+"/connection", "", "", 401)
	connectionA := connectionFor(comboA)
	require.False(t, connectionA.Ready)
	require.Equal(t, first.IP, connectionA.Address)
	require.Equal(t, 443, connectionA.Port)
	require.Contains(t, connectionA.URI, "vless://"+connectionA.UUID+"@")
	require.Contains(t, connectionA.URI, "sid=abcdef1234")
	require.NotEqual(t, connectionA.UUID, connectionFor(comboC).UUID, "different relays get separate exit credentials")
	jsonRequest("POST", "/api/relay-combinations", model.CombinationRequest{Name: "未接管出口", RelayServerID: entry.ID, ExitServerID: legacy.ID}, 400)
	jsonRequest("POST", "/api/relay-combinations", model.CombinationRequest{Name: "重复路径", RelayServerID: entry.ID, ExitServerID: first.ID}, 400)
	jsonRequest("POST", "/api/relay-combinations", model.CombinationRequest{Name: "错误方向", RelayServerID: first.ID, ExitServerID: entry.ID}, 400)
	jsonRequest("POST", "/api/relay-combinations", model.CombinationRequest{Name: "中转链", RelayServerID: entry.ID, ExitServerID: entryB.ID}, 400)
	w = jsonRequest("POST", "/api/subscriptions", model.CreateSubscriptionRequest{Name: "混合订阅", AccountMappings: []model.AccountMapping{{ServerID: first.ID, AutoCreate: true}, {CombinationID: comboA, AutoCreate: true}, {CombinationID: comboB, AutoCreate: true}}}, 201)
	var sub model.SubscriptionWithAccounts
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sub))
	require.Len(t, sub.Accounts, 3)
	gateway := []*model.Account{}
	for _, a := range sub.Accounts {
		if a.CombinationID != "" {
			gateway = append(gateway, &a.Account)
			require.NotEmpty(t, a.CombinationName)
		}
	}
	require.Len(t, gateway, 2)
	require.NotEqual(t, gateway[0].UUID, gateway[1].UUID)
	request(t, router, "GET", "/api/subscribe/"+sub.UUID+"?format=clash_meta", "", "", 503)
	routes, err := nodes.ListRoutes()
	require.NoError(t, err)
	require.Len(t, routes, 2)
	var upstream *model.Account
	for _, route := range routes {
		a, e := accounts.GetByID(route.ExitAccountID)
		require.NoError(t, e)
		require.True(t, a.Transport)
		upstream = a
	}
	// A failed finalization cannot partially mark routes as applied.
	err = nodes.FinalizePublish(entry.ID, "new-hash", routes, []*model.Account{{ID: uuid.NewString(), UUID: uuid.NewString()}}, map[string]bool{})
	require.Error(t, err)
	unchanged, e := nodes.ListRoutes()
	require.NoError(t, e)
	for _, route := range unchanged {
		require.Empty(t, route.AppliedExitAccountID)
	}
	// Only publication metadata is seeded: actual core validation and filesystem rollback have separate tests.
	for _, server := range []*model.Server{entry, entryB, first, second} {
		list, e := accounts.ListByServerID(server.ID)
		require.NoError(t, e)
		live := map[string]bool{}
		for _, a := range list {
			live[a.ID] = true
		}
		applied := []*model.RelayRoute{}
		if server.ID == entry.ID {
			applied = routes
		}
		require.NoError(t, nodes.FinalizePublish(server.ID, "fixture", applied, list, live))
	}
	require.True(t, connectionFor(comboA).Ready)
	w = request(t, router, "GET", "/api/subscribe/"+sub.UUID+"?format=clash_meta", "", "", 200)
	var profile map[string]any
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &profile))
	require.Len(t, profile["proxies"], 3)
	require.Len(t, profile["proxy-groups"], 13)
	require.Len(t, profile["rules"], 872)
	for _, client := range []struct{ ua, format, fingerprint string }{
		{"Shadowrocket/2.2.94", "", "Safari26_3"},
		{"Shadowrocket/2.2.94", "clash_meta", "chrome"},
		{"ClashVerge/2.5.2", "", "chrome"},
		{"mihomo/1.19", "", "chrome"},
		{"", "shadowrocket", "Safari26_3"},
	} {
		path := "/api/subscribe/" + sub.UUID
		if client.format != "" {
			path += "?format=" + client.format
		}
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("User-Agent", client.ua)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Equal(t, "User-Agent", response.Header().Get("Vary"))
		var body map[string]any
		require.NoError(t, yaml.Unmarshal(response.Body.Bytes(), &body))
		clients := body["proxies"].([]any)
		require.Len(t, clients, 3)
		for _, raw := range clients {
			require.Equal(t, client.fingerprint, raw.(map[string]any)["client-fingerprint"])
		}
		if client.fingerprint == "Safari26_3" {
			require.Len(t, body, 1)
		} else {
			require.Len(t, body["rules"], 872)
			require.Len(t, body["proxy-groups"], 13)
		}
	}
	publicNodes := profile["proxies"].([]any)
	for i, flag := range []string{"🇺🇸", "🇺🇸", "🇯🇵"} {
		require.True(t, strings.HasPrefix(publicNodes[i].(map[string]any)["name"].(string), flag+" "), "flags must follow the landing, even when paths share a Hong Kong entry")
	}
	for _, format := range []string{"vless", "singbox"} {
		response := request(t, router, "GET", "/api/subscribe/"+sub.UUID+"?format="+format, "", "", 200)
		var names []string
		if format == "vless" {
			for _, line := range strings.Split(response.Body.String(), "\n") {
				names = append(names, strings.SplitN(line, "#", 2)[1])
			}
		} else {
			var config struct{ Outbounds []struct{ Tag string } }
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &config))
			for _, outbound := range config.Outbounds {
				names = append(names, outbound.Tag)
			}
		}
		require.Len(t, names, 3)
		for i, flag := range []string{"🇺🇸", "🇺🇸", "🇯🇵"} {
			require.True(t, strings.HasPrefix(names[i], flag+" "), format+" must use landing flags")
		}
	}
	for _, format := range []string{"vless", "clash_meta", "singbox"} {
		for _, account := range gateway {
			response := request(t, router, "GET", "/api/accounts/"+account.ID+"/subscribe?type="+format, "", auth.Token, 200)
			flag := "🇺🇸"
			if account.CombinationID == comboB {
				flag = "🇯🇵"
			}
			require.Contains(t, response.Body.String(), flag+" ")
			require.NotContains(t, response.Body.String(), "🇭🇰 ")
			require.Contains(t, response.Body.String(), entry.IP, "country display must not change the relay address")
		}
	}
	require.Equal(t, first.IP, publicNodes[0].(map[string]any)["server"], "subscription ordering must preserve the direct node followed by selected combinations")
	require.Contains(t, publicNodes[1].(map[string]any)["name"], "A到a")
	require.Contains(t, w.Body.String(), "192.0.2.1")
	require.Contains(t, w.Body.String(), "192.0.2.10")
	require.NotContains(t, w.Body.String(), upstream.UUID)
	// Internal transports are not customer accounts and cannot be edited manually.
	customerList := request(t, router, "GET", "/api/servers/"+first.ID+"/accounts", "", auth.Token, 200)
	require.NotContains(t, customerList.Body.String(), connectionA.UUID)
	jsonRequest("PUT", "/api/accounts/"+upstream.ID, map[string]any{"enabled": false}, 409)
	require.Contains(t, w.Body.String(), "A到a")
	require.Contains(t, w.Body.String(), "A到b")
	names := map[any]bool{}
	for _, p := range profile["proxies"].([]any) {
		names[p.(map[string]any)["name"]] = true
	}
	require.Len(t, names, 3)
	require.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
	request(t, router, "GET", "/api/accounts/"+upstream.ID+"/subscribe", "", auth.Token, 409)
	jsonRequest("POST", "/api/subscriptions/"+sub.ID+"/accounts", model.AccountMapping{ServerID: upstream.ServerID, AccountID: upstream.ID}, 400)
	jsonRequest("PUT", "/api/subscriptions/"+sub.ID, map[string]any{"account_mappings": []model.AccountMapping{{ServerID: upstream.ServerID, AccountID: upstream.ID}}}, 400)
	request(t, router, "DELETE", "/api/accounts/"+upstream.ID, "", auth.Token, 409)
	jsonRequest("PUT", "/api/accounts/"+gateway[0].ID, map[string]any{"uuid": uuid.NewString()}, 400)
	jsonRequest("PUT", "/api/relay-combinations/"+comboA, model.CombinationRequest{Name: "A到a", RelayServerID: entryB.ID, ExitServerID: first.ID}, 400)
	request(t, router, "DELETE", "/api/relay-combinations/"+comboA, "", auth.Token, 409)
	unused := createCombination("B到b", entryB.ID, second.ID)
	unusedUUID := connectionFor(unused).UUID
	jsonRequest("PUT", "/api/relay-combinations/"+unused, model.CombinationRequest{Name: "未使用的组合", RelayServerID: entryB.ID, ExitServerID: second.ID}, 200)
	require.Equal(t, unusedUUID, connectionFor(unused).UUID, "renaming must preserve the transport credential")
	third := createNode("出口c", "192.0.2.40", "exit", true)
	jsonRequest("PUT", "/api/relay-combinations/"+unused, model.CombinationRequest{Name: "未使用的组合", RelayServerID: entryB.ID, ExitServerID: third.ID}, 200)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE uuid=$1`, unusedUUID).Scan(&count))
	require.Zero(t, count, "switching an unused combination retires its old transport")
	newConnection := connectionFor(unused)
	require.Equal(t, third.IP, newConnection.Address)
	require.NotEqual(t, unusedUUID, newConnection.UUID)
	unusedUUID = newConnection.UUID
	request(t, router, "DELETE", "/api/relay-combinations/"+unused, "", auth.Token, 200)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE uuid=$1`, unusedUUID).Scan(&count))
	require.Zero(t, count, "deleting an unused combination removes its dedicated credential")
	// Both batch creation and bulk adding nodes reuse the same transactional provisioning path.
	batch := func(req model.BatchSubscriptionRequest) []model.BatchSubscriptionResult {
		w := jsonRequest("POST", "/api/subscriptions/batch", req, 200)
		var response struct {
			Results []model.BatchSubscriptionResult
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		return response.Results
	}
	viaB := batch(model.BatchSubscriptionRequest{Names: []string{"B路用户"}, CombinationID: comboC})
	require.Equal(t, "created", viaB[0].Status)
	viaBAccounts, e := repository.NewSubscriptionAccountRepository(db.DB).ListBySubscriptionID(viaB[0].ID)
	require.NoError(t, e)
	require.Len(t, viaBAccounts, 1)
	require.Equal(t, entryB.ID, viaBAccounts[0].ServerID)
	require.Equal(t, comboC, viaBAccounts[0].CombinationID)
	created := batch(model.BatchSubscriptionRequest{Names: []string{"用户甲", "用户乙"}, CombinationID: comboA})
	ids := []string{}
	uuids := map[string]bool{}
	for _, item := range created {
		require.Equal(t, "created", item.Status)
		ids = append(ids, item.ID)
		list, e := repository.NewSubscriptionAccountRepository(db.DB).ListBySubscriptionID(item.ID)
		require.NoError(t, e)
		require.Len(t, list, 1)
		require.Equal(t, comboA, list[0].CombinationID)
		var exitUUID string
		require.NoError(t, db.QueryRow(`SELECT e.uuid FROM relay_routes r JOIN accounts e ON e.id=r.exit_account_id WHERE r.account_id=$1`, list[0].ID).Scan(&exitUUID))
		require.Equal(t, connectionA.UUID, exitUUID, "customers share only their combination's upstream credential")
		uuids[list[0].UUID] = true
	}
	require.Len(t, uuids, 2)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE transport`).Scan(&count))
	require.Equal(t, 3, count, "batch provisioning must not duplicate the upstream transport")
	for _, item := range batch(model.BatchSubscriptionRequest{SubscriptionIDs: ids, CombinationID: comboB}) {
		require.Equal(t, "added", item.Status)
	}
	for _, item := range batch(model.BatchSubscriptionRequest{SubscriptionIDs: ids, CombinationID: comboB}) {
		require.Equal(t, "skipped", item.Status)
	}
	for _, item := range batch(model.BatchSubscriptionRequest{Names: []string{"用户丙", "用户丁"}, ServerID: second.ID}) {
		require.Equal(t, "created", item.Status)
	}
	for _, item := range batch(model.BatchSubscriptionRequest{Names: []string{"用户甲", "用户乙"}, CombinationID: comboA}) {
		require.Equal(t, "failed", item.Status)
	}
	// The pre-existing unadopted direct path still exports immediately.
	old := batch(model.BatchSubscriptionRequest{Names: []string{"直连用户"}, ServerID: legacy.ID})
	require.Equal(t, "created", old[0].Status)
	oldSub, e := repository.NewSubscriptionRepository(db.DB).GetByID(old[0].ID)
	require.NoError(t, e)
	request(t, router, "GET", "/api/subscribe/"+oldSub.UUID, "", "", 200)
	bad := batch(model.BatchSubscriptionRequest{Names: []string{"不应留下半份订阅"}, ServerID: entry.ID})
	require.Equal(t, "failed", bad[0].Status)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE name='不应留下半份订阅'`).Scan(&count))
	require.Zero(t, count)
	partial := batch(model.BatchSubscriptionRequest{Names: []string{"用户甲", "用户戊"}, ServerID: first.ID})
	require.Equal(t, "failed", partial[0].Status)
	require.Equal(t, "created", partial[1].Status)
	jsonRequest("POST", "/api/subscriptions/batch", model.BatchSubscriptionRequest{Names: []string{"重名", "重名"}, ServerID: first.ID}, 400)
	jsonRequest("POST", "/api/subscriptions/batch", model.BatchSubscriptionRequest{Names: []string{"冲突目标"}, ServerID: first.ID, CombinationID: comboA}, 400)
	jsonRequest("POST", "/api/subscriptions/"+sub.ID+"/accounts", model.AccountMapping{CombinationID: comboB, AutoCreate: true}, 200)
	w = request(t, router, "GET", "/api/subscriptions/"+sub.ID, "", auth.Token, 200)
	var updated model.SubscriptionWithAccounts
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	require.Len(t, updated.Accounts, 3, "retry must not duplicate an existing target")
	w = request(t, router, "GET", "/api/subscriptions/full", "", auth.Token, 200)
	require.Contains(t, w.Body.String(), "combination_name")
	// Authenticated cumulative statistics retry safely across batches and core epochs.
	secret := "test-agent-token"
	digest := sha256.Sum256([]byte(secret))
	require.NoError(t, nodes.SetAgentTokenHash(entry.ID, hex.EncodeToString(digest[:])))
	key, e := accounts.StatsKey(gateway[0].ID)
	require.NoError(t, e)
	send := func(batch, epoch string, up, down int64, token string, want int) {
		body, _ := json.Marshal(model.HeartbeatRequest{ServerID: entry.ID, BatchID: batch, TrafficEpoch: epoch, V2rayStatus: "running", UserTrafficStats: []model.UserTrafficStat{{Email: key, Upload: up, Download: down}}})
		req := httptest.NewRequest("POST", "/api/agent/heartbeat", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-PSK", token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, want, rec.Code, rec.Body.String())
	}
	batchID := uuid.NewString()
	send(batchID, "core-1", 100, 200, "", http.StatusUnauthorized)
	send(batchID, "core-1", 100, 200, secret, 200)
	send(batchID, "core-1", 100, 200, secret, 200)
	send(uuid.NewString(), "core-1", 100, 200, secret, 200)
	send(uuid.NewString(), "core-1", 200, 400, secret, 200)
	send(uuid.NewString(), "core-2", 10, 20, secret, 200)
	a, e := accounts.GetByID(gateway[0].ID)
	require.NoError(t, e)
	require.Equal(t, int64(630), a.TrafficUsed)
	a, e = accounts.GetByID(upstream.ID)
	require.NoError(t, e)
	require.Zero(t, a.TrafficUsed)
	jsonRequest("PUT", "/api/accounts/"+gateway[0].ID, map[string]any{"enabled": false}, 200)
	w = request(t, router, "GET", "/api/subscribe/"+sub.UUID+"?format=clash_meta", "", "", 200)
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &profile))
	require.Len(t, profile["proxies"], 2)
	// The account-management style batch flow removes exactly the selected path.
	for _, item := range batch(model.BatchSubscriptionRequest{Action: "remove", SubscriptionIDs: ids, CombinationID: comboA}) {
		require.Equal(t, "removed", item.Status)
	}
	for _, id := range ids {
		list, e := repository.NewSubscriptionAccountRepository(db.DB).ListBySubscriptionID(id)
		require.NoError(t, e)
		require.Len(t, list, 1)
		require.Equal(t, comboB, list[0].CombinationID, "another combination on the same relay must remain")
	}
	for _, item := range batch(model.BatchSubscriptionRequest{Action: "remove", SubscriptionIDs: ids, CombinationID: comboA}) {
		require.Equal(t, "skipped", item.Status, "removal retry must be idempotent")
	}
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE transport`).Scan(&count))
	require.Equal(t, 3, count, "customer removal must preserve automatic combination transports")
	for _, item := range batch(model.BatchSubscriptionRequest{Action: "add", SubscriptionIDs: ids, CombinationID: comboA}) {
		require.Equal(t, "added", item.Status)
	}
	var direct *model.Account
	for _, a := range sub.Accounts {
		if a.CombinationID == "" {
			direct = &a.Account
		}
	}
	require.NotNil(t, direct)
	w = jsonRequest("POST", "/api/subscriptions", model.CreateSubscriptionRequest{Name: "共享直连引用", AccountMappings: []model.AccountMapping{{ServerID: first.ID, AccountID: direct.ID}}}, 201)
	var shared model.SubscriptionWithAccounts
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &shared))
	require.Equal(t, "removed", batch(model.BatchSubscriptionRequest{Action: "remove", SubscriptionIDs: []string{sub.ID}, ServerID: first.ID})[0].Status)
	_, e = accounts.GetByID(direct.ID)
	require.NoError(t, e, "an account still used by another subscription must remain")
	remaining, e := repository.NewSubscriptionAccountRepository(db.DB).ListBySubscriptionID(sub.ID)
	require.NoError(t, e)
	require.Len(t, remaining, 2, "removing the direct exit must preserve both relay paths")
	require.Equal(t, "removed", batch(model.BatchSubscriptionRequest{Action: "remove", SubscriptionIDs: []string{shared.ID}, ServerID: first.ID})[0].Status)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE id=$1`, direct.ID).Scan(&count))
	require.Zero(t, count, "the final removal cleans up the unused customer credential")
	jsonRequest("POST", "/api/subscriptions/batch", model.BatchSubscriptionRequest{Action: "unknown", SubscriptionIDs: ids, CombinationID: comboA}, 400)
	jsonRequest("POST", "/api/subscriptions/batch", model.BatchSubscriptionRequest{Action: "remove", Names: []string{"不能新建"}, CombinationID: comboA}, 400)
	t.Run("rename adopted relay with imported accounts", func(t *testing.T) {
		jsonRequest := func(method, path string, body any, want int) *httptest.ResponseRecorder {
			encoded, err := json.Marshal(body)
			require.NoError(t, err)
			return request(t, router, method, path, string(encoded), auth.Token, want)
		}
		imported := createNode("已导入中转", "192.0.2.40", "relay", true)
		account, err := accounts.Upsert(&model.CreateAccountRequest{ServerID: imported.ID, UUID: uuid.NewString(), Email: "existing-relay-user", Protocols: []string{"vless_reality_vision"}})
		require.NoError(t, err)
		require.Empty(t, account.CombinationID, "imported relay users have no manually created chain yet")
		_, err = db.Exec(`UPDATE servers SET ssh_password='retained-test-password' WHERE id=$1`, imported.ID)
		require.NoError(t, err)
		w := jsonRequest("PUT", "/api/servers/"+imported.ID, map[string]any{"name": "香港中转改名", "role": "relay", "ip": imported.IP, "ssh_port": 9, "ssh_user": "root", "ssh_key_type": "password"}, 200)
		var renamed model.Server
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &renamed))
		require.Equal(t, "香港中转改名", renamed.Name)
		require.Equal(t, "relay", renamed.Role)
		require.Equal(t, "adopted", renamed.Node.State)
		stored, err := nodes.GetByIDForInstall(imported.ID)
		require.NoError(t, err)
		require.Equal(t, "retained-test-password", stored.SSHPassword)
		_, err = accounts.GetByID(account.ID)
		require.NoError(t, err)
		// A real role change must still reject managed nodes and existing direct accounts.
		jsonRequest("PUT", "/api/servers/"+imported.ID, map[string]any{"role": "exit", "name": "不应保存"}, 500)
		stored, err = nodes.GetByID(imported.ID)
		require.NoError(t, err)
		require.Equal(t, "香港中转改名", stored.Name)
		jsonRequest("PUT", "/api/servers/"+legacy.ID, map[string]any{"role": "relay"}, 500)
		jsonRequest("PUT", "/api/servers/"+first.ID, map[string]any{"name": "落地改名", "role": "exit"}, 200)
	})
	t.Run("SSH management jump preserves node addresses and rejects cycles", func(t *testing.T) {
		w := jsonRequest("PUT", "/api/servers/"+legacy.ID, map[string]any{"ssh_jump_server_id": entry.ID}, 200)
		var changed model.Server
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &changed))
		require.Equal(t, entry.ID, changed.SSHJumpServerID)
		require.Equal(t, legacy.IP, changed.IP)
		stored, err := nodes.GetByIDForInstall(legacy.ID)
		require.NoError(t, err)
		require.Equal(t, entry.ID, stored.SSHJumpServerID)
		jsonRequest("PUT", "/api/servers/"+entry.ID, map[string]any{"ssh_jump_server_id": entry.ID}, 500)
		jsonRequest("PUT", "/api/servers/"+legacy.ID, map[string]any{"ssh_jump_server_id": first.ID}, 500)
		jsonRequest("PUT", "/api/servers/"+entry.ID, map[string]any{"ssh_jump_server_id": entryB.ID}, 500)
		jsonRequest("PUT", "/api/servers/"+entryB.ID, map[string]any{"ssh_jump_server_id": entry.ID}, 200)
		jsonRequest("PUT", "/api/servers/"+legacy.ID, map[string]any{"ssh_jump_server_id": entryB.ID}, 500)
		jsonRequest("PUT", "/api/servers/"+legacy.ID, map[string]any{"ssh_jump_server_id": ""}, 200)
		stored, err = nodes.GetByID(legacy.ID)
		require.NoError(t, err)
		require.Empty(t, stored.SSHJumpServerID, "clearing the jump restores direct SSH")
		jsonRequest("POST", "/api/servers", map[string]any{"name": "跳板管理的新落地", "ip": "192.0.2.60", "country_code": "US", "role": "exit", "ssh_jump_server_id": entry.ID}, 201)
	})
	t.Run("export flag follows applied landing until a new route is published", func(t *testing.T) {
		account, err := accounts.Create(&model.CreateAccountRequest{ServerID: entry.ID, Email: "legacy-route", Protocols: []string{"vless_reality_vision"}})
		require.NoError(t, err)
		us, err := accounts.GetByUUID(connectionA.UUID)
		require.NoError(t, err)
		jp, err := accounts.GetByUUID(connectionFor(comboB).UUID)
		require.NoError(t, err)
		require.NoError(t, nodes.BindRoute(account.ID, us.ID))
		require.NoError(t, nodes.MarkRoutesApplied([]*model.RelayRoute{{AccountID: account.ID, ExitAccountID: us.ID, Revision: 1}}))
		require.NoError(t, nodes.SavePublishedAccounts([]*model.Account{account}, map[string]bool{account.ID: true}))
		require.NoError(t, nodes.BindRoute(account.ID, jp.ID))
		// An unknown stored country still uses the landing name, never the relay country.
		country, name := "", "US landing"
		_, err = nodes.Update(first.ID, &model.UpdateServerRequest{CountryCode: &country, Name: &name})
		require.NoError(t, err)
		svc := service.NewAccountService(accounts, nodes)
		ready, err := svc.ReadyAccounts([]*model.Account{account})
		require.NoError(t, err)
		require.Len(t, ready, 1)
		require.Equal(t, "🇺🇸", ready[0].ExportFlag)
		require.NoError(t, nodes.MarkRoutesApplied([]*model.RelayRoute{{AccountID: account.ID, ExitAccountID: jp.ID, Revision: 2}}))
		ready, err = svc.ReadyAccounts([]*model.Account{account})
		require.NoError(t, err)
		require.Equal(t, "🇯🇵", ready[0].ExportFlag)
	})
}
