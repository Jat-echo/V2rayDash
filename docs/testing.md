# End-to-End Testing Guide

## Prerequisites

1. PostgreSQL 15+ running
2. Go 1.21+ installed
3. Node.js 18+ installed
4. npm or yarn for frontend

## Test Steps

### 1. Start PostgreSQL

```bash
docker run -d --name v2ray-dash-db \
  -e POSTGRES_DB=v2ray_dash \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15
```

### 2. Start Backend

```bash
cd backend
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/v2ray_dash?sslmode=disable"
export JWT_SECRET="your-secret-key-change-in-production"
go run cmd/server/main.go
```

Expected: Server starts on :8080

### 3. Start Frontend

```bash
cd frontend
npm install
npm run dev
```

Expected: Frontend starts on :3000

### 4. Test API

```bash
# Health check
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Create server
curl -X POST http://localhost:8080/api/servers \
  -H "Content-Type: application/json" \
  -d '{"name":"test-server","ip":"192.168.1.100","ssh_port":22,"ssh_user":"root"}'

# List servers
curl http://localhost:8080/api/servers
```

### 5. Test Frontend

Open browser to http://localhost:3000

Verify:
- Server list page loads
- Add server modal works
- Navigation between pages works

## Cleanup

```bash
docker stop v2ray-dash-db
docker rm v2ray-dash-db
```
