# Deployment & Scaling Guide

This guide covers deploying Yop PMS to production and scaling for growth.

## Table of Contents

- [Building for Production](#building-for-production)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Database Migrations](#database-migrations)
- [Scaling Considerations](#scaling-considerations)
- [Monitoring & Alerts](#monitoring--alerts)

---

## Building for Production

### Build Binary

```bash
# Build for Linux (production typically runs on Linux)
GOOS=linux GOARCH=amd64 go build -o /tmp/yop-server ./cmd/server

# Or use Make
make build
```

### Docker Image

Create a production-ready image with minimal size:

```dockerfile
# Dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -o yop-server ./cmd/server

FROM scratch
COPY --from=builder /app/yop-server /yop-server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080
ENTRYPOINT ["/yop-server"]
```

### Build and Push Image

```bash
# Build image
docker build -t yop-pms:latest .
docker build -t yop-pms:$(git rev-parse --short HEAD) .

# Push to registry
docker tag yop-pms:latest registry.example.com/yop-pms:latest
docker push registry.example.com/yop-pms:latest
```

---

## Docker Deployment

### Single Instance Deployment

```bash
docker run \
  -e DB_URL=postgres://user:password@db.example.com:5432/yop_pms \
  -e REDIS_ADDR=redis.example.com:6379 \
  -e APP_ENV=prod \
  -e ALLOWED_ORIGINS=https://app.example.com \
  -p 8080:8080 \
  yop-pms:latest
```

### Docker Compose for Staging

```yaml
version: "3.9"

services:
  api:
    image: yop-pms:latest
    ports:
      - "8080:8080"
    environment:
      DB_URL: postgres://yop:${DB_PASSWORD}@db:5432/yop_pms?sslmode=disable
      APP_ENV: staging
      REDIS_ADDR: redis:6379
      REDIS_PASSWORD: ${REDIS_PASSWORD}
      ALLOWED_ORIGINS: https://staging.example.com
      OTLP_ENDPOINT: http://jaeger:14268/api/traces
      SERVICE_VERSION: ${GIT_COMMIT}
    depends_on:
      - db
      - redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3

  db:
    image: postgres:18-alpine
    environment:
      POSTGRES_USER: yop
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: yop_pms
    volumes:
      - db-data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis-data:/data
    restart: unless-stopped

  jaeger:
    image: jaegertracing/all-in-one
    ports:
      - "16686:16686"
    restart: unless-stopped

volumes:
  db-data:
  redis-data:
```

---

## Kubernetes Deployment

### Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: yop-pms
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: yop-config
  namespace: yop-pms
data:
  APP_ENV: prod
  PORT: "8080"
  ALLOWED_ORIGINS: "https://app.example.com"
  OTLP_ENDPOINT: "http://jaeger-collector:14268/api/traces"
  SERVICE_NAME: yop-pms-prod
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: yop-secrets
  namespace: yop-pms
type: Opaque
stringData:
  DB_URL: "postgres://yop:password@postgres:5432/yop_pms"
  REDIS_PASSWORD: "redis-password"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: yop-pms-api
  namespace: yop-pms
spec:
  replicas: 3
  selector:
    matchLabels:
      app: yop-pms-api
  template:
    metadata:
      labels:
        app: yop-pms-api
    spec:
      containers:
      - name: api
        image: registry.example.com/yop-pms:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
          name: http

        env:
        # ConfigMap values
        - name: APP_ENV
          valueFrom:
            configMapKeyRef:
              name: yop-config
              key: APP_ENV
        - name: PORT
          valueFrom:
            configMapKeyRef:
              name: yop-config
              key: PORT
        - name: ALLOWED_ORIGINS
          valueFrom:
            configMapKeyRef:
              name: yop-config
              key: ALLOWED_ORIGINS
        - name: OTLP_ENDPOINT
          valueFrom:
            configMapKeyRef:
              name: yop-config
              key: OTLP_ENDPOINT
        - name: SERVICE_NAME
          valueFrom:
            configMapKeyRef:
              name: yop-config
              key: SERVICE_NAME

        # Secret values
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: yop-secrets
              key: DB_URL
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: yop-secrets
              key: REDIS_PASSWORD

        # Derived from pod metadata
        - name: SERVICE_VERSION
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['version']

        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"

        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
          timeoutSeconds: 5

        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5

      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - yop-pms-api
              topologyKey: kubernetes.io/hostname
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: yop-pms-api
  namespace: yop-pms
spec:
  type: ClusterIP
  selector:
    app: yop-pms-api
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
```

### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: yop-pms
  namespace: yop-pms
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - api.example.com
    secretName: yop-pms-tls
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: yop-pms-api
            port:
              number: 80
```

---

## Database Migrations

### Pre-deployment Migration

Run migrations **before** deploying new code that depends on schema changes:

```bash
# Local
goose -dir migrations postgres "$DB_URL" up

# Kubernetes Job
apiVersion: batch/v1
kind: Job
metadata:
  name: yop-migrate
  namespace: yop-pms
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: registry.example.com/yop-pms:latest
        env:
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: yop-secrets
              key: DB_URL
        command: ["goose", "-dir", "migrations", "postgres", "$DB_URL", "up"]
      restartPolicy: Never
  backoffLimit: 3
```

### Rollback Procedure

```bash
# Check current migration status
goose -dir migrations postgres "$DB_URL" status

# Rollback to previous version
goose -dir migrations postgres "$DB_URL" down

# Or rollback N versions
goose -dir migrations postgres "$DB_URL" down-to 4
```

---

## Scaling Considerations

### Horizontal Scaling (Multiple Instances)

The API server is stateless and can be scaled horizontally:

```yaml
# Kubernetes
spec:
  replicas: 10  # Scale to 10 instances
```

### Database Scaling

PostgreSQL is the bottleneck as load increases:

1. **Read Replicas** — For read-heavy queries (availability checks)
   ```
   Primary: postgres://primary-db.aws.com:5432/yop_pms
   Replica: postgres://replica-db.aws.com:5432/yop_pms
   ```

2. **Connection Pooling** — Use pgBouncer between app and database
   ```
   pgBouncer -> Primary DB (writes)
   pgBouncer -> Replica DB (reads)
   ```

3. **Query Optimization** — Add indexes for hot queries
   ```sql
   CREATE INDEX idx_reservations_property_dates ON reservations(property_id, check_in, check_out);
   ```

### Redis Scaling

Redis can become a bottleneck for caching/idempotency:

1. **Redis Cluster** — Distribute data across multiple nodes
2. **Redis Sentinel** — High availability with automatic failover
3. **Local Cache** — Add LRU cache in application memory for ultra-hot keys

### Load Balancing

```yaml
# Kubernetes Service automatically load balances across pods
apiVersion: v1
kind: Service
metadata:
  name: yop-pms-api
spec:
  type: LoadBalancer  # Or ClusterIP + Ingress controller
  selector:
    app: yop-pms-api
  ports:
  - port: 80
    targetPort: 8080
```

### Caching Strategy

For read-heavy operations (checking availability):

1. **HTTP Cache** — Set `Cache-Control` headers in responses
2. **Redis Cache** — TTL-based caching for availability lookups
3. **Browser Cache** — Cache on frontend for repeated checks

---

## Monitoring & Alerts

### Metrics to Monitor

```yaml
# Prometheus ServiceMonitor (if using Prometheus)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: yop-pms
  namespace: yop-pms
spec:
  selector:
    matchLabels:
      app: yop-pms-api
  endpoints:
  - port: metrics
```

Key metrics:

1. **Request Latency** — p50, p95, p99
2. **Error Rate** — Errors per second
3. **Database Connection Pool** — Connections used / available
4. **Redis Memory** — Memory used / max
5. **Trace Sampling** — Traces exported per second

### Alert Rules

```yaml
# PrometheusRule
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: yop-pms-alerts
  namespace: yop-pms
spec:
  groups:
  - name: yop-pms
    interval: 30s
    rules:
    - alert: HighErrorRate
      expr: rate(yop_http_requests_total{status=~"5.."}[5m]) > 0.05
      for: 5m
      annotations:
        summary: "High error rate on {{ $labels.instance }}"

    - alert: HighLatency
      expr: histogram_quantile(0.95, yop_http_request_duration_seconds_bucket) > 1
      for: 5m
      annotations:
        summary: "High p95 latency: {{ $value }}s"

    - alert: DatabaseConnectionPoolExhausted
      expr: yop_db_connections_used / yop_db_connections_max > 0.9
      for: 5m
      annotations:
        summary: "Database connection pool 90% utilized"
```

### Logging & Tracing

All requests automatically:
- Logged with structured JSON (request_id, latency, status)
- Traced with OpenTelemetry (exportable to Jaeger, Datadog, etc.)

### Debugging Production

1. **Check logs** — Filter by trace_id to see full request flow
   ```bash
   kubectl logs -n yop-pms deployment/yop-pms-api | jq 'select(.trace_id == "abc123")'
   ```

2. **View traces** — Open Jaeger UI to see request spans
3. **Check health** — `curl https://api.example.com/healthz`

---

## Release Process

1. **Tag release**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Build and push image**
   ```bash
   docker build -t yop-pms:v1.0.0 .
   docker push registry.example.com/yop-pms:v1.0.0
   ```

3. **Run migrations**
   ```bash
   kubectl apply -f migrate-job.yaml
   kubectl wait --for=condition=complete job/yop-migrate -n yop-pms
   ```

4. **Deploy to Kubernetes**
   ```bash
   kubectl set image deployment/yop-pms-api \
     yop-pms-api=registry.example.com/yop-pms:v1.0.0 \
     -n yop-pms
   ```

5. **Verify rollout**
   ```bash
   kubectl rollout status deployment/yop-pms-api -n yop-pms
   ```

---

## Rollback Procedure

If something goes wrong:

1. **Rollback deployment**
   ```bash
   kubectl rollout undo deployment/yop-pms-api -n yop-pms
   ```

2. **Or deploy previous version**
   ```bash
   kubectl set image deployment/yop-pms-api \
     yop-pms-api=registry.example.com/yop-pms:v0.9.0 \
     -n yop-pms
   ```

3. **Verify health**
   ```bash
   kubectl logs -n yop-pms deployment/yop-pms-api --tail=50
   curl https://api.example.com/healthz
   ```

