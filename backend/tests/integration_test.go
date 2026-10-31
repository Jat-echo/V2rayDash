package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/handler"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/pkg/database"
)

func newRouter(db *database.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.SetupRoutes(r, db, &config.Config{})
	return r
}

func request(t *testing.T, r http.Handler, method, path, body, token string, want int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, want, w.Code, "%s %s: %s", method, path, w.Body.String())
	return w
}

// Exercise production route registration, middleware and validation without a database.
func TestRoutes(t *testing.T) {
	r := newRouter(&database.DB{})
	w := request(t, r, "GET", "/health", "", "", http.StatusOK)
	require.JSONEq(t, `{"status":"ok"}`, w.Body.String())
	for _, path := range []string{"/api/servers", "/api/subscriptions", "/api/subscriptions/full", "/api/settings/public-url", "/api/logs/node-status"} {
		request(t, r, "GET", path, "", "", http.StatusUnauthorized)
	}
	request(t, r, "POST", "/api/servers/ping", "{}", "", http.StatusUnauthorized)
	request(t, r, "POST", "/api/auth/login", "{}", "", http.StatusBadRequest)
	request(t, r, "POST", "/api/agent/heartbeat", "{}", "", http.StatusBadRequest)
}

func TestAgentDownloadUsesConfiguredArtifact(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "linux-amd64-agent")
	require.NoError(t, os.WriteFile(artifact, []byte("test-agent-artifact"), 0600))
	t.Setenv("AGENT_BINARY_PATH", artifact)
	r := newRouter(&database.DB{})
	for _, path := range []string{"linux/amd64", "linux/x86_64", "linux_amd64", "linux_x86_64"} {
		w := request(t, r, "GET", "/agents/latest/"+path+"/agent", "", "", 200)
		require.Equal(t, "test-agent-artifact", w.Body.String())
	}
	request(t, r, "GET", "/agents/latest/linux_arm64/agent", "", "", 404)
	request(t, r, "GET", "/agents/latest/windows/amd64/agent", "", "", 404)
}

// Use an isolated schema; never reset or delete existing application tables.
func TestDatabaseWorkflow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to a PostgreSQL test database to run the real workflow")
	}
	db, err := database.NewPostgres(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	schema := pq.QuoteIdentifier("test_" + strings.ReplaceAll(uuid.NewString(), "-", ""))
	_, err = db.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.Exec("DROP SCHEMA " + schema + " CASCADE")
		require.NoError(t, err)
	})
	_, err = db.Exec("SET search_path TO " + schema)
	require.NoError(t, err)
	require.NoError(t, database.InitSchema(db))
	// Startup migrations must be safe to run again.
	require.NoError(t, database.InitSchema(db))
	r := newRouter(db)

	w := request(t, r, "GET", "/api/auth/status", "", "", http.StatusOK)
	require.JSONEq(t, `{"setup_required":true}`, w.Body.String())
	w = request(t, r, "POST", "/api/auth/setup", `{"username":"test-admin","password":"test-password"}`, "", http.StatusOK)
	var auth struct{ Token string }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &auth))
	require.NotEmpty(t, auth.Token)
	request(t, r, "POST", "/api/auth/setup", `{}`, "", http.StatusForbidden)
	request(t, r, "POST", "/api/auth/login", `{"username":"test-admin","password":"wrong-password"}`, "", http.StatusUnauthorized)

	w = request(t, r, "POST", "/api/servers", `{"name":"test-node","ip":"127.0.0.1","country_code":"US","ssh_port":22,"ssh_user":"root"}`, auth.Token, http.StatusCreated)
	var server model.Server
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &server))
	require.NotEmpty(t, server.ID)
	w = request(t, r, "GET", "/api/servers", "", auth.Token, http.StatusOK)
	var servers []model.Server
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &servers))
	require.Len(t, servers, 1)
	require.Equal(t, server.ID, servers[0].ID)
	request(t, r, "PUT", "/api/servers/"+server.ID, `{"name":"renamed-node"}`, auth.Token, http.StatusOK)

	// Seed one account directly: this workflow must never initiate an SSH install/sync.
	var accountID string
	err = db.QueryRow(`INSERT INTO accounts (server_id, uuid, email, protocols) VALUES ($1, $2, 'test@example.com', ARRAY['vless_tcp']) RETURNING id`, server.ID, uuid.NewString()).Scan(&accountID)
	require.NoError(t, err)
	body := `{"name":"test-subscription","remark":"integration","account_mappings":[{"server_id":"` + server.ID + `","account_id":"` + accountID + `"}]}`
	w = request(t, r, "POST", "/api/subscriptions", body, auth.Token, http.StatusCreated)
	var sub model.SubscriptionWithAccounts
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sub))
	require.NotEmpty(t, sub.ID)
	require.Equal(t, "integration", sub.Remark)
	require.Len(t, sub.Accounts, 1)
	request(t, r, "PUT", "/api/subscriptions/"+sub.ID+"/accounts/order", `{"order":[{"id":"`+accountID+`","sort_order":2}]}`, auth.Token, http.StatusOK)
	var sortOrder int
	require.NoError(t, db.QueryRow(`SELECT sort_order FROM subscription_accounts WHERE subscription_id = $1 AND account_id = $2`, sub.ID, accountID).Scan(&sortOrder))
	require.Equal(t, 2, sortOrder)
	request(t, r, "GET", "/api/subscriptions/full", "", auth.Token, http.StatusOK)
	w = request(t, r, "GET", "/api/subscriptions/"+sub.ID+"/link", "", auth.Token, http.StatusOK)
	require.Contains(t, w.Body.String(), "/api/subscribe/"+sub.UUID)
	request(t, r, "GET", "/api/subscribe/"+sub.UUID, "", "", http.StatusOK)

	body = `{"server_id":"` + server.ID + `","cpu_percent":25,"memory_percent":50,"user_traffic_stats":[{"email":"test@example.com","upload":100,"download":200}]}`
	request(t, r, "POST", "/api/agent/heartbeat", body, "", http.StatusOK)
	var used int64
	require.NoError(t, db.QueryRow(`SELECT traffic_used FROM subscriptions WHERE id = $1`, sub.ID).Scan(&used))
	require.EqualValues(t, 300, used)
	request(t, r, "GET", "/api/subscriptions/"+sub.ID+"/accounts/traffic", "", auth.Token, http.StatusOK)
	request(t, r, "GET", "/api/logs/node-status?server_id="+server.ID, "", auth.Token, http.StatusOK)
	request(t, r, "DELETE", "/api/subscriptions/"+sub.ID, "", auth.Token, http.StatusOK)
	request(t, r, "DELETE", "/api/servers/"+server.ID, "", auth.Token, http.StatusOK)
	request(t, r, "POST", "/api/auth/logout", "{}", auth.Token, http.StatusOK)
	request(t, r, "GET", "/api/servers", "", auth.Token, http.StatusUnauthorized)
}
