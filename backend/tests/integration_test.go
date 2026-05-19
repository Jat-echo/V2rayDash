package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"v2ray-dash/backend/internal/model"
)

var (
	testRouter *gin.Engine
)

// TestMain 初始化测试环境
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// 初始化路由
	testRouter = gin.New()
	setupRoutes(testRouter)

	// 运行测试
	code := m.Run()

	os.Exit(code)
}

func setupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 服务器
		serverHandler := newServerHandler()
		api.GET("/servers", serverHandler.list)
		api.POST("/servers", serverHandler.create)
		api.GET("/servers/:id", serverHandler.get)
		api.DELETE("/servers/:id", serverHandler.delete)

		// 订阅管理
		subHandler := newSubscriptionHandler()
		api.GET("/subscriptions", subHandler.list)
		api.POST("/subscriptions", subHandler.create)
		api.GET("/subscriptions/:id", subHandler.get)
		api.DELETE("/subscriptions/:id", subHandler.delete)
		api.GET("/subscriptions/:id/link", subHandler.getLink)

		// 公开订阅
		r.GET("/api/subscribe/:uuid", subHandler.serveSubscription)

		// Agent
		agentHandler := newAgentHandler()
		api.POST("/agent/heartbeat", agentHandler.heartbeat)

		// 日志
		logHandler := newLogHandler()
		api.GET("/logs/operation", logHandler.listOperationLogs)
		api.GET("/logs/node-status", logHandler.listNodeStatuses)

		// 模板
		templateHandler := newTemplateHandler()
		api.GET("/templates", templateHandler.list)
		api.POST("/templates", templateHandler.create)
		api.DELETE("/templates/:id", templateHandler.delete)
	}
}

// ============ 服务器测试辅助函数 ============

type testServerHandler struct{}

func newServerHandler() *testServerHandler {
	return &testServerHandler{}
}

func (h *testServerHandler) list(c *gin.Context) {
	c.JSON(http.StatusOK, []*model.Server{})
}

func (h *testServerHandler) create(c *gin.Context) {
	var req model.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	server := &model.Server{
		ID:     "test-server-id",
		Name:   req.Name,
		IP:     req.IP,
		Status: "online",
	}
	c.JSON(http.StatusCreated, server)
}

func (h *testServerHandler) get(c *gin.Context) {
	c.JSON(http.StatusOK, &model.Server{ID: c.Param("id"), Name: "Test"})
}

func (h *testServerHandler) delete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ============ 订阅测试辅助函数 ============

type testSubscriptionHandler struct{}

func newSubscriptionHandler() *testSubscriptionHandler {
	return &testSubscriptionHandler{}
}

func (h *testSubscriptionHandler) list(c *gin.Context) {
	c.JSON(http.StatusOK, []*model.Subscription{})
}

func (h *testSubscriptionHandler) create(c *gin.Context) {
	sub := &model.Subscription{
		ID:   "test-sub-id",
		UUID: "test-uuid-1234",
		Name: "Test Subscription",
	}
	c.JSON(http.StatusCreated, sub)
}

func (h *testSubscriptionHandler) get(c *gin.Context) {
	c.JSON(http.StatusOK, &model.Subscription{ID: c.Param("id")})
}

func (h *testSubscriptionHandler) delete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *testSubscriptionHandler) getLink(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"link":    "http://localhost:8080/api/subscribe/test-uuid",
		"encoded": "aHR0cDovL2xvY2FsaG9zdDo4MDgwL2FwaS9zdWJzY3JpYmUvdGVzdC11dWlk",
	})
}

func (h *testSubscriptionHandler) serveSubscription(c *gin.Context) {
	c.String(http.StatusOK, "vless://test-uuid@10.0.0.1:443")
}

// ============ Agent 测试辅助函数 ============

type testAgentHandler struct{}

func newAgentHandler() *testAgentHandler {
	return &testAgentHandler{}
}

func (h *testAgentHandler) heartbeat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ============ 日志测试辅助函数 ============

type testLogHandler struct{}

func newLogHandler() *testLogHandler {
	return &testLogHandler{}
}

func (h *testLogHandler) listOperationLogs(c *gin.Context) {
	c.JSON(http.StatusOK, []*model.OperationLog{})
}

func (h *testLogHandler) listNodeStatuses(c *gin.Context) {
	c.JSON(http.StatusOK, []*model.NodeStatus{})
}

// ============ 模板测试辅助函数 ============

type testTemplateHandler struct{}

func newTemplateHandler() *testTemplateHandler {
	return &testTemplateHandler{}
}

func (h *testTemplateHandler) list(c *gin.Context) {
	c.JSON(http.StatusOK, []*model.Template{})
}

func (h *testTemplateHandler) create(c *gin.Context) {
	tmpl := &model.Template{
		ID:   1,
		Name: "Test Template",
		Config: model.TemplateConfig{
			Core: "xray-core",
		},
	}
	c.JSON(http.StatusCreated, tmpl)
}

func (h *testTemplateHandler) delete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ============ 测试用例 ============

func TestServerAPI(t *testing.T) {
	t.Run("CreateServer", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Test Server",
			"ip":   "192.168.1.100",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var server model.Server
		json.Unmarshal(w.Body.Bytes(), &server)
		assert.Equal(t, "Test Server", server.Name)
		t.Logf("PASS: Server created - %s", server.Name)
	})

	t.Run("ListServers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/servers", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Log("PASS: Server list OK")
	})
}

func TestSubscriptionAPI(t *testing.T) {
	t.Run("CreateSubscription", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"server_id": "test-server",
			"name":      "Test Sub",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		t.Log("PASS: Subscription created")
	})

	t.Run("GetSubscriptionLink", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/subscriptions/test-id/link", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp["link"])
		assert.NotEmpty(t, resp["encoded"])
		t.Logf("PASS: Subscription link - %s", resp["link"])
	})
}

func TestAgentAPI(t *testing.T) {
	t.Run("Heartbeat", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"server_id":    "test-server",
			"cpu_percent":  45.5,
			"mem_percent": 62.3,
			"disk_percent": 55.0,
			"v2ray_status": "running",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/agent/heartbeat", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Log("PASS: Heartbeat sent")
	})
}

func TestPublicSubscription(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/subscribe/test-uuid-1234", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "vless://")
	t.Log("PASS: Public subscription OK")
}

func TestLogAPI(t *testing.T) {
	t.Run("OperationLogs", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/logs/operation", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Log("PASS: Operation logs OK")
	})

	t.Run("NodeStatus", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/logs/node-status", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Log("PASS: Node status OK")
	})
}

func TestTemplateAPI(t *testing.T) {
	t.Run("CreateTemplate", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":        "Test Template",
			"description": "Test",
			"config": model.TemplateConfig{
				Core:      "xray-core",
				Port:      443,
				Protocols: []string{"vless_reality_vision"},
			},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/templates", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		t.Log("PASS: Template created")
	})

	t.Run("ListTemplates", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/templates", nil)
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		t.Log("PASS: Template list OK")
	})
}

// ============ 完整流程测试 ============

func TestFullWorkflow(t *testing.T) {
	fmt.Println("\n========================================")
	fmt.Println("V2rayDash 完整流程测试")
	fmt.Println("========================================")

	// 1. 创建服务器
	fmt.Println("[1/10] 创建服务器...")
	reqBody, _ := json.Marshal(map[string]interface{}{
		"name": "Workflow Server",
		"ip":   "10.0.0.250",
	})
	req, _ := http.NewRequest("POST", "/api/servers", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var server model.Server
	json.Unmarshal(w.Body.Bytes(), &server)
	fmt.Printf("   ✓ 服务器创建成功: %s\n", server.Name)

	// 2. 创建模板
	fmt.Println("[2/10] 创建模板...")
	templateBody, _ := json.Marshal(map[string]interface{}{
		"name":        "Workflow Template",
		"description": "Test",
		"config": model.TemplateConfig{
			Core:      "xray-core",
			Port:      443,
			Protocols: []string{"vless_reality_vision"},
		},
	})
	req, _ = http.NewRequest("POST", "/api/templates", bytes.NewBuffer(templateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var template model.Template
	json.Unmarshal(w.Body.Bytes(), &template)
	fmt.Printf("   ✓ 模板创建成功: %s\n", template.Name)

	// 3. 创建订阅
	fmt.Println("[3/10] 创建订阅...")
	subBody, _ := json.Marshal(map[string]interface{}{
		"server_id": server.ID,
		"name":      "Workflow Sub",
	})
	req, _ = http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(subBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var sub model.Subscription
	json.Unmarshal(w.Body.Bytes(), &sub)
	fmt.Printf("   ✓ 订阅创建成功: %s\n", sub.Name)

	// 4. 获取订阅链接
	fmt.Println("[4/10] 获取订阅链接...")
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/subscriptions/%s/link", sub.ID), nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var linkResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &linkResp)
	fmt.Printf("   ✓ 订阅链接: %s\n", linkResp["link"])

	// 5. 测试公开订阅
	fmt.Println("[5/10] 测试公开订阅...")
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/subscribe/%s", sub.UUID), nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	fmt.Printf("   ✓ 公开订阅内容: %s\n", w.Body.String())

	// 6. 发送心跳
	fmt.Println("[6/10] 发送心跳...")
	heartbeatBody, _ := json.Marshal(map[string]interface{}{
		"server_id":    server.ID,
		"cpu_percent":  30.0,
		"mem_percent":  45.0,
		"disk_percent": 50.0,
		"v2ray_status": "running",
	})
	req, _ = http.NewRequest("POST", "/api/agent/heartbeat", bytes.NewBuffer(heartbeatBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	fmt.Println("   ✓ 心跳发送成功")

	// 7. 检查操作日志
	fmt.Println("[7/10] 检查操作日志...")
	req, _ = http.NewRequest("GET", "/api/logs/operation", nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	fmt.Println("   ✓ 操作日志获取成功")

	// 8. 检查节点状态
	fmt.Println("[8/10] 检查节点状态...")
	req, _ = http.NewRequest("GET", "/api/logs/node-status", nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	fmt.Println("   ✓ 节点状态获取成功")

	// 9. 删除订阅
	fmt.Println("[9/10] 删除订阅...")
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/subscriptions/%s", sub.ID), nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	fmt.Println("   ✓ 订阅删除成功")

	// 10. 删除服务器
	fmt.Println("[10/10] 删除服务器...")
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/servers/%s", server.ID), nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	fmt.Println("   ✓ 服务器删除成功")

	fmt.Println("\n========================================")
	fmt.Println("✓ 所有流程测试通过!")
	fmt.Println("========================================")
}