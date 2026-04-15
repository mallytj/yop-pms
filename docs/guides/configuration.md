# Configuration Guide

This document explains all environment variables and configuration options for Yop PMS.

## Table of Contents

- [Environment Variables](#environment-variables)
- [Development Setup](#development-setup)
- [Production Setup](#production-setup)
- [Secrets Management](#secrets-management)

---

## Environment Variables

### Required Variables

These must be set or the server will fail to start.

#### `DB_URL` (Required)

PostgreSQL connection string.

```
DB_URL=postgres://user:password@localhost:5433/yop_pms
DB_URL=postgres://user:password@db-prod.aws.com:5432/yop_pms?sslmode=require
```

**Format:** `postgres://[user[:password]@][netloc][:port][/dbname][?param1=value1&...]`

**Parameters:**
- `sslmode=disable` — No SSL (dev only)
- `sslmode=require` — Require SSL (prod)
- `sslmode=verify-full` — Verify SSL certificate

---

### Optional Variables

#### `PORT` (Default: `8080`)

HTTP server port.

```
PORT=8080        # Development
PORT=8080        # Production (behind reverse proxy)
```

#### `APP_ENV` (Default: `dev`)

Application environment. Controls logging level and sampling behavior.

```
APP_ENV=dev      # Debug logs, always sample traces
APP_ENV=prod     # Info logs, 10% sample traces
APP_ENV=staging  # Info logs, 10% sample traces
```

#### `REDIS_ADDR` (Default: `localhost:6379`)

Redis server address.

```
REDIS_ADDR=localhost:6379         # Development
REDIS_ADDR=redis-prod.aws.com:6379  # Production
REDIS_ADDR=redis-sentinel:26379   # Sentinel-managed Redis
```

#### `REDIS_PASSWORD` (Default: `""`)

Redis password (empty for no auth).

```
REDIS_PASSWORD=              # No auth
REDIS_PASSWORD=your-password # With auth
```

#### `ALLOWED_ORIGINS` (Default: `http://localhost:5173`)

Space-separated list of allowed CORS origins.

```
ALLOWED_ORIGINS=http://localhost:5173                           # Development
ALLOWED_ORIGINS=https://app.example.com https://admin.example.com  # Production
```

#### `OTLP_ENDPOINT` (Default: `""`)

OpenTelemetry collector endpoint. If empty, tracing is disabled (no-op exporter).

```
OTLP_ENDPOINT=              # Disabled (dev)
OTLP_ENDPOINT=http://localhost:4318  # Local Jaeger (HTTP)
OTLP_ENDPOINT=localhost:4317         # Datadog (gRPC)
```

#### `SERVICE_NAME` (Default: `yop-pms`)

Service name in OpenTelemetry traces.

```
SERVICE_NAME=yop-pms
SERVICE_NAME=yop-pms-prod
```

#### `SERVICE_VERSION` (Default: `0.1.0`)

Service version in OpenTelemetry traces.

```
SERVICE_VERSION=0.1.0
SERVICE_VERSION=1.0.0
SERVICE_VERSION=2024-03-15-abc123def  # Git commit hash
```

---

## Development Setup

### Using `.env` file

Create `.env` in the project root:

```bash
# Database
DB_URL=postgres://yop:password@localhost:5433/yop_pms?sslmode=disable

# Server
PORT=8080
APP_ENV=dev

# Cache
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# CORS
ALLOWED_ORIGINS=http://localhost:5173 http://localhost:3000

# Observability
OTLP_ENDPOINT=
SERVICE_NAME=yop-pms
SERVICE_VERSION=0.1.0-dev
```

### Start Services

```bash
# Start PostgreSQL, Redis, and other services
make docker-up

# Or just database services
make db-up

# Start Go server with hot-reload
make dev

# Frontend runs on http://localhost:5173
# Backend runs on http://localhost:8080
# Swagger UI: http://localhost:8080/swagger/index.html
```

### Testing with Local Tracer Collector

For testing OpenTelemetry without production infrastructure:

```bash
# Start Jaeger all-in-one container
docker run -d \
  -p 6831:6831/udp \
  -p 16686:16686 \
  jaegertracing/all-in-one

# Update .env
OTLP_ENDPOINT=http://localhost:14268/api/traces

# Visit Jaeger UI: http://localhost:16686
```

---

## Production Setup

### Environment Variables

Set via your deployment platform (Kubernetes, AWS, Docker, etc.).

```yaml
# Kubernetes ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: yop-config
data:
  APP_ENV: prod
  PORT: "8080"
  SERVICE_NAME: yop-pms-prod
  SERVICE_VERSION: "1.0.0"
```

```yaml
# Kubernetes Secret (for sensitive values)
apiVersion: v1
kind: Secret
metadata:
  name: yop-secrets
type: Opaque
stringData:
  DB_URL: postgres://user:password@db-prod.aws.com:5432/yop_pms?sslmode=require
  REDIS_PASSWORD: very-secure-password
```

### Example: Docker Compose

```yaml
version: "3.9"

services:
  api:
    image: yop-pms:latest
    ports:
      - "8080:8080"
    environment:
      DB_URL: postgres://yop:${DB_PASSWORD}@db:5432/yop_pms
      APP_ENV: prod
      PORT: 8080
      REDIS_ADDR: redis:6379
      REDIS_PASSWORD: ${REDIS_PASSWORD}
      ALLOWED_ORIGINS: https://app.example.com
      OTLP_ENDPOINT: http://jaeger:14268/api/traces
      SERVICE_NAME: yop-pms-prod
      SERVICE_VERSION: ${VERSION}
    depends_on:
      - db
      - redis
      - jaeger

  db:
    image: postgres:18-alpine
    environment:
      POSTGRES_USER: yop
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: yop_pms
    volumes:
      - db-data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis-data:/data

  jaeger:
    image: jaegertracing/all-in-one
    ports:
      - "16686:16686"

volumes:
  db-data:
  redis-data:
```

### SSL Certificate Configuration

For HTTPS endpoints (handled by reverse proxy typically, but shown here for completeness):

```bash
# PostgreSQL with SSL
DB_URL=postgres://user:password@db.example.com:5432/yop_pms?sslmode=require

# Redis with TLS (if supported)
REDIS_ADDR=redis.example.com:6380
# Note: Current setup uses redis/go-redis/v9 which supports TLS via custom options
```

---

## Secrets Management

### Secret Handling Best Practices

1. **Never commit secrets** — `.env` is in `.gitignore`
2. **Use secret managers** — HashiCorp Vault, AWS Secrets Manager, Kubernetes Secrets
3. **Rotate regularly** — Database passwords, Redis passwords, API keys
4. **Audit access** — Log who accessed what secrets and when

### Using AWS Secrets Manager

```bash
# Store secret
aws secretsmanager create-secret \
  --name prod/yop-pms/db-password \
  --secret-string "your-db-password"

# Retrieve at runtime (in Go code or Docker init script)
aws secretsmanager get-secret-value \
  --secret-id prod/yop-pms/db-password \
  --query SecretString \
  --output text
```

### Using Kubernetes Secrets

```bash
# Create secret
kubectl create secret generic yop-secrets \
  --from-literal=DB_URL='postgres://...' \
  --from-literal=REDIS_PASSWORD='...'

# Reference in Pod
spec:
  containers:
  - name: api
    env:
    - name: DB_URL
      valueFrom:
        secretKeyRef:
          name: yop-secrets
          key: DB_URL
```

---

## Validation at Startup

The server validates configuration at startup:

```go
// cmd/server/main.go
cfg := config.MustLoad()  // Panics if required vars missing

if cfg.DatabaseURL == "" {
    log.Fatal("DB_URL is required")
}
```

Required variables that cause startup failure:
- `DB_URL`

Optional variables fall back to defaults if missing.

---

## Performance Tuning

### PostgreSQL Connection Pool

Managed by pgxpool (auto-tuned based on CPU count). Override if needed:

```bash
# In go code - future enhancement
// poolConfig.MaxConns = 50
// poolConfig.MinConns = 5
```

### Redis Connection

Single client with connection pooling built-in.

### HTTP Server

Configured with sensible defaults:

```bash
IdleTimeout:  1 minute  # Close idle connections
ReadTimeout:  10 seconds
WriteTimeout: 30 seconds
```

Override with environment variables (future enhancement):

```bash
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=60s
HTTP_IDLE_TIMEOUT=2m
```

---

## Monitoring & Observability

### Metrics to Monitor

1. **HTTP Requests**
   - Requests/sec
   - Error rate
   - Latency (p50, p95, p99)

2. **Database**
   - Query latency
   - Connections used
   - Connection pool utilization

3. **Redis**
   - Cache hit rate
   - Eviction rate
   - Command latency

4. **Application**
   - Goroutines running
   - Memory allocation
   - GC pause time

### Health Check Endpoint

```bash
curl http://localhost:8080/healthz
```

Response:

```json
{
  "status": "ok",
  "message": "Yop API health check",
  "version": "0.1.0",
  "services": {
    "postgres": {
      "status": "ok",
      "latency": "5ms"
    },
    "redis": {
      "status": "ok",
      "latency": "2ms"
    }
  }
}
```

---

## Troubleshooting Configuration Issues

### "DB_URL is required"

```bash
echo $DB_URL  # Should not be empty
export DB_URL=postgres://user:password@localhost:5433/yop_pms
make dev
```

### "connection refused" to Redis

```bash
# Check if Redis is running
redis-cli ping  # Should return PONG

# Or via Docker
docker-compose up redis  # If using compose
```

### "permission denied" on database

```bash
# Check PostgreSQL user has correct permissions
psql -U yop -d yop_pms -c "SELECT 1;"

# Or recreate with correct password
make reset-db
```

### Traces not appearing in collector

```bash
# Verify endpoint is correct
echo $OTLP_ENDPOINT

# Check collector is running and listening
curl http://localhost:14268/api/traces

# Enable verbose logging temporarily
APP_ENV=dev make dev
```

