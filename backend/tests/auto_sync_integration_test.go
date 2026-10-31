package tests

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/service"
	"v2ray-dash/backend/pkg/database"
)

func autoSyncDB(t *testing.T) *database.DB {
	t.Helper()
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("set TEST_DATABASE_URL for isolated auto-sync tests")
	}
	db, err := database.NewPostgres(connection)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	schema := pq.QuoteIdentifier("auto_" + strings.ReplaceAll(uuid.NewString(), "-", ""))
	_, err = db.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.Exec("DROP SCHEMA " + schema + " CASCADE")
		require.NoError(t, err)
		require.NoError(t, db.Close())
	})
	_, err = db.Exec("SET search_path TO " + schema)
	require.NoError(t, err)
	require.NoError(t, database.InitSchema(db))
	return db
}

func TestAutomaticSyncQueueAndMutationGuards(t *testing.T) {
	db := autoSyncDB(t)
	nodes := repository.NewServerRepository(db.DB)
	accounts := repository.NewAccountRepository(db.DB)
	subs := repository.NewSubscriptionRepository(db.DB)
	links := repository.NewSubscriptionAccountRepository(db.DB)
	chains := repository.NewCombinationRepository(db.DB)
	create := func(role string) *model.Server {
		n, err := nodes.Create(&model.CreateServerRequest{Name: role, Role: role, IP: "127.0.0.1", CountryCode: "US", SSHPort: 9, SSHUser: "root", SSHKeyType: "password"})
		require.NoError(t, err)
		require.NoError(t, nodes.SaveNode(&model.ManagedNode{ServerID: n.ID, Role: role, Layout: role, InboundTag: "entry", ConfigHash: "fixture"}))
		return n
	}
	relay, exit := create("relay"), create("exit")
	chain, err := chains.Save("", &model.CombinationRequest{Name: "path", RelayServerID: relay.ID, ExitServerID: exit.ID})
	require.NoError(t, err)
	state := func(id string) string { n, err := nodes.GetNode(id); require.NoError(t, err); return n.State }
	require.Equal(t, "pending", state(exit.ID), "a new transport queues its landing even without customers")
	upstreams, err := accounts.ListByServerID(exit.ID)
	require.NoError(t, err)
	enabled := map[string]bool{}
	for _, a := range upstreams {
		enabled[a.ID] = true
	}
	require.NoError(t, nodes.FinalizePublish(exit.ID, "fixture", nil, upstreams, enabled))
	createSub := func(name string) string {
		id, _, err := subs.CreateWithTargets(&model.CreateSubscriptionRequest{Name: name, AccountMappings: []model.AccountMapping{{CombinationID: chain, AutoCreate: true}}})
		require.NoError(t, err)
		return id
	}
	sid := createSub("one")
	createSub("two")
	createSub("three")
	require.Equal(t, "pending", state(relay.ID))
	require.Equal(t, "active", state(exit.ID), "new users reuse a published upstream without restarting the landing")
	jobs, err := nodes.PendingNodes(5 * time.Second)
	require.NoError(t, err)
	require.Empty(t, jobs)
	_, err = db.Exec(`UPDATE managed_nodes SET updated_at=CURRENT_TIMESTAMP-interval '10 seconds' WHERE server_id=$1`, relay.ID)
	require.NoError(t, err)
	jobs, err = nodes.PendingNodes(5 * time.Second)
	require.NoError(t, err)
	require.Equal(t, []string{relay.ID}, jobs, "multiple mutations coalesce to one job")
	claimed, err := nodes.BeginAutoPublish(relay.ID, 5*time.Second)
	require.NoError(t, err)
	require.True(t, claimed)
	claimed, err = nodes.BeginAutoPublish(relay.ID, 0)
	require.NoError(t, err)
	require.False(t, claimed, "another publisher cannot claim an in-flight node")
	list, err := links.ListBySubscriptionID(sid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	account := list[0]
	disabled := false
	require.Error(t, accounts.Update(account.ID, &model.UpdateAccountRequest{Enabled: &disabled}))
	require.Error(t, accounts.Delete(account.ID))
	_, err = accounts.Create(&model.CreateAccountRequest{ServerID: relay.ID, Email: "blocked", Protocols: []string{"vless_reality_vision"}})
	require.Error(t, err)
	require.Error(t, subs.Update(sid, &model.UpdateSubscriptionRequest{Enable: &disabled}))
	require.Error(t, subs.Delete(sid))
	require.Error(t, links.RemoveAccount(sid, account.ID))
	_, err = subs.ReplaceTargets(sid, nil)
	require.Error(t, err)
	stored, err := accounts.GetByID(account.ID)
	require.NoError(t, err)
	require.True(t, stored.Enabled)
	list, err = links.ListBySubscriptionID(sid)
	require.NoError(t, err)
	require.Len(t, list, 1, "rejected mutations must roll back")
	// Display-only edits neither restart nor clear a running job.
	name, country := "renamed", "HK"
	_, err = nodes.Update(relay.ID, &model.UpdateServerRequest{Name: &name, CountryCode: &country})
	require.NoError(t, err)
	_, err = chains.Save(chain, &model.CombinationRequest{Name: name, RelayServerID: relay.ID, ExitServerID: exit.ID})
	require.NoError(t, err)
	require.NoError(t, subs.Update(sid, &model.UpdateSubscriptionRequest{Name: &name}))
	require.Equal(t, "publishing", state(relay.ID))
	require.NoError(t, nodes.MarkNodePending(relay.ID))
	require.Equal(t, "publishing", state(relay.ID), "a late request must not overwrite an active claim")
	require.NoError(t, nodes.SetNodeState(relay.ID, "active", "", ""))
	require.NoError(t, accounts.Update(account.ID, &model.UpdateAccountRequest{Email: &name}))
	require.Equal(t, "active", state(relay.ID), "stable stats keys make account labels display-only")
	require.NoError(t, accounts.Update(account.ID, &model.UpdateAccountRequest{Enabled: &disabled}))
	require.Equal(t, "pending", state(relay.ID))
	require.NoError(t, nodes.SetNodeState(relay.ID, "active", "", ""))
	require.NoError(t, subs.Update(sid, &model.UpdateSubscriptionRequest{Enable: &disabled}))
	require.Equal(t, "pending", state(relay.ID))
	require.NoError(t, nodes.SetNodeState(relay.ID, "active", "", ""))
	require.NoError(t, links.RemoveAccount(sid, account.ID))
	require.Equal(t, "pending", state(relay.ID))
	_, err = accounts.GetByID(account.ID)
	require.NoError(t, err, "unlinking retains the existing account as before")
	require.NoError(t, nodes.SetNodeState(relay.ID, "active", "", ""))
	_, err = subs.ReplaceTargets(sid, []model.AccountMapping{{CombinationID: chain, AutoCreate: true}})
	require.NoError(t, err)
	require.NoError(t, nodes.SetNodeState(relay.ID, "active", "", ""))
	_, err = subs.ReplaceTargets(sid, nil)
	require.NoError(t, err)
	require.Equal(t, "pending", state(relay.ID), "removed paths also queue their old nodes")
	require.NoError(t, nodes.SetNodeState(relay.ID, "active", "", ""))
	require.NoError(t, subs.Delete(sid))
	// Fresh dependencies defer the relay, and eligible landings sort before relays.
	require.NoError(t, nodes.MarkNodePending(exit.ID))
	require.NoError(t, nodes.MarkNodePending(relay.ID))
	_, err = db.Exec(`UPDATE managed_nodes SET updated_at=CURRENT_TIMESTAMP-interval '10 seconds' WHERE server_id=$1`, relay.ID)
	require.NoError(t, err)
	claimed, err = nodes.BeginAutoPublish(relay.ID, 5*time.Second)
	require.NoError(t, err)
	require.False(t, claimed)
	_, err = db.Exec(`UPDATE managed_nodes SET updated_at=CURRENT_TIMESTAMP-interval '10 seconds'`)
	require.NoError(t, err)
	jobs, err = nodes.PendingNodes(5 * time.Second)
	require.NoError(t, err)
	require.Equal(t, []string{exit.ID, relay.ID}, jobs)
}

func TestAutoSyncWorkerRecoversPendingAndStopsAfterFailure(t *testing.T) {
	db := autoSyncDB(t)
	nodes := repository.NewServerRepository(db.DB)
	accounts := repository.NewAccountRepository(db.DB)
	n, err := nodes.Create(&model.CreateServerRequest{Name: "offline", Role: "exit", IP: "127.0.0.1", CountryCode: "US", SSHPort: 9, SSHUser: "root", SSHKeyType: "password"})
	require.NoError(t, err)
	require.NoError(t, nodes.SaveNode(&model.ManagedNode{ServerID: n.ID, Role: "exit", Layout: "exit", ConfigHash: "unchanged"}))
	a, err := accounts.Create(&model.CreateAccountRequest{ServerID: n.ID, Email: "one", Protocols: []string{"vless_reality_vision"}})
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE servers SET country_code=NULL,ssh_key=NULL,ssh_password=NULL WHERE id=$1`, n.ID)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		service.RunNodeAutoSync(ctx, db.DB, 20*time.Millisecond, 5*time.Millisecond)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("worker did not stop")
		}
	})
	count := func(action string) int {
		var count int
		err := db.QueryRow(`SELECT count(*) FROM operation_logs WHERE action=$1`, action).Scan(&count)
		require.NoError(t, err)
		return count
	}
	require.Eventually(t, func() bool { return count("auto_sync_node") == 1 }, 2*time.Second, 10*time.Millisecond)
	state, err := nodes.GetNode(n.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", state.State)
	require.NotEmpty(t, state.LastError)
	require.Contains(t, state.LastError, "country_code", "an error before SSH must not leave the claim stuck publishing")
	require.Equal(t, "unchanged", state.ConfigHash)
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 1, count("auto_sync_node"), "failed work must not retry on every polling tick")
	disabled := false
	_, err = db.Exec(`UPDATE servers SET country_code='US' WHERE id=$1`, n.ID)
	require.NoError(t, err)
	stored, err := nodes.GetByIDForInstall(n.ID)
	require.NoError(t, err, "nullable legacy authentication fields are read as empty credentials")
	require.Empty(t, stored.SSHKey)
	require.Empty(t, stored.SSHPassword)
	require.NoError(t, accounts.Update(a.ID, &model.UpdateAccountRequest{Enabled: &disabled}))
	require.Eventually(t, func() bool { return count("auto_sync_node") == 2 }, 2*time.Second, 10*time.Millisecond)
	// Manual retry remains available and the structured error is visible through the real API.
	router := newRouter(db)
	w := request(t, router, "POST", "/api/auth/setup", `{"username":"sync-admin","password":"sync-test-password"}`, "", 200)
	var auth struct{ Token string }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &auth))
	request(t, router, "POST", "/api/servers/"+n.ID+"/publish", `{}`, auth.Token, 202)
	require.Eventually(t, func() bool { return count("publish_node") == 1 }, 2*time.Second, 10*time.Millisecond)
	w = request(t, router, "GET", "/api/logs/operation", "", auth.Token, 200)
	require.Contains(t, w.Body.String(), "auto_sync_node")
	require.Contains(t, w.Body.String(), "error")
	logs := repository.NewLogRepository(db.DB)
	require.NoError(t, logs.Create(&model.OperationLog{Operator: "system", Action: "batch_test", Detail: map[string]any{"count": 3}}))
	_, err = logs.List(&model.OperationLogFilter{})
	require.NoError(t, err, "batch log rows without a target ID must remain readable")
}
