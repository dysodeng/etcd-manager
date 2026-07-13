# Backend Runtime Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bound configuration API resource usage and add structured logging, request IDs, HTTP timeouts, and graceful shutdown without changing database schema.

**Architecture:** Configuration handlers and services enforce fixed import/list limits with typed errors. A small `internal/logging` package configures `slog`, Gin middleware supplies request observability, and testable `http.Server` helpers own deadlines and shutdown while preserving SSE behavior.

**Tech Stack:** Go 1.25, Gin, standard-library `slog`/`net/http`, etcd client v3, Viper, Go table-driven tests.

---

## File Structure

- Create `internal/logging/logging.go`: parse configured level and create JSON `slog.Logger`.
- Create `internal/logging/logging_test.go`: level/fallback and JSON output tests.
- Create `internal/middleware/observability.go`: request ID, access log, and structured recovery middleware.
- Create `internal/middleware/observability_test.go`: generated/forwarded IDs, access fields, and panic recovery tests.
- Modify `internal/config/config.go`: HTTP/shutdown duration fields, defaults, and environment bindings using an isolated Viper instance.
- Create `internal/config/config_test.go`: old-config default compatibility and explicit duration tests.
- Modify `internal/service/config_errors.go`, `internal/service/config_service.go`: fixed list item/byte limits and typed overflow error.
- Modify `internal/service/config_consistency_test.go`: list limit behavior using the existing fake store.
- Modify `internal/handler/config_center.go`, `internal/handler/config_center_test.go`: 10 MiB body reader and limit error mapping.
- Modify `internal/response/response.go`, `internal/handler/response.go`: add config-limit business code.
- Create `cmd/server/http_server.go`: route-aware write deadline, server construction, and graceful serving.
- Create `cmd/server/http_server_test.go`: server timeouts, SSE exemption, and context-driven shutdown tests.
- Modify `cmd/server/main.go`: `gin.New`, middleware wiring, structured lifecycle logs, signal context, resource closure.
- Modify `configs/config.yaml`, `deploy/helm/values.yaml`: expose timeout defaults.

### Task 1: Bound Config List and Import Resources

**Files:**
- Modify: `internal/service/config_errors.go`
- Modify: `internal/service/config_service.go`
- Modify: `internal/service/config_consistency_test.go`
- Modify: `internal/handler/config_center.go`
- Modify: `internal/handler/config_center_test.go`
- Modify: `internal/response/response.go`
- Modify: `internal/handler/response.go`

- [ ] **Step 1: Write failing service list-limit tests**

Extend `fakeConfigStore` with `prefixResponse *clientv3.GetResponse`, return it from `GetWithPrefix`, and add:

~~~go
func TestConfigListRejectsTooManyItems(t *testing.T) {
    kvs := make([]*mvccpb.KeyValue, MaxConfigListItems+1)
    for i := range kvs {
        kvs[i] = &mvccpb.KeyValue{Key: []byte(fmt.Sprintf("/dev/config/key-%03d", i)), Value: []byte("value")}
    }
    store := &fakeConfigStore{prefixResponse: &clientv3.GetResponse{Kvs: kvs}}
    svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

    _, err := svc.List(authorizedConfigContext(), "dev", "")

    if !errors.Is(err, ErrConfigListLimitExceeded) {
        t.Fatalf("error = %v, want ErrConfigListLimitExceeded", err)
    }
}

func TestConfigListRejectsTooManyBytes(t *testing.T) {
    store := &fakeConfigStore{prefixResponse: &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{{
        Key: []byte("/dev/config/large.yaml"), Value: bytes.Repeat([]byte("x"), MaxConfigListBytes),
    }}}}
    svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

    _, err := svc.List(authorizedConfigContext(), "dev", "")

    if !errors.Is(err, ErrConfigListLimitExceeded) {
        t.Fatalf("error = %v, want ErrConfigListLimitExceeded", err)
    }
}
~~~

- [ ] **Step 2: Verify service tests are RED**

Run: `go test ./internal/service -run '^TestConfigListRejects'`

Expected: build fails because the constants/sentinel are undefined.

- [ ] **Step 3: Implement list limits**

Add to `config_errors.go`:

~~~go
var ErrConfigListLimitExceeded = errors.New("configuration list limit exceeded")
~~~

Add to `config_service.go` and update `List`:

~~~go
const (
    MaxConfigListItems = 500
    MaxConfigListBytes = 10 << 20
)

resp, err := s.configStore.GetWithPrefix(ctx, fullPrefix, MaxConfigListItems+1)
if err != nil {
    return nil, err
}
if len(resp.Kvs) > MaxConfigListItems {
    return nil, fmt.Errorf("%w: more than %d items; narrow the prefix", ErrConfigListLimitExceeded, MaxConfigListItems)
}
items := make([]ConfigItem, 0, len(resp.Kvs))
totalBytes := 0
for _, kv := range resp.Kvs {
    totalBytes += len(kv.Key) + len(kv.Value)
    if totalBytes > MaxConfigListBytes {
        return nil, fmt.Errorf("%w: response exceeds %d bytes; narrow the prefix", ErrConfigListLimitExceeded, MaxConfigListBytes)
    }
    shortKey := strings.TrimPrefix(string(kv.Key), configBase)
    items = append(items, ConfigItem{Key: shortKey, Value: string(kv.Value)})
}
~~~

- [ ] **Step 4: Verify list tests are GREEN**

Run: `gofmt -w internal/service/config_errors.go internal/service/config_service.go internal/service/config_consistency_test.go && go test ./internal/service -run '^TestConfigListRejects'`

Expected: PASS.

- [ ] **Step 5: Write failing import body-limit and error-code tests**

Add imports `net/http`, `net/http/httptest`, and `strings` to `config_center_test.go`, then add:

~~~go
func TestReadConfigImportBodyRejectsOversizeRequest(t *testing.T) {
    request := httptest.NewRequest(http.MethodPost, "/api/v1/configs/import", strings.NewReader(strings.Repeat("x", int(MaxConfigImportBytes)+1)))
    recorder := httptest.NewRecorder()

    _, err := readConfigImportBody(recorder, request)

    var maxBytesErr *http.MaxBytesError
    if !errors.As(err, &maxBytesErr) {
        t.Fatalf("error = %v, want *http.MaxBytesError", err)
    }
}

func TestReadConfigImportBodyAcceptsLimit(t *testing.T) {
    request := httptest.NewRequest(http.MethodPost, "/api/v1/configs/import", strings.NewReader(strings.Repeat("x", int(MaxConfigImportBytes))))
    recorder := httptest.NewRecorder()
    body, err := readConfigImportBody(recorder, request)
    if err != nil || len(body) != int(MaxConfigImportBytes) {
        t.Fatalf("len=%d error=%v", len(body), err)
    }
}
~~~

Add `{name: "list limit", err: service.ErrConfigListLimitExceeded, code: CodeConfigLimitExceeded}` to `TestConfigWriteErrorCode`.

- [ ] **Step 6: Verify handler tests are RED**

Run: `go test ./internal/handler -run 'Test(ReadConfigImportBody|ConfigWriteErrorCode)'`

Expected: build fails because the reader, limit constant, and response code are undefined.

- [ ] **Step 7: Implement bounded body reading and typed mapping**

Add to `config_center.go`:

~~~go
const MaxConfigImportBytes int64 = 10 << 20

func readConfigImportBody(w http.ResponseWriter, request *http.Request) ([]byte, error) {
    request.Body = http.MaxBytesReader(w, request.Body, MaxConfigImportBytes)
    defer request.Body.Close()
    return io.ReadAll(request.Body)
}
~~~

Use it from `Import`:

~~~go
body, err := readConfigImportBody(c.Writer, c.Request)
if err != nil {
    var maxBytesErr *http.MaxBytesError
    if errors.As(err, &maxBytesErr) {
        Fail(c, CodeParamInvalid, "import body exceeds 10 MiB")
    } else {
        Fail(c, CodeImportFormat, "failed to read request body")
    }
    return
}
~~~

Add `CodeConfigLimitExceeded = 20007` to `internal/response/response.go`, re-export it from `internal/handler/response.go`, and add this classifier branch:

~~~go
case errors.Is(err, service.ErrConfigListLimitExceeded):
    return CodeConfigLimitExceeded
~~~

- [ ] **Step 8: Verify all resource-limit tests and commit**

Run: `gofmt -w internal/service/config_errors.go internal/service/config_service.go internal/service/config_consistency_test.go internal/handler/config_center.go internal/handler/config_center_test.go internal/response/response.go internal/handler/response.go && go test ./internal/service ./internal/handler`

Expected: PASS.

~~~bash
git add internal/service/config_errors.go internal/service/config_service.go internal/service/config_consistency_test.go internal/handler/config_center.go internal/handler/config_center_test.go internal/response/response.go internal/handler/response.go
git commit -m "fix: bound config import and list resources"
~~~

### Task 2: Structured Logger and Configured Level

**Files:**
- Create: `internal/logging/logging.go`
- Create: `internal/logging/logging_test.go`

- [ ] **Step 1: Write failing logger tests**

~~~go
package logging

import (
    "bytes"
    "encoding/json"
    "log/slog"
    "testing"
)

func TestParseLevel(t *testing.T) {
    tests := []struct{name string; want slog.Level; valid bool}{
        {"debug", slog.LevelDebug, true}, {"INFO", slog.LevelInfo, true},
        {"warn", slog.LevelWarn, true}, {"error", slog.LevelError, true},
        {"verbose", slog.LevelInfo, false},
    }
    for _, tt := range tests {
        got, valid := ParseLevel(tt.name)
        if got != tt.want || valid != tt.valid { t.Fatalf("%q: level=%v valid=%v", tt.name, got, valid) }
    }
}

func TestNewJSONLoggerHonorsLevel(t *testing.T) {
    var output bytes.Buffer
    logger, _ := NewJSONLogger(&output, "warn")
    logger.Info("hidden")
    logger.Warn("visible", "request_id", "req-1")
    var event map[string]any
    if err := json.Unmarshal(output.Bytes(), &event); err != nil { t.Fatal(err) }
    if event["msg"] != "visible" || event["request_id"] != "req-1" { t.Fatalf("event=%v", event) }
}
~~~

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/logging`

Expected: build fails because `ParseLevel` and `NewJSONLogger` are undefined.

- [ ] **Step 3: Implement logger construction**

~~~go
package logging

import (
    "io"
    "log/slog"
    "strings"
)

func ParseLevel(value string) (slog.Level, bool) {
    switch strings.ToLower(strings.TrimSpace(value)) {
    case "debug": return slog.LevelDebug, true
    case "", "info": return slog.LevelInfo, true
    case "warn", "warning": return slog.LevelWarn, true
    case "error": return slog.LevelError, true
    default: return slog.LevelInfo, false
    }
}

func NewJSONLogger(writer io.Writer, configuredLevel string) (*slog.Logger, bool) {
    level, valid := ParseLevel(configuredLevel)
    handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
    return slog.New(handler), valid
}
~~~

- [ ] **Step 4: Verify GREEN and commit**

Run: `gofmt -w internal/logging/logging.go internal/logging/logging_test.go && go test ./internal/logging`

Expected: PASS.

~~~bash
git add internal/logging/logging.go internal/logging/logging_test.go
git commit -m "feat: add configured structured logger"
~~~

### Task 3: Request Observability Middleware

**Files:**
- Create: `internal/middleware/observability.go`
- Create: `internal/middleware/observability_test.go`
- Modify: `internal/middleware/cors.go`

- [ ] **Step 1: Write failing request ID tests**

Create Gin test routers and cover both paths:

~~~go
func TestRequestIDForwardsAndGeneratesID(t *testing.T) {
    tests := []struct{name, supplied string}{
        {name: "forwarded", supplied: "request-from-proxy"},
        {name: "generated"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            router := gin.New()
            var contextID string
            router.Use(RequestID())
            router.GET("/", func(c *gin.Context) { contextID = RequestIDFromContext(c.Request.Context()); c.Status(http.StatusNoContent) })
            request := httptest.NewRequest(http.MethodGet, "/", nil)
            request.Header.Set(RequestIDHeader, tt.supplied)
            recorder := httptest.NewRecorder()
            router.ServeHTTP(recorder, request)
            got := recorder.Header().Get(RequestIDHeader)
            if got == "" || got != contextID { t.Fatalf("header=%q context=%q", got, contextID) }
            if tt.supplied != "" && got != tt.supplied { t.Fatalf("got=%q", got) }
        })
    }
}
~~~

- [ ] **Step 2: Write failing access log and recovery tests**

~~~go
func TestAccessLoggerWritesStructuredFields(t *testing.T) {
    var output bytes.Buffer
    logger := slog.New(slog.NewJSONHandler(&output, nil))
    router := gin.New()
    router.Use(RequestID(), AccessLogger(logger), Recovery(logger))
    router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusCreated) })
    recorder := httptest.NewRecorder()
    router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ok?token=secret", nil))
    var event map[string]any
    if err := json.Unmarshal(output.Bytes(), &event); err != nil { t.Fatal(err) }
    for _, key := range []string{"request_id", "method", "path", "status", "latency_ms", "client_ip"} {
        if _, ok := event[key]; !ok { t.Fatalf("missing %s in %v", key, event) }
    }
    if strings.Contains(output.String(), "token=secret") { t.Fatal("query leaked into access log") }
}

func TestRecoveryReturnsInternalError(t *testing.T) {
    var output bytes.Buffer
    logger := slog.New(slog.NewJSONHandler(&output, nil))
    router := gin.New()
    router.Use(RequestID(), Recovery(logger))
    router.GET("/panic", func(*gin.Context) { panic("boom") })
    recorder := httptest.NewRecorder()
    router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
    if recorder.Code != http.StatusInternalServerError { t.Fatalf("status=%d", recorder.Code) }
    if !strings.Contains(output.String(), "request_id") || !strings.Contains(output.String(), "panic") { t.Fatalf("log=%s", output.String()) }
}
~~~

- [ ] **Step 3: Verify RED**

Run: `go test ./internal/middleware -run 'Test(RequestID|AccessLogger|Recovery)'`

Expected: build fails because the observability middleware is undefined.

- [ ] **Step 4: Implement middleware**

`observability.go` defines a private context key and these exported APIs:

~~~go
const RequestIDHeader = "X-Request-ID"

func RequestIDFromContext(ctx context.Context) string {
    id, _ := ctx.Value(requestIDContextKey{}).(string)
    return id
}

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        id := strings.TrimSpace(c.GetHeader(RequestIDHeader))
        if id == "" || len(id) > 128 { id = uuid.NewString() }
        c.Header(RequestIDHeader, id)
        c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDContextKey{}, id))
        c.Next()
    }
}

func AccessLogger(logger *slog.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        started := time.Now()
        c.Next()
        logger.InfoContext(c.Request.Context(), "http request",
            "request_id", RequestIDFromContext(c.Request.Context()), "method", c.Request.Method,
            "path", c.Request.URL.Path, "status", c.Writer.Status(),
            "latency_ms", time.Since(started).Milliseconds(), "client_ip", c.ClientIP())
    }
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if recovered := recover(); recovered != nil {
                logger.ErrorContext(c.Request.Context(), "http panic", "request_id", RequestIDFromContext(c.Request.Context()), "panic", fmt.Sprint(recovered))
                c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{Code: response.CodeInternalError, Message: "internal server error"})
            }
        }()
        c.Next()
    }
}
~~~

Update CORS headers to include and expose request ID:

~~~go
c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
c.Header("Access-Control-Expose-Headers", "X-Request-ID")
~~~

- [ ] **Step 5: Verify GREEN and commit**

Run: `gofmt -w internal/middleware/observability.go internal/middleware/observability_test.go internal/middleware/cors.go && go test ./internal/middleware`

Expected: PASS.

~~~bash
git add internal/middleware/observability.go internal/middleware/observability_test.go internal/middleware/cors.go
git commit -m "feat: add structured request observability"
~~~

### Task 4: HTTP Timeout Configuration and Graceful Server Helpers

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `cmd/server/http_server.go`
- Create: `cmd/server/http_server_test.go`

- [ ] **Step 1: Write failing config default tests**

~~~go
func TestLoadAppliesServerTimeoutDefaults(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.yaml")
    if err := os.WriteFile(path, []byte("server:\n  port: 8080\n"), 0o600); err != nil { t.Fatal(err) }
    cfg, err := Load(path)
    if err != nil { t.Fatal(err) }
    if cfg.Server.ReadHeaderTimeout != 5*time.Second || cfg.Server.ReadTimeout != 15*time.Second ||
        cfg.Server.WriteTimeout != 30*time.Second || cfg.Server.IdleTimeout != 60*time.Second ||
        cfg.Server.ShutdownTimeout != 15*time.Second {
        t.Fatalf("server config=%+v", cfg.Server)
    }
}
~~~

- [ ] **Step 2: Verify config test is RED**

Run: `go test ./internal/config -run '^TestLoadAppliesServerTimeoutDefaults$'`

Expected: build fails because duration fields are undefined.

- [ ] **Step 3: Implement isolated Viper defaults and bindings**

Extend `ServerConfig`:

~~~go
type ServerConfig struct {
    Port              int           `mapstructure:"port"`
    ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
    ReadTimeout       time.Duration `mapstructure:"read_timeout"`
    WriteTimeout      time.Duration `mapstructure:"write_timeout"`
    IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
    ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}
~~~

In `Load`, replace package-global Viper calls with `v := viper.New()`, set defaults before `ReadInConfig`, and bind the new environment variables:

~~~go
v.SetDefault("server.read_header_timeout", "5s")
v.SetDefault("server.read_timeout", "15s")
v.SetDefault("server.write_timeout", "30s")
v.SetDefault("server.idle_timeout", "60s")
v.SetDefault("server.shutdown_timeout", "15s")
_ = v.BindEnv("server.read_header_timeout", "SERVER_READ_HEADER_TIMEOUT")
_ = v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
_ = v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
_ = v.BindEnv("server.idle_timeout", "SERVER_IDLE_TIMEOUT")
_ = v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")
_ = v.BindEnv("log.level", "LOG_LEVEL")
~~~

- [ ] **Step 4: Verify config test is GREEN**

Run: `gofmt -w internal/config/config.go internal/config/config_test.go && go test ./internal/config`

Expected: PASS.

- [ ] **Step 5: Write failing HTTP server helper tests**

Define a `deadlineRecorder` implementing `SetWriteDeadline(time.Time) error`, then add:

~~~go
func TestResponseWriteDeadlineExemptsWatch(t *testing.T) {
    tests := []struct{path string; wantCalls int}{{"/api/v1/configs", 2}, {"/api/v1/watch", 0}}
    for _, tt := range tests {
        recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
        handler := withResponseWriteDeadline(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), 30*time.Second)
        handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
        if len(recorder.deadlines) != tt.wantCalls { t.Fatalf("%s calls=%d", tt.path, len(recorder.deadlines)) }
    }
}

func TestNewHTTPServerUsesConfiguredTimeouts(t *testing.T) {
    cfg := config.ServerConfig{ReadHeaderTimeout: time.Second, ReadTimeout: 2*time.Second, WriteTimeout: 3*time.Second, IdleTimeout: 4*time.Second}
    server := newHTTPServer(":0", http.NotFoundHandler(), cfg)
    if server.ReadHeaderTimeout != time.Second || server.ReadTimeout != 2*time.Second || server.IdleTimeout != 4*time.Second || server.WriteTimeout != 0 {
        t.Fatalf("server=%+v", server)
    }
}

func TestServeHTTPShutsDownWhenContextIsCanceled(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    ctx, cancel := context.WithCancel(context.Background())
    server := newHTTPServer(listener.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), config.ServerConfig{})
    done := make(chan error, 1)
    go func() { done <- serveHTTP(ctx, server, listener, time.Second) }()
    cancel()
    if err := <-done; err != nil { t.Fatalf("serveHTTP() error=%v", err) }
}
~~~

- [ ] **Step 6: Verify helper tests are RED**

Run: `go test ./cmd/server -run 'Test(ResponseWriteDeadline|NewHTTPServer|ServeHTTP)'`

Expected: build fails because server helpers are undefined.

- [ ] **Step 7: Implement deadline and graceful serve helpers**

`cmd/server/http_server.go`:

~~~go
func withResponseWriteDeadline(next http.Handler, timeout time.Duration) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if timeout > 0 && r.URL.Path != "/api/v1/watch" {
            controller := http.NewResponseController(w)
            if err := controller.SetWriteDeadline(time.Now().Add(timeout)); err == nil {
                defer controller.SetWriteDeadline(time.Time{})
            }
        }
        next.ServeHTTP(w, r)
    })
}

func newHTTPServer(address string, handler http.Handler, cfg config.ServerConfig) *http.Server {
    return &http.Server{
        Addr: address, Handler: withResponseWriteDeadline(handler, cfg.WriteTimeout),
        ReadHeaderTimeout: cfg.ReadHeaderTimeout, ReadTimeout: cfg.ReadTimeout,
        WriteTimeout: 0, IdleTimeout: cfg.IdleTimeout,
    }
}

func serveHTTP(ctx context.Context, server *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
    result := make(chan error, 1)
    go func() { result <- server.Serve(listener) }()
    select {
    case err := <-result:
        if errors.Is(err, http.ErrServerClosed) { return nil }
        return err
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
        defer cancel()
        if err := server.Shutdown(shutdownCtx); err != nil { return err }
        err := <-result
        if errors.Is(err, http.ErrServerClosed) { return nil }
        return err
    }
}
~~~

- [ ] **Step 8: Verify helper tests are GREEN and commit**

Run: `gofmt -w cmd/server/http_server.go cmd/server/http_server_test.go && go test ./cmd/server ./internal/config`

Expected: PASS.

~~~bash
git add internal/config/config.go internal/config/config_test.go cmd/server/http_server.go cmd/server/http_server_test.go
git commit -m "feat: add HTTP timeouts and graceful serving"
~~~

### Task 5: Wire Runtime Lifecycle and Deployment Defaults

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `configs/config.yaml`
- Modify: `deploy/helm/values.yaml`

- [ ] **Step 1: Replace process-global startup with signal context**

Use:

~~~go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    if err := run(ctx); err != nil {
        slog.Error("server stopped with error", "error", err)
        os.Exit(1)
    }
}
~~~

- [ ] **Step 2: Configure logger, close resources, and build Gin without default logging**

At the start of `run`, load config, create the logger, set it as default, and warn on an unknown configured level:

~~~go
logger, validLevel := logging.NewJSONLogger(os.Stdout, cfg.Log.Level)
slog.SetDefault(logger)
if !validLevel { logger.Warn("unknown log level; using info", "configured_level", cfg.Log.Level) }
~~~

After opening GORM and etcd resources:

~~~go
sqlDB, err := db.DB()
if err != nil { return fmt.Errorf("get sql database: %w", err) }
defer sqlDB.Close()
defer etcdClient.Close()
~~~

Replace `gin.Default()` with:

~~~go
r := gin.New()
r.Use(middleware.RequestID(), middleware.AccessLogger(logger), middleware.Recovery(logger))
~~~

Replace `r.Run` with listener/server helpers:

~~~go
listener, err := net.Listen("tcp", addr)
if err != nil { return fmt.Errorf("listen %s: %w", addr, err) }
server := newHTTPServer(addr, r, cfg.Server)
logger.Info("server starting", "address", addr)
if err := serveHTTP(ctx, server, listener, cfg.Server.ShutdownTimeout); err != nil {
    return fmt.Errorf("serve HTTP: %w", err)
}
logger.Info("server stopped")
return nil
~~~

Convert every existing `log.Fatalf/Printf` branch into returned wrapped errors or `logger.Info/Error` calls. Remove the standard `log` import.

- [ ] **Step 3: Add timeout defaults to local and Helm config**

Under `server` in both YAML files add:

~~~yaml
read_header_timeout: 5s
read_timeout: 15s
write_timeout: 30s
idle_timeout: 60s
shutdown_timeout: 15s
~~~

Under Helm `backend.env` document the five `SERVER_*_TIMEOUT` overrides plus `LOG_LEVEL`.

- [ ] **Step 4: Verify runtime wiring and commit**

Run:

~~~bash
gofmt -w cmd/server/main.go
go test ./cmd/server ./internal/config ./internal/logging ./internal/middleware
go vet ./cmd/server ./internal/config ./internal/logging ./internal/middleware
helm lint deploy/helm
~~~

Expected: all Go commands exit 0; Helm reports 0 failed charts.

~~~bash
git add cmd/server/main.go configs/config.yaml deploy/helm/values.yaml
git commit -m "feat: wire graceful structured HTTP runtime"
~~~

### Task 6: Backend Verification

**Files:**
- Verify all backend files changed by Tasks 1-5.

- [ ] **Step 1: Verify changed-file formatting and diff hygiene**

Run:

~~~bash
changed_go=$(git diff --name-only main...HEAD -- '*.go')
test -z "$(printf '%s\n' "$changed_go" | xargs gofmt -l)"
git diff --check
~~~

Expected: no output and exit 0.

- [ ] **Step 2: Run full backend verification**

Run:

~~~bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
~~~

Expected: all commands exit 0 with no test failures or race reports.

- [ ] **Step 3: Confirm database schema is untouched**

Run:

~~~bash
git diff --quiet main...HEAD -- internal/store internal/domain/entity.go
~~~

Expected: exit 0; no model, migration, table, or field changes.
