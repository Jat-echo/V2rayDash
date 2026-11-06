package service

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unicode/utf8"
	"v2ray-dash/backend/internal/model"
)

func managedFixture(t *testing.T) (map[string]any, *model.ManagedNode, string) {
	t.Helper()
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	private := base64.RawURLEncoding.EncodeToString(key.Bytes())
	source := map[string]any{
		"log": map[string]any{"loglevel": "warning"}, "api": map[string]any{"tag": "api", "services": []any{"StatsService"}}, "stats": map[string]any{},
		"inbounds":  []any{map[string]any{"tag": "entry", "listen": "0.0.0.0", "port": float64(443), "protocol": "vless", "settings": map[string]any{"clients": []any{}, "decryption": "none"}, "streamSettings": map[string]any{"network": "tcp", "security": "reality", "realitySettings": map[string]any{"target": "academy.nvidia.com:443", "serverNames": []any{"academy.nvidia.com"}, "privateKey": private, "shortIds": []any{"1234abcd"}}}}, map[string]any{"tag": "api-in", "listen": "127.0.0.1", "port": 10085, "protocol": "dokodemo-door", "settings": map[string]any{"address": "127.0.0.1"}}},
		"outbounds": []any{map[string]any{"tag": "blocked", "protocol": "blackhole"}},
		"routing":   map[string]any{"rules": []any{map[string]any{"type": "field", "inboundTag": []any{"api-in"}, "outboundTag": "api"}}},
	}
	return source, &model.ManagedNode{Role: "relay", Layout: "relay", InboundTag: "entry", ShortID: "1234abcd"}, base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
}

func TestManagedConfigRoutesAndRealXrayValidation(t *testing.T) {
	source, node, pub := managedFixture(t)
	before, _ := json.Marshal(source)
	u1, u2 := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	up1, up2 := "33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444"
	exits := map[string]managedExit{u1: {UUID: up1, Server: &model.Server{IP: "192.0.2.10", RealityPort: 443, RealityServerName: "academy.nvidia.com", RealityPublicKey: pub, Node: &model.ManagedNode{ShortID: "5678"}}}, u2: {UUID: up2, Server: &model.Server{IP: "192.0.2.20", RealityPort: 14143, RealityServerName: "academy.nvidia.com", RealityPublicKey: pub, Node: &model.ManagedNode{ShortID: "abcd"}}}}
	users := []managedUser{{ID: u1, Email: "first"}, {ID: u2, Email: "second"}}
	config, err := buildManagedConfig(source, node, users, exits)
	require.NoError(t, err)
	encoded, _ := json.Marshal(config)
	after, _ := json.Marshal(source)
	require.Equal(t, string(before), string(after), "builder must not mutate its source")
	out := array(config["outbounds"])
	require.Equal(t, "blackhole", object(out[0])["protocol"])
	require.Len(t, out, 3)
	for _, v := range out {
		require.NotEqual(t, "freedom", object(v)["protocol"])
	}
	for i, pair := range []struct{ user, uuid, host string }{{"first", up1, "192.0.2.10"}, {"second", up2, "192.0.2.20"}} {
		rule := object(array(object(config["routing"])["rules"])[i+1])
		require.Equal(t, pair.user, textValue(array(rule["user"])[0]))
		peer := object(array(object(object(out[i+1])["settings"])["vnext"])[0])
		require.Equal(t, pair.host, peer["address"])
		require.Equal(t, pair.uuid, object(array(peer["users"])[0])["id"])
	}
	shared, err := buildManagedConfig(source, node, users, map[string]managedExit{u1: exits[u1], u2: exits[u1]})
	require.NoError(t, err)
	require.Len(t, array(shared["outbounds"]), 2, "one blackhole and one shared combination transport")
	sharedRules := array(object(shared["routing"])["rules"])
	require.Equal(t, object(sharedRules[1])["outboundTag"], object(sharedRules[2])["outboundTag"])
	require.Len(t, array(object(object(array(shared["inbounds"])[0])["settings"])["clients"]), 2, "customers keep independent ingress credentials")
	_, err = buildManagedConfig(source, node, users, map[string]managedExit{})
	require.Error(t, err)
	empty, err := buildManagedConfig(source, node, nil, nil)
	require.NoError(t, err)
	require.Empty(t, array(object(object(array(empty["inbounds"])[0])["settings"])["clients"]))
	t.Run("official core accepts generated config", func(t *testing.T) {
		binary := os.Getenv("XRAY_TEST_BINARY")
		if binary == "" {
			t.Skip("set XRAY_TEST_BINARY to validate with the actual core")
		}
		path := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(path, encoded, 0600))
		output, err := exec.Command(binary, "run", "-test", "-config", path).CombinedOutput()
		require.NoError(t, err, string(output))
	})
}

func TestManagedConfigAcceptsUniqueUntaggedRealityInbound(t *testing.T) {
	source, node, _ := managedFixture(t)
	delete(object(array(source["inbounds"])[0]), "tag")
	node.InboundTag, node.Role, node.Layout = "", "exit", "legacy"
	tls := map[string]any{"protocol": "vless", "streamSettings": map[string]any{"security": "tls"}, "settings": map[string]any{"clients": []any{map[string]any{"id": "existing-tls-user"}}}}
	source["inbounds"] = append(array(source["inbounds"]), tls)
	config, err := buildManagedConfig(source, node, []managedUser{{ID: "11111111-1111-4111-8111-111111111111", Email: "reality-user"}}, nil)
	require.NoError(t, err)
	inbounds := array(config["inbounds"])
	require.Equal(t, "reality-user", object(array(object(object(inbounds[0])["settings"])["clients"])[0])["email"])
	require.Equal(t, tls, inbounds[len(inbounds)-1], "untagged TLS users must not be replaced")
	source["inbounds"] = append(array(source["inbounds"]), array(source["inbounds"])[0])
	_, err = buildManagedConfig(source, node, nil, nil)
	require.Error(t, err, "multiple matching Reality inbounds must be rejected")
}

func TestReadRealityPreservesExistingAcceptedParameters(t *testing.T) {
	source, _, _ := managedFixture(t)
	inbound := object(array(source["inbounds"])[0])
	settings := object(object(inbound["streamSettings"])["realitySettings"])
	settings["shortIds"] = []any{"", "6ba85179e30d4fc2"}
	settings["serverNames"] = []any{"www.example.com", "academy.nvidia.com"}
	reality, err := realityFromInbound(inbound, &model.Server{RealityServerName: "academy.nvidia.com"})
	require.NoError(t, err)
	require.Equal(t, "6ba85179e30d4fc2", reality.ShortID)
	require.Equal(t, "academy.nvidia.com", reality.ServerName)
	reality, err = realityFromInbound(inbound, &model.Server{Node: &model.ManagedNode{ShortID: ""}})
	require.NoError(t, err)
	require.Empty(t, reality.ShortID, "an existing accepted empty Short ID is preserved")
}

func TestManagedSubscriptionsKeepTemplateAndRealityParameters(t *testing.T) {
	s := NewAccountService(nil, nil)
	account := &model.Account{ID: "one", ServerID: "entry", UUID: "11111111-1111-4111-8111-111111111111", Email: "中文用户的美国出口线路一号", Enabled: true}
	server := &model.Server{ID: "entry", Name: "中转入口", IP: "192.0.2.1", CountryCode: "US"}
	reality := &RealityConfig{Enabled: true, Port: 443, ServerName: "academy.nvidia.com", PublicKey: "public-key", ShortID: "abcdef1234", SupportMLKEM: true}
	content, err := s.GenerateClashMetaSubscriptionMulti([]*model.Account{account}, map[string]*model.Server{"entry": server}, map[string]*RealityConfig{"entry": reality})
	require.NoError(t, err)
	var config map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content), &config))
	proxies := array(config["proxies"])
	require.Len(t, proxies, 1)
	proxy := object(proxies[0])
	require.True(t, utf8.ValidString(textValue(proxy["name"])))
	require.Equal(t, "192.0.2.1", proxy["server"])
	require.Equal(t, "abcdef1234", object(proxy["reality-opts"])["short-id"])
	require.Equal(t, "chrome", proxy["client-fingerprint"])
	require.Equal(t, true, object(proxy["reality-opts"])["support-x25519mlkem768"])
	require.Len(t, array(config["proxy-groups"]), 13)
	require.Len(t, array(config["rules"]), 872)
	require.Equal(t, true, object(config["dns"])["enable"])
	require.Contains(t, s.GetAccountLink(account, server.IP, "vless", reality, "test"), "sid=abcdef1234")
	shadowrocket, err := s.GenerateShadowrocketSubscriptionMulti([]*model.Account{account}, map[string]*model.Server{"entry": server}, map[string]*RealityConfig{"entry": reality})
	require.NoError(t, err)
	var ios map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(shadowrocket), &ios))
	require.Len(t, ios, 1, "a node subscription must not overwrite phone routing or DNS")
	iosProxies := array(ios["proxies"])
	require.Len(t, iosProxies, 1)
	iosProxy := object(iosProxies[0])
	require.Equal(t, "Safari26_3", iosProxy["client-fingerprint"])
	require.NotContains(t, object(iosProxy["reality-opts"]), "support-x25519mlkem768")
	// All authentication, transport, naming and routing parameters remain identical.
	iosProxy["client-fingerprint"] = "chrome"
	object(iosProxy["reality-opts"])["support-x25519mlkem768"] = true
	require.Equal(t, proxy, iosProxy)
	account.Enabled = false
	empty, err := s.GenerateShadowrocketSubscriptionMulti([]*model.Account{account}, map[string]*model.Server{"entry": server}, nil)
	require.NoError(t, err)
	require.Equal(t, "proxies: []\n", empty)
	reality.ShortID = ""
	reality.ShortIDConfigured = true
	require.Equal(t, "", shortID(reality), "an explicitly empty short ID must not use the legacy default")
}
