# V2ray 服务管理平台 - 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个 Web 管理平台，管理多台 v2ray 服务器，实现服务器管理、订阅账号管理、健康监控和日志系统。

**Architecture:** Go + Gin 后端 API，React + Ant Design Pro 前端，PostgreSQL 数据库，节点 Agent 通过 HTTP POST 上报状态。

**Tech Stack:** Go 1.21+, Gin, React 18, Ant Design Pro 5, PostgreSQL 15+, 原生 Go SSH (golang.org/x/crypto/ssh)

---

## Phase 1: 项目结构与数据库设计

### Task 1: 初始化项目结构

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `frontend/package.json`
- Create: `agent/go.mod`

- [ ] **Step 1: 创建 backend/go.mod**

```go
module v2ray-dash/backend

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.4.0
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.14.0
	github.com/golang-jwt/jwt/v5 v5.1.0
)
```

- [ ] **Step 2: 创建 backend/cmd/server/main.go**

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/handler"
	"v2ray-dash/backend/pkg/database"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	r := gin.Default()
	handler.SetupRoutes(r, db)

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

- [ ] **Step 3: 创建 frontend/package.json**

```json
{
  "name": "v2ray-dash-frontend",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "antd": "^5.12.0",
    "@ant-design/pro-components": "^2.6.0",
    "axios": "^1.6.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "@vitejs/plugin-react": "^4.2.0",
    "typescript": "^5.3.0",
    "vite": "^5.0.0"
  }
}
```

- [ ] **Step 4: 创建 agent/go.mod**

```go
module v2ray-dash/agent

go 1.21

require (
	github.com/google/uuid v1.4.0
)
```

- [ ] **Step 5: Commit**

```bash
git add backend/go.mod backend/cmd/server/main.go frontend/package.json agent/go.mod
git commit -m "chore: initialize project structure"
```

---

### Task 2: 数据库设计与初始化

**Files:**
- Create: `backend/pkg/database/postgres.go`
- Create: `backend/pkg/database/schema.sql`

- [ ] **Step 1: 创建 backend/pkg/database/postgres.go**

```go
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewPostgres(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}
```

- [ ] **Step 2: 创建 backend/pkg/database/schema.sql**

```sql
-- 服务器表
CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    ssh_port INTEGER DEFAULT 22,
    ssh_user VARCHAR(50) DEFAULT 'root',
    ssh_key TEXT,
    tags JSONB DEFAULT '[]',
    status VARCHAR(20) DEFAULT 'unknown',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 订阅账号表
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    uuid VARCHAR(36) NOT NULL UNIQUE,
    enable BOOLEAN DEFAULT true,
    traffic_limit BIGINT DEFAULT 0,
    traffic_used BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(30),
    target_id UUID,
    detail JSONB,
    ip VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 节点状态表
CREATE TABLE IF NOT EXISTS node_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    cpu_percent FLOAT,
    memory_percent FLOAT,
    disk_percent FLOAT,
    bandwidth_in BIGINT DEFAULT 0,
    bandwidth_out BIGINT DEFAULT 0,
    v2ray_status VARCHAR(20) DEFAULT 'unknown',
    reported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_server_id ON subscriptions(server_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created_at ON operation_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_node_status_server_id ON node_status(server_id);
CREATE INDEX IF NOT EXISTS idx_node_status_reported_at ON node_status(reported_at);
```

- [ ] **Step 3: 创建 backend/pkg/database/init.go**

```go
package database

import (
	"database/sql"
	"os"
)

func InitSchema(db *DB) error {
	schema, err := os.ReadFile("pkg/database/schema.sql")
	if err != nil {
		return err
	}

	_, err = db.Exec(string(schema))
	return err
}
```

- [ ] **Step 4: 测试数据库连接**

Run: `cd backend && go mod tidy && go run cmd/server/main.go`
Expected: Server starts and connects to PostgreSQL

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/database/
git commit -m "feat: add database layer with schema"
```

---

### Task 3: 配置管理

**Files:**
- Create: `backend/internal/config/config.go`

- [ ] **Step 1: 创建 backend/internal/config/config.go**

```go
package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort   string
	DatabaseURL  string
	JWTSecret    string
	ControlCenterURL string
}

func Load() *Config {
	return &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://localhost:5432/v2ray_dash?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production"),
		ControlCenterURL: getEnv("CONTROL_CENTER_URL", "http://localhost:8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) Validate() error {
	if c.JWTSecret == "" || c.JWTSecret == "change-me-in-production" {
		return fmt.Errorf("JWT_SECRET must be set")
	}
	return nil
}
```

- [ ] **Step 2: 更新 main.go 使用配置**

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"v2ray-dash/backend/internal/config"
	"v2ray-dash/backend/internal/handler"
	"v2ray-dash/backend/pkg/database"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config invalid: %v", err)
	}

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	r := gin.Default()
	handler.SetupRoutes(r, db, cfg)

	go func() {
		log.Println("Server starting on :" + cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/config/config.go backend/cmd/server/main.go
git commit -m "feat: add configuration management"
```

---

## Phase 2: Go 后端 API 开发

### Task 4: 数据模型定义

**Files:**
- Create: `backend/internal/model/server.go`
- Create: `backend/internal/model/subscription.go`
- Create: `backend/internal/model/operation_log.go`
- Create: `backend/internal/model/node_status.go`

- [ ] **Step 1: 创建 backend/internal/model/server.go**

```go
package model

import (
	"time"
)

type Server struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	SSHPort   int       `json:"ssh_port"`
	SSHUser   string    `json:"ssh_user"`
	SSHKey    string    `json:"-"` // 敏感字段不暴露
	Tags      []string  `json:"tags"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateServerRequest struct {
	Name    string   `json:"name" binding:"required"`
	IP      string   `json:"ip" binding:"required"`
	SSHPort int      `json:"ssh_port"`
	SSHUser string   `json:"ssh_user"`
	SSHKey  string   `json:"ssh_key"`
	Tags    []string `json:"tags"`
}

type UpdateServerRequest struct {
	Name    *string  `json:"name"`
	SSHPort *int     `json:"ssh_port"`
	SSHUser *string  `json:"ssh_user"`
	SSHKey  *string  `json:"ssh_key"`
	Tags    []string `json:"tags"`
}
```

- [ ] **Step 2: 创建 backend/internal/model/subscription.go**

```go
package model

import (
	"time"
)

type Subscription struct {
	ID           string    `json:"id"`
	ServerID     string    `json:"server_id"`
	Name         string    `json:"name"`
	UUID         string    `json:"uuid"`
	Enable       bool      `json:"enable"`
	TrafficLimit int64     `json:"traffic_limit"`
	TrafficUsed  int64     `json:"traffic_used"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateSubscriptionRequest struct {
	ServerID     string `json:"server_id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	TrafficLimit int64  `json:"traffic_limit"`
}

type UpdateSubscriptionRequest struct {
	Name         *string `json:"name"`
	Enable       *bool   `json:"enable"`
	TrafficLimit *int64  `json:"traffic_limit"`
}
```

- [ ] **Step 3: 创建 backend/internal/model/operation_log.go**

```go
package model

import (
	"time"
)

type OperationLog struct {
	ID         string    `json:"id"`
	Operator   string    `json:"operator"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Detail     map[string]any `json:"detail"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

type OperationLogFilter struct {
	StartTime *time.Time
	EndTime   *time.Time
	TargetType string
	Operator   string
}
```

- [ ] **Step 4: 创建 backend/internal/model/node_status.go**

```go
package model

import (
	"time"
)

type NodeStatus struct {
	ID           string    `json:"id"`
	ServerID     string    `json:"server_id"`
	CPUPercent   float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskPercent  float64   `json:"disk_percent"`
	BandwidthIn  int64     `json:"bandwidth_in"`
	BandwidthOut int64     `json:"bandwidth_out"`
	V2rayStatus  string    `json:"v2ray_status"`
	ReportedAt   time.Time `json:"reported_at"`
}

type HeartbeatRequest struct {
	ServerID    string  `json:"server_id" binding:"required"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
	DiskPercent float64 `json:"disk_percent"`
	BandwidthIn int64   `json:"bandwidth_in"`
	BandwidthOut int64  `json:"bandwidth_out"`
	V2rayStatus string  `json:"v2ray_status"`
}
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/model/
git commit -m "feat: add data models"
```

---

### Task 5: Repository 层实现

**Files:**
- Create: `backend/internal/repository/server.go`
- Create: `backend/internal/repository/subscription.go`
- Create: `backend/internal/repository/log.go`

- [ ] **Step 1: 创建 backend/internal/repository/server.go**

```go
package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"v2ray-dash/backend/internal/model"
)

type ServerRepository struct {
	db *sql.DB
}

func NewServerRepository(db *sql.DB) *ServerRepository {
	return &ServerRepository{db: db}
}

func (r *ServerRepository) Create(req *model.CreateServerRequest) (*model.Server, error) {
	tagsJSON, _ := json.Marshal(req.Tags)
	result := r.db.QueryRow(
		`INSERT INTO servers (name, ip, ssh_port, ssh_user, ssh_key, tags)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, ip, ssh_port, ssh_user, tags, status, created_at, updated_at`,
		req.Name, req.IP, req.SSHPort, req.SSHUser, req.SSHKey, tagsJSON,
	)

	var s model.Server
	var tagsBytes []byte
	err := result.Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &tagsBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	return &s, nil
}

func (r *ServerRepository) GetByID(id string) (*model.Server, error) {
	var s model.Server
	var tagsBytes []byte
	err := r.db.QueryRow(
		`SELECT id, name, ip, ssh_port, ssh_user, tags, status, created_at, updated_at
		 FROM servers WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &tagsBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	return &s, nil
}

func (r *ServerRepository) List() ([]*model.Server, error) {
	rows, err := r.db.Query(
		`SELECT id, name, ip, ssh_port, ssh_user, tags, status, created_at, updated_at
		 FROM servers ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*model.Server
	for rows.Next() {
		var s model.Server
		var tagsBytes []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &tagsBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(tagsBytes, &s.Tags)
		servers = append(servers, &s)
	}
	return servers, nil
}

func (r *ServerRepository) Update(id string, req *model.UpdateServerRequest) (*model.Server, error) {
	// 实现动态更新逻辑
	return r.GetByID(id)
}

func (r *ServerRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM servers WHERE id = $1`, id)
	return err
}

func (r *ServerRepository) UpdateStatus(id, status string) error {
	_, err := r.db.Exec(
		`UPDATE servers SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id,
	)
	return err
}
```

- [ ] **Step 2: 创建 backend/internal/repository/subscription.go**

```go
package repository

import (
	"database/sql"
	"time"

	"v2ray-dash/backend/internal/model"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(req *model.CreateSubscriptionRequest) (*model.Subscription, error) {
	uuid := generateUUID()
	result := r.db.QueryRow(
		`INSERT INTO subscriptions (server_id, name, uuid, traffic_limit)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at`,
		req.ServerID, req.Name, uuid, req.TrafficLimit,
	)

	var s model.Subscription
	err := result.Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepository) GetByID(id string) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.QueryRow(
		`SELECT id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	return &s, err
}

func (r *SubscriptionRepository) List() ([]*model.Subscription, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	return subs, nil
}

func (r *SubscriptionRepository) ListByServerID(serverID string) ([]*model.Subscription, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions WHERE server_id = $1 ORDER BY created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	return subs, nil
}

func (r *SubscriptionRepository) Update(id string, req *model.UpdateSubscriptionRequest) error {
	// 实现动态更新
	_, err := r.db.Exec(`UPDATE subscriptions SET updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *SubscriptionRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM subscriptions WHERE id = $1`, id)
	return err
}

func generateUUID() string {
	// 使用 github.com/google/uuid
	return "placeholder-uuid"
}
```

- [ ] **Step 3: 创建 backend/internal/repository/log.go**

```go
package repository

import (
	"database/sql"
	"time"

	"v2ray-dash/backend/internal/model"
)

type LogRepository struct {
	db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) Create(log *model.OperationLog) error {
	_, err := r.db.Exec(
		`INSERT INTO operation_logs (operator, action, target_type, target_id, detail, ip)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		log.Operator, log.Action, log.TargetType, log.TargetID, log.Detail, log.IP,
	)
	return err
}

func (r *LogRepository) List(filter *model.OperationLogFilter) ([]*model.OperationLog, error) {
	query := `SELECT id, operator, action, target_type, target_id, detail, ip, created_at
		 FROM operation_logs WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter.StartTime != nil {
		query += ` AND created_at >= $` + string(rune('0'+argIdx))
		args = append(args, *filter.StartTime)
		argIdx++
	}
	// ... 其他过滤条件

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.OperationLog
	for rows.Next() {
		var l model.OperationLog
		if err := rows.Scan(&l.ID, &l.Operator, &l.Action, &l.TargetType, &l.TargetID, &l.Detail, &l.IP, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, nil
}

func (r *LogRepository) CreateNodeStatus(status *model.NodeStatus) error {
	_, err := r.db.Exec(
		`INSERT INTO node_status (server_id, cpu_percent, memory_percent, disk_percent, bandwidth_in, bandwidth_out, v2ray_status, reported_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		status.ServerID, status.CPUPercent, status.MemoryPercent, status.DiskPercent,
		status.BandwidthIn, status.BandwidthOut, status.V2rayStatus, time.Now(),
	)
	return err
}

func (r *LogRepository) GetLatestNodeStatus(serverID string) (*model.NodeStatus, error) {
	var s model.NodeStatus
	err := r.db.QueryRow(
		`SELECT id, server_id, cpu_percent, memory_percent, disk_percent, bandwidth_in, bandwidth_out, v2ray_status, reported_at
		 FROM node_status WHERE server_id = $1 ORDER BY reported_at DESC LIMIT 1`,
		serverID,
	).Scan(&s.ID, &s.ServerID, &s.CPUPercent, &s.MemoryPercent, &s.DiskPercent, &s.BandwidthIn, &s.BandwidthOut, &s.V2rayStatus, &s.ReportedAt)
	return &s, err
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/
git commit -m "feat: add repository layer"
```

---

### Task 6: HTTP Handlers 实现

**Files:**
- Create: `backend/internal/handler/server.go`
- Create: `backend/internal/handler/subscription.go`
- Create: `backend/internal/handler/agent.go`
- Create: `backend/internal/handler/log.go`
- Create: `backend/internal/handler/routes.go`

- [ ] **Step 1: 创建 backend/internal/handler/server.go**

```go
package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type ServerHandler struct {
	repo     *repository.ServerRepository
	logRepo  *repository.LogRepository
}

func NewServerHandler(db *sql.DB) *ServerHandler {
	return &ServerHandler{
		repo:    repository.NewServerRepository(db),
		logRepo: repository.NewLogRepository(db),
	}
}

func (h *ServerHandler) List(c *gin.Context) {
	servers, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

func (h *ServerHandler) Get(c *gin.Context) {
	id := c.Param("id")
	server, err := h.repo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req model.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.repo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录操作日志
	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "create_server",
		TargetType: "server",
		TargetID:   server.ID,
		Detail:     map[string]any{"name": server.Name, "ip": server.IP},
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusCreated, server)
}

func (h *ServerHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.repo.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "delete_server",
		TargetType: "server",
		TargetID:   id,
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
```

- [ ] **Step 2: 创建 backend/internal/handler/subscription.go**

```go
package handler

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type SubscriptionHandler struct {
	repo    *repository.SubscriptionRepository
	logRepo *repository.LogRepository
}

func NewSubscriptionHandler(db *sql.DB) *SubscriptionHandler {
	return &SubscriptionHandler{
		repo:    repository.NewSubscriptionRepository(db),
		logRepo: repository.NewLogRepository(db),
	}
}

func (h *SubscriptionHandler) List(c *gin.Context) {
	serverID := c.Query("server_id")
	var subs []*model.Subscription
	var err error

	if serverID != "" {
		subs, err = h.repo.ListByServerID(serverID)
	} else {
		subs, err = h.repo.List()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

func (h *SubscriptionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.repo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req model.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.repo.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "create_subscription",
		TargetType: "subscription",
		TargetID:   sub.ID,
		Detail:     map[string]any{"name": sub.Name, "server_id": sub.ServerID},
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusCreated, sub)
}

func (h *SubscriptionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *SubscriptionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logRepo.Create(&model.OperationLog{
		Operator:   "admin",
		Action:     "delete_subscription",
		TargetType: "subscription",
		TargetID:   id,
		IP:         c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetLink 生成订阅链接
func (h *SubscriptionHandler) GetLink(c *gin.Context) {
	id := c.Param("id")
	sub, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// 生成 Base64 编码的订阅链接
	link := fmt.Sprintf("https://your-domain.com/api/subscribe/%s", sub.UUID)
	encoded := base64.StdEncoding.EncodeToString([]byte(link))

	c.JSON(http.StatusOK, gin.H{
		"link":    link,
		"encoded": encoded,
	})
}
```

- [ ] **Step 3: 创建 backend/internal/handler/agent.go**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type AgentHandler struct {
	logRepo *repository.LogRepository
}

func NewAgentHandler(db interface{}) *AgentHandler {
	return &AgentHandler{}
}

func (h *AgentHandler) Heartbeat(c *gin.Context) {
	var req model.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新节点状态
	status := &model.NodeStatus{
		ServerID:      req.ServerID,
		CPUPercent:   req.CPUPercent,
		MemoryPercent: req.MemPercent,
		DiskPercent:  req.DiskPercent,
		BandwidthIn:  req.BandwidthIn,
		BandwidthOut: req.BandwidthOut,
		V2rayStatus:  req.V2rayStatus,
	}

	// 这里需要注入 LogRepository 来保存状态
	// h.logRepo.CreateNodeStatus(status)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AgentHandler) GetConfig(c *gin.Context) {
	serverID := c.Param("server_id")
	// 返回该服务器的最新配置
	c.JSON(http.StatusOK, gin.H{
		"server_id":      serverID,
		"control_center": "http://your-control-center:8080",
	})
}
```

- [ ] **Step 4: 创建 backend/internal/handler/log.go**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type LogHandler struct {
	repo *repository.LogRepository
}

func NewLogHandler(db interface{}) *LogHandler {
	return &LogHandler{}
}

func (h *LogHandler) ListOperationLogs(c *gin.Context) {
	filter := &model.OperationLogFilter{
		// 从 query 参数构建 filter
	}

	logs, err := h.repo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}
```

- [ ] **Step 5: 创建 backend/internal/handler/routes.go**

```go
package handler

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/config"
)

func SetupRoutes(r *gin.Engine, db *sql.DB, cfg *config.Config) {
	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	// API 路由组
	api := r.Group("/api")
	{
		// 服务器管理
		serverHandler := NewServerHandler(db)
		api.GET("/servers", serverHandler.List)
		api.POST("/servers", serverHandler.Create)
		api.GET("/servers/:id", serverHandler.Get)
		api.PUT("/servers/:id", serverHandler.Update)
		api.DELETE("/servers/:id", serverHandler.Delete)

		// 订阅管理
		subHandler := NewSubscriptionHandler(db)
		api.GET("/subscriptions", subHandler.List)
		api.POST("/subscriptions", subHandler.Create)
		api.GET("/subscriptions/:id", subHandler.Get)
		api.PUT("/subscriptions/:id", subHandler.Update)
		api.DELETE("/subscriptions/:id", subHandler.Delete)
		api.GET("/subscriptions/:id/link", subHandler.GetLink)

		// Agent 通信
		agentHandler := NewAgentHandler(db)
		api.POST("/agent/heartbeat", agentHandler.Heartbeat)
		api.GET("/agent/config/:server_id", agentHandler.GetConfig)

		// 日志
		logHandler := NewLogHandler(db)
		api.GET("/logs/operation", logHandler.ListOperationLogs)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/
git commit -m "feat: add HTTP handlers"
```

---

### Task 7: SSH 服务实现（远程命令执行）

**Files:**
- Create: `backend/internal/service/ssh.go`

- [ ] **Step 1: 创建 backend/internal/service/ssh.go**

```go
package service

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

type SSHService struct{}

func NewSSHService() *SSHService {
	return &SSHService{}
}

type SSHResult struct {
	Stdout  string
	Stderr  string
	ExitCode int
}

func (s *SSHService) Connect(host string, port int, user, privateKey string) (*ssh.Client, error) {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicAuth(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return conn, nil
}

func (s *SSHService) Execute(client *ssh.Client, command string) (*SSHResult, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}

	return &SSHResult{
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		ExitCode: exitCode,
	}, nil
}

func (s *SSHService) ExecuteWithPassword(host string, port int, user, password, command string) (*SSHResult, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}

	return &SSHResult{
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// ReadPrivateKeyFromFile 从文件读取私钥
func ReadPrivateKeyFromFile(path string) (string, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(keyBytes), nil
}

// GetLocalIP 获取本机 IP
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
```

- [ ] **Step 2: 添加执行 install.sh 的功能**

```go
func (s *SSHService) InstallV2ray(client *ssh.Client, controlCenterURL string) (*SSHResult, error) {
	installCmd := fmt.Sprintf(
		"curl -sL https://raw.githubusercontent.com/your-repo/install.sh | bash -s -- --agent %s",
		controlCenterURL,
	)
	return s.Execute(client, installCmd)
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/ssh.go
git commit -m "feat: add SSH service for remote command execution"
```

---

## Phase 3: React 前端开发

### Task 8: 前端项目初始化

**Files:**
- Create: `frontend/index.html`
- Create: `frontend/vite.config.ts`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`
- Create: `frontend/src/services/api.ts`
- Create: `frontend/src/pages/servers/index.tsx`
- Create: `frontend/src/pages/subscriptions/index.tsx`
- Create: `frontend/src/pages/monitor/index.tsx`
- Create: `frontend/src/pages/logs/index.tsx`

- [ ] **Step 1: 创建 frontend/index.html**

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>V2ray 管理平台</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 2: 创建 frontend/vite.config.ts**

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

- [ ] **Step 3: 创建 frontend/src/main.tsx**

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import 'antd/dist/reset.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
```

- [ ] **Step 4: 创建 frontend/src/App.tsx**

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from 'antd'
import ServerList from './pages/servers'
import SubscriptionList from './pages/subscriptions'
import Monitor from './pages/monitor'
import Logs from './pages/logs'

const { Header, Content } = Layout

function App() {
  return (
    <BrowserRouter>
      <Layout style={{ minHeight: '100vh' }}>
        <Header style={{ color: '#fff', fontSize: '18px', padding: '0 24px' }}>
          V2ray 管理平台
        </Header>
        <Content style={{ padding: '24px' }}>
          <Routes>
            <Route path="/" element={<Navigate to="/servers" replace />} />
            <Route path="/servers" element={<ServerList />} />
            <Route path="/subscriptions" element={<SubscriptionList />} />
            <Route path="/monitor" element={<Monitor />} />
            <Route path="/logs" element={<Logs />} />
          </Routes>
        </Content>
      </Layout>
    </BrowserRouter>
  )
}

export default App
```

- [ ] **Step 5: 创建 frontend/src/services/api.ts**

```typescript
import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

export interface Server {
  id: string
  name: string
  ip: string
  ssh_port: number
  ssh_user: string
  tags: string[]
  status: string
  created_at: string
  updated_at: string
}

export interface Subscription {
  id: string
  server_id: string
  name: string
  uuid: string
  enable: boolean
  traffic_limit: number
  traffic_used: number
  created_at: string
  updated_at: string
}

export interface OperationLog {
  id: string
  operator: string
  action: string
  target_type: string
  target_id: string
  detail: Record<string, any>
  ip: string
  created_at: string
}

export const serverAPI = {
  list: () => api.get<Server[]>('/servers').then(r => r.data),
  get: (id: string) => api.get<Server>(`/servers/${id}`).then(r => r.data),
  create: (data: Partial<Server>) => api.post<Server>('/servers', data).then(r => r.data),
  update: (id: string, data: Partial<Server>) => api.put(`/servers/${id}`, data),
  delete: (id: string) => api.delete(`/servers/${id}`),
}

export const subscriptionAPI = {
  list: (serverId?: string) => {
    const params = serverId ? { server_id: serverId } : {}
    return api.get<Subscription[]>('/subscriptions', { params }).then(r => r.data)
  },
  create: (data: Partial<Subscription>) => api.post<Subscription>('/subscriptions', data).then(r => r.data),
  delete: (id: string) => api.delete(`/subscriptions/${id}`),
  getLink: (id: string) => api.get<{ link: string; encoded: string }>(`/subscriptions/${id}/link`).then(r => r.data),
}

export const logAPI = {
  list: (params?: { start_time?: string; end_time?: string; target_type?: string }) =>
    api.get<OperationLog[]>('/logs/operation', { params }).then(r => r.data),
}
```

- [ ] **Step 6: 创建 frontend/src/pages/servers/index.tsx**

```tsx
import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, message } from 'antd'
import { serverAPI, Server } from '../../services/api'

export default function ServerList() {
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadServers()
  }, [])

  const loadServers = async () => {
    setLoading(true)
    try {
      const data = await serverAPI.list()
      setServers(data)
    } catch (e) {
      message.error('加载服务器列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = async (values: any) => {
    try {
      await serverAPI.create(values)
      message.success('添加成功')
      setModalVisible(false)
      form.resetFields()
      loadServers()
    } catch (e) {
      message.error('添加失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await serverAPI.delete(id)
      message.success('删除成功')
      loadServers()
    } catch (e) {
      message.error('删除失败')
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: 'IP', dataIndex: 'ip' },
    { title: 'SSH端口', dataIndex: 'ssh_port' },
    { title: '状态', dataIndex: 'status' },
    { title: '创建时间', dataIndex: 'created_at' },
    {
      title: '操作',
      render: (_: any, record: Server) => (
        <Space>
          <Button size="small">安装</Button>
          <Button size="small" danger onClick={() => handleDelete(record.id)}>删除</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setModalVisible(true)}>添加服务器</Button>
      </Space>

      <Table columns={columns} dataSource={servers} rowKey="id" loading={loading} />

      <Modal title="添加服务器" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null}>
        <Form form={form} onFinish={handleAdd} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="ip" label="IP地址" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="ssh_port" label="SSH端口" initialValue={22}>
            <Input type="number" />
          </Form.Item>
          <Form.Item name="ssh_user" label="SSH用户" initialValue="root">
            <Input />
          </Form.Item>
          <Form.Item name="ssh_key" label="SSH私钥">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 7: 创建 frontend/src/pages/subscriptions/index.tsx**

```tsx
import { useState, useEffect } from 'react'
import { Table, Button, Space, Modal, Form, Input, Select, message } from 'antd'
import { subscriptionAPI, serverAPI, Subscription, Server } from '../../services/api'

export default function SubscriptionList() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [subs, srvs] = await Promise.all([
        subscriptionAPI.list(),
        serverAPI.list(),
      ])
      setSubscriptions(subs)
      setServers(srvs)
    } catch (e) {
      message.error('加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = async (values: any) => {
    try {
      await subscriptionAPI.create(values)
      message.success('添加成功')
      setModalVisible(false)
      form.resetFields()
      loadData()
    } catch (e) {
      message.error('添加失败')
    }
  }

  const handleGetLink = async (id: string) => {
    try {
      const { link, encoded } = await subscriptionAPI.getLink(id)
      message.success(`订阅链接: ${link}`)
    } catch (e) {
      message.error('获取链接失败')
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name' },
    { title: 'UUID', dataIndex: 'uuid', render: (v: string) => v.slice(0, 8) + '...' },
    { title: '服务器', dataIndex: 'server_id', render: (id: string) => servers.find(s => s.id === id)?.name || id },
    { title: '流量限制', dataIndex: 'traffic_limit', render: (v: number) => v ? `${(v/1024**3).toFixed(1)} GB` : '无限' },
    { title: '已用流量', dataIndex: 'traffic_used', render: (v: number) => `${(v/1024**3).toFixed(1)} GB` },
    { title: '状态', dataIndex: 'enable', render: (v: boolean) => v ? '启用' : '禁用' },
    {
      title: '操作',
      render: (_: any, record: Subscription) => (
        <Space>
          <Button size="small" onClick={() => handleGetLink(record.id)}>链接</Button>
          <Button size="small" danger onClick={() => subscriptionAPI.delete(record.id).then(loadData)}>删除</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setModalVisible(true)}>添加账号</Button>
      </Space>

      <Table columns={columns} dataSource={subscriptions} rowKey="id" loading={loading} />

      <Modal title="添加账号" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null}>
        <Form form={form} onFinish={handleAdd} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="server_id" label="服务器" rules={[{ required: true }]}>
            <Select>
              {servers.map(s => <Select.Option key={s.id} value={s.id}>{s.name}</Select.Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="traffic_limit" label="流量限制(GB)">
            <Input type="number" />
          </Form.Item>
          <Button type="primary" htmlType="submit">提交</Button>
        </Form>
      </Modal>
    </div>
  )
}
```

- [ ] **Step 8: 创建 frontend/src/pages/monitor/index.tsx**

```tsx
import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Table, Tag } from 'antd'
import { serverAPI, Server } from '../../services/api'

interface NodeStatus {
  server_id: string
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  v2ray_status: string
  reported_at: string
}

export default function Monitor() {
  const [servers, setServers] = useState<Server[]>([])
  const [statuses, setStatuses] = useState<Map<string, NodeStatus>>(new Map())

  useEffect(() => {
    loadServers()
    const interval = setInterval(loadServers, 30000) // 30秒刷新
    return () => clearInterval(interval)
  }, [])

  const loadServers = async () => {
    const data = await serverAPI.list()
    setServers(data)
  }

  const getStatusTag = (status: string) => {
    const color = status === 'online' ? 'green' : status === 'offline' ? 'red' : 'default'
    return <Tag color={color}>{status}</Tag>
  }

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        {servers.map(server => (
          <Col span={6} key={server.id}>
            <Card title={server.name} extra={getStatusTag(server.status)}>
              <Statistic title="IP" value={server.ip} />
              <div style={{ marginTop: 16 }}>
                <div>SSH: {server.ssh_port}</div>
                <div>最后更新: {server.updated_at}</div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      <Card title="节点状态">
        <Table
          dataSource={Array.from(statuses.values())}
          rowKey="server_id"
          columns={[
            { title: '服务器', dataIndex: 'server_id', render: (id: string) => servers.find(s => s.id === id)?.name || id },
            { title: 'CPU', dataIndex: 'cpu_percent', render: (v: number) => `${v?.toFixed(1)}%` },
            { title: '内存', dataIndex: 'memory_percent', render: (v: number) => `${v?.toFixed(1)}%` },
            { title: '硬盘', dataIndex: 'disk_percent', render: (v: number) => `${v?.toFixed(1)}%` },
            { title: 'V2ray', dataIndex: 'v2ray_status' },
            { title: '上报时间', dataIndex: 'reported_at' },
          ]}
        />
      </Card>
    </div>
  )
}
```

- [ ] **Step 9: 创建 frontend/src/pages/logs/index.tsx**

```tsx
import { useState, useEffect } from 'react'
import { Table, Select, DatePicker, Space } from 'antd'
import { logAPI, OperationLog } from '../../services/api'

const { RangePicker } = DatePicker

export default function Logs() {
  const [logs, setLogs] = useState<OperationLog[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    loadLogs()
  }, [])

  const loadLogs = async () => {
    setLoading(true)
    try {
      const data = await logAPI.list()
      setLogs(data)
    } catch (e) {
      // handle error
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    { title: '时间', dataIndex: 'created_at' },
    { title: '操作人', dataIndex: 'operator' },
    { title: '动作', dataIndex: 'action' },
    { title: '目标类型', dataIndex: 'target_type' },
    { title: '目标ID', dataIndex: 'target_id', render: (v: string) => v?.slice(0, 8) || '-' },
    { title: 'IP', dataIndex: 'ip' },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <RangePicker />
        <Select placeholder="操作类型" style={{ width: 120 }} allowClear>
          <Select.Option value="create_server">创建服务器</Select.Option>
          <Select.Option value="delete_server">删除服务器</Select.Option>
          <Select.Option value="create_subscription">创建账号</Select.Option>
        </Select>
      </Space>

      <Table columns={columns} dataSource={logs} rowKey="id" loading={loading} />
    </div>
  )
}
```

- [ ] **Step 10: Commit**

```bash
git add frontend/
git commit -m "feat: add React frontend with Ant Design"
```

---

## Phase 4: Agent 开发

### Task 9: Agent 实现

**Files:**
- Create: `agent/cmd/agent/main.go`
- Create: `agent/internal/config/config.go`
- Create: `agent/internal/reporter/reporter.go`
- Create: `agent/internal/collector/collector.go`

- [ ] **Step 1: 创建 agent/cmd/agent/main.go**

```go
package main

import (
	"flag"
	"log"
	"time"

	"v2ray-dash/agent/internal/collector"
	"v2ray-dash/agent/internal/config"
	"v2ray-dash/agent/internal/reporter"
)

func main() {
	configPath := flag.String("config", "/etc/v2ray-agent/agent.json", "Agent config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Agent starting, server_id: %s, control_center: %s", cfg.ServerID, cfg.ControlCenterURL)

	reporterClient := reporter.New(cfg.ControlCenterURL, cfg.ServerID, cfg.PSK)
	col := collector.New()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 立即执行一次
	reportStatus(cfg.ServerID, reporterClient, col)

	for range ticker.C {
		reportStatus(cfg.ServerID, reporterClient, col)
	}
}

func reportStatus(serverID string, client *reporter.Client, col *collector.Collector) {
	status, err := col.Collect()
	if err != nil {
		log.Printf("Collect error: %v", err)
		return
	}

	status.ServerID = serverID

	if err := client.ReportStatus(status); err != nil {
		log.Printf("Report error: %v", err)
	}
}
```

- [ ] **Step 2: 创建 agent/internal/config/config.go**

```go
package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ServerID        string `json:"server_id"`
	ControlCenterURL string `json:"control_center_url"`
	PSK             string `json:"psk"` // Pre-shared key for authentication
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
```

- [ ] **Step 3: 创建 agent/internal/collector/collector.go**

```go
package collector

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"v2ray-dash/agent/internal/model"
)

type Collector struct{}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (*model.NodeStatus, error) {
	cpu, err := c.getCPUUsage()
	if err != nil {
		cpu = 0
	}

	mem, err := c.getMemoryUsage()
	if err != nil {
		mem = 0
	}

	disk, err := c.getDiskUsage()
	if err != nil {
		disk = 0
	}

	v2rayStatus := c.checkV2ray()

	return &model.NodeStatus{
		CPUPercent:    cpu,
		MemoryPercent: mem,
		DiskPercent:   disk,
		V2rayStatus:   v2rayStatus,
	}, nil
}

func (c *Collector) getCPUUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		return c.getLinuxCPU()
	}
	return 0, nil
}

func (c *Collector) getLinuxCPU() (float64, error) {
	cmd := exec.Command("top", "-bn1")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Cpu(s)") {
			// 解析 CPU 使用率
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "id," || p == "id" {
					if i > 0 {
						idle, _ := strconv.ParseFloat(strings.ReplaceAll(parts[i-1], ",", ""), 64)
						return 100 - idle, nil
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("could not parse CPU usage")
}

func (c *Collector) getMemoryUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		cmd := exec.Command("free", "-m")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 3 {
				total, _ := strconv.ParseFloat(fields[1], 64)
				used, _ := strconv.ParseFloat(fields[2], 64)
				if total > 0 {
					return (used / total) * 100, nil
				}
			}
		}
	}
	return 0, nil
}

func (c *Collector) getDiskUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		cmd := exec.Command("df", "-h", "/")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 5 {
				usage := strings.TrimSuffix(fields[4], "%")
				return strconv.ParseFloat(usage, 64)
			}
		}
	}
	return 0, nil
}

func (c *Collector) checkV2ray() string {
	if runtime.GOOS == "linux" {
		cmd := exec.Command("systemctl", "is-active", "v2ray")
		output, _ := cmd.Output()
		if strings.TrimSpace(string(output)) == "active" {
			return "running"
		}
	}
	return "stopped"
}
```

- [ ] **Step 4: 创建 agent/internal/reporter/reporter.go**

```go
package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"v2ray-dash/agent/internal/model"
)

type Client struct {
	serverID       string
	controlCenterURL string
	psk            string
	httpClient     *http.Client
}

func New(controlCenterURL, serverID, psk string) *Client {
	return &Client{
		serverID:        serverID,
		controlCenterURL: controlCenterURL,
		psk:             psk,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) ReportStatus(status *model.NodeStatus) error {
	url := fmt.Sprintf("%s/api/agent/heartbeat", c.controlCenterURL)

	body, err := json.Marshal(status)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PSK", c.psk)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] **Step 5: 创建 agent/internal/model/model.go**

```go
package model

type NodeStatus struct {
	ServerID       string  `json:"server_id"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	BandwidthIn   int64   `json:"bandwidth_in"`
	BandwidthOut  int64   `json:"bandwidth_out"`
	V2rayStatus   string  `json:"v2ray_status"`
}
```

- [ ] **Step 6: 更新 install.sh 添加 agent 安装选项（需要在 install.sh 中添加）**

需要修改现有的 install.sh，添加一个安装 agent 的选项。

```bash
# 在 install.sh 中找到合适的位置添加 agent 安装逻辑
install_agent() {
    local control_center_url="$1"
    local server_id="$2"
    local psk="$3"

    # 下载 agent 二进制
    curl -sL "${control_center_url}/agents/latest/linux_amd64/agent" -o /usr/local/bin/agent
    chmod +x /usr/local/bin/agent

    # 创建配置
    mkdir -p /etc/v2ray-agent
    cat > /etc/v2ray-agent/agent.json <<EOF
{
    "server_id": "${server_id}",
    "control_center_url": "${control_center_url}",
    "psk": "${psk}"
}
EOF

    # 创建 systemd 服务
    cat > /etc/systemd/system/v2ray-agent.service <<EOF
[Unit]
Description=V2ray Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/agent -config /etc/v2ray-agent/agent.json
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable v2ray-agent
    systemctl start v2ray-agent
}
```

- [ ] **Step 7: Commit**

```bash
git add agent/
git commit -m "feat: add node agent for status reporting"
```

---

## Phase 5: 集成与测试

### Task 10: 端到端测试

- [ ] **Step 1: 启动 PostgreSQL**

```bash
docker run -d --name v2ray-dash-db \
  -e POSTGRES_DB=v2ray_dash \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15
```

- [ ] **Step 2: 配置环境变量并启动后端**

```bash
cd backend
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/v2ray_dash?sslmode=disable"
export JWT_SECRET="your-secret-key-change-in-production"
go run cmd/server/main.go
```

Expected: Server starts on :8080

- [ ] **Step 3: 启动前端**

```bash
cd frontend
npm install
npm run dev
```

- [ ] **Step 4: 测试 API**

```bash
# 健康检查
curl http://localhost:8080/health

# 创建服务器
curl -X POST http://localhost:8080/api/servers \
  -H "Content-Type: application/json" \
  -d '{"name":"test-server","ip":"192.168.1.100","ssh_port":22,"ssh_user":"root"}'

# 获取服务器列表
curl http://localhost:8080/api/servers
```

- [ ] **Step 5: 测试前端界面**

打开浏览器访问 http://localhost:3000

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "test: add e2e test results"
```

---

## 实施检查清单

| Phase | Task | Description | Status |
|-------|------|-------------|--------|
| 1 | 1 | 项目结构初始化 | ☐ |
| 1 | 2 | 数据库设计与初始化 | ☐ |
| 1 | 3 | 配置管理 | ☐ |
| 2 | 4 | 数据模型定义 | ☐ |
| 2 | 5 | Repository 层实现 | ☐ |
| 2 | 6 | HTTP Handlers 实现 | ☐ |
| 2 | 7 | SSH 服务实现 | ☐ |
| 3 | 8 | 前端项目初始化 | ☐ |
| 4 | 9 | Agent 实现 | ☐ |
| 5 | 10 | 端到端测试 | ☐ |

---

**Plan saved to:** `docs/superpowers/plans/2026-05-12-v2ray-management-platform-plan.md`

**执行选项：**

**1. Subagent-Driven (推荐)** - 每个任务分配一个 subagent 执行，完成后 review，快速迭代

**2. Inline Execution** - 在当前 session 中执行任务，带 checkpoint 审核

选择哪个方式？