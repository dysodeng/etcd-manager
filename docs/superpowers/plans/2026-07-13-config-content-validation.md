# Config Content Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reject syntactically invalid YAML, JSON, and TOML before configuration-center values are written to etcd.

**Architecture:** Add a pure validator in the service package that selects a parser from the config key suffix and returns a typed validation error. Call it at the service write boundary so create, update, rollback, and import share the same rule; handlers map typed validation failures to the existing parameter-error response code.

**Tech Stack:** Go 1.25, `gopkg.in/yaml.v3`, `encoding/json`, `github.com/pelletier/go-toml/v2`, Gin, Go table-driven tests.

---

## File Structure

- Create `internal/service/config_validator.go`: suffix detection, full-input parsing, and typed validation errors.
- Create `internal/service/config_validator_test.go`: parser behavior and suffix-routing unit tests.
- Modify `internal/service/config_service.go`: run validation before create/update writes and validate every imported item.
- Create `internal/service/config_service_test.go`: verify import dry-run counts syntax failures without requiring etcd.
- Modify `internal/handler/config_center.go`: map typed validation errors to `CodeParamInvalid`.
- Create `internal/handler/config_center_test.go`: verify handler error classification.
- Modify `go.mod` and `go.sum`: promote the existing TOML parser to a direct dependency.

### Task 1: Configuration Content Validator

**Files:**
- Create: `internal/service/config_validator.go`
- Create: `internal/service/config_validator_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Write the failing validator tests**

Create `internal/service/config_validator_test.go`:

~~~go
package service

import (
    "errors"
    "testing"
)

func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name string
        key string
        value string
        wantErr bool
    }{
        {name: "valid yaml", key: "app.yaml", value: "server:\n  port: 8080\n"},
        {name: "valid uppercase yml", key: "app.YML", value: "enabled: true\n"},
        {name: "invalid yaml", key: "app.yaml", value: "server: {host: localhost", wantErr: true},
        {name: "valid multi-document yaml", key: "app.yml", value: "name: first\n---\nname: second\n"},
        {name: "invalid later yaml document", key: "app.yml", value: "name: first\n---\nitems: [one, two", wantErr: true},
        {name: "valid json", key: "app.json", value: `{"server":{"port":8080}}`},
        {name: "invalid incomplete json", key: "app.json", value: `{"server":`, wantErr: true},
        {name: "invalid json trailing content", key: "app.json", value: `{"ok":true} garbage`, wantErr: true},
        {name: "valid toml", key: "app.toml", value: "[server]\nport = 8080\n"},
        {name: "invalid toml", key: "app.toml", value: "name = \"broken", wantErr: true},
        {name: "unknown suffix", key: "app.conf", value: "{ incomplete"},
        {name: "no suffix", key: "app/config", value: "{ incomplete"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateConfig(tt.key, tt.value)
            if tt.wantErr && err == nil {
                t.Fatal("ValidateConfig() error = nil, want validation error")
            }
            if !tt.wantErr && err != nil {
                t.Fatalf("ValidateConfig() error = %v, want nil", err)
            }
            if tt.wantErr {
                var validationErr *ConfigValidationError
                if !errors.As(err, &validationErr) {
                    t.Fatalf("error type = %T, want *ConfigValidationError", err)
                }
            }
        })
    }
}
~~~

- [ ] **Step 2: Run the test and verify RED**

Run: `go test ./internal/service -run '^TestValidateConfig$'`

Expected: build fails because `ValidateConfig` and `ConfigValidationError` are undefined.

- [ ] **Step 3: Add the TOML parser dependency**

Run: `go get github.com/pelletier/go-toml/v2@v2.2.4`

Expected: the TOML module moves from the indirect block to the direct block in `go.mod`.

- [ ] **Step 4: Implement the minimal validator**

Create `internal/service/config_validator.go`:

~~~go
package service

import (
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "path"
    "strings"

    "github.com/pelletier/go-toml/v2"
    "gopkg.in/yaml.v3"
)

type ConfigValidationError struct {
    Format string
    Err error
}

func (e *ConfigValidationError) Error() string {
    return fmt.Sprintf("invalid %s config: %v", e.Format, e.Err)
}

func (e *ConfigValidationError) Unwrap() error {
    return e.Err
}

func ValidateConfig(key, value string) error {
    ext := strings.ToLower(path.Ext(key))
    var format string
    var err error
    switch ext {
    case ".yaml", ".yml":
        format, err = "YAML", validateYAML(value)
    case ".json":
        format, err = "JSON", validateJSON(value)
    case ".toml":
        format, err = "TOML", validateTOML(value)
    default:
        return nil
    }
    if err != nil {
        return &ConfigValidationError{Format: format, Err: err}
    }
    return nil
}

func validateYAML(value string) error {
    decoder := yaml.NewDecoder(strings.NewReader(value))
    for {
        var document any
        err := decoder.Decode(&document)
        if errors.Is(err, io.EOF) {
            return nil
        }
        if err != nil {
            return err
        }
    }
}

func validateJSON(value string) error {
    decoder := json.NewDecoder(strings.NewReader(value))
    var document any
    if err := decoder.Decode(&document); err != nil {
        return err
    }
    var trailing any
    if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
        if err == nil {
            return errors.New("multiple JSON values")
        }
        return err
    }
    return nil
}

func validateTOML(value string) error {
    var document map[string]any
    return toml.Unmarshal([]byte(value), &document)
}
~~~

- [ ] **Step 5: Format and verify GREEN**

Run:

~~~bash
gofmt -w internal/service/config_validator.go internal/service/config_validator_test.go
go test ./internal/service -run '^TestValidateConfig$'
~~~

Expected: the validator test passes.

- [ ] **Step 6: Commit**

~~~bash
git add internal/service/config_validator.go internal/service/config_validator_test.go go.mod go.sum
git commit -m "feat: validate structured config syntax"
~~~

### Task 2: Service Write-Boundary Integration

**Files:**
- Modify: `internal/service/config_service.go`
- Create: `internal/service/config_service_test.go`

- [ ] **Step 1: Write the failing import dry-run test**

Create `internal/service/config_service_test.go`:

~~~go
package service

import (
    "context"
    "strings"
    "testing"

    "github.com/google/uuid"
)

func TestConfigServiceImportDryRunValidatesConfigValues(t *testing.T) {
    svc := &ConfigService{}
    data := []byte(`{"good.yaml":"name: app\\n","bad.json":"{\\\"broken\\\":"}`)
    result, err := svc.Import(context.Background(), "dev", data, true, uuid.Nil)
    if err != nil {
        t.Fatalf("Import() error = %v", err)
    }
    if result.Total != 2 || result.Success != 1 || len(result.Failed) != 1 {
        t.Fatalf("Import() result = %+v, want total=2 success=1 failed=1", result)
    }
    if !strings.Contains(result.Failed[0], "bad.json") ||
        !strings.Contains(result.Failed[0], "invalid JSON config") {
        t.Fatalf("Import() failure = %q, want key and validation reason", result.Failed[0])
    }
}
~~~

- [ ] **Step 2: Run the test and verify RED**

Run: `go test ./internal/service -run '^TestConfigServiceImportDryRunValidatesConfigValues$'`

Expected: test fails because current dry-run reports `success=2` and no failed entries.

- [ ] **Step 3: Validate before Create and Update side effects**

Add this at the beginning of both `ConfigService.Create` and `ConfigService.Update`:

~~~go
if err := ValidateConfig(key, value); err != nil {
    return err
}
~~~

`Rollback` already calls `Update`, so it automatically uses the same rule.

- [ ] **Step 4: Validate imports before dry-run or writes**

After parsing `configs`, replace the unconditional dry-run block with:

~~~go
result := &ImportResult{Total: len(configs)}
validConfigs := make(map[string]string, len(configs))
for key, value := range configs {
    if err := ValidateConfig(key, value); err != nil {
        result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", key, err))
        continue
    }
    validConfigs[key] = value
}
if dryRun {
    result.Success = len(validConfigs)
    return result, nil
}
~~~

Change the actual write loop to `for key, value := range validConfigs`.

- [ ] **Step 5: Format and verify GREEN**

Run:

~~~bash
gofmt -w internal/service/config_service.go internal/service/config_service_test.go
go test ./internal/service
~~~

Expected: all service tests pass.

- [ ] **Step 6: Commit**

~~~bash
git add internal/service/config_service.go internal/service/config_service_test.go
git commit -m "feat: enforce config validation before writes"
~~~

### Task 3: Handler Error Classification

**Files:**
- Modify: `internal/handler/config_center.go`
- Create: `internal/handler/config_center_test.go`

- [ ] **Step 1: Write the failing classification test**

Create `internal/handler/config_center_test.go`:

~~~go
package handler

import (
    "errors"
    "testing"

    "github.com/dysodeng/etcd-manager/internal/service"
)

func TestConfigWriteErrorCode(t *testing.T) {
    validationErr := &service.ConfigValidationError{
        Format: "YAML",
        Err: errors.New("line 2: unexpected end"),
    }
    if got := configWriteErrorCode(validationErr); got != CodeParamInvalid {
        t.Fatalf("validation code = %d, want %d", got, CodeParamInvalid)
    }
    if got := configWriteErrorCode(errors.New("etcd unavailable")); got != CodeEtcdOpFailed {
        t.Fatalf("generic code = %d, want %d", got, CodeEtcdOpFailed)
    }
}
~~~

- [ ] **Step 2: Run the test and verify RED**

Run: `go test ./internal/handler -run '^TestConfigWriteErrorCode$'`

Expected: build fails because `configWriteErrorCode` is undefined.

- [ ] **Step 3: Implement typed classification**

Add `errors` to the handler imports and add:

~~~go
func configWriteErrorCode(err error) int {
    var validationErr *service.ConfigValidationError
    if errors.As(err, &validationErr) {
        return CodeParamInvalid
    }
    return CodeEtcdOpFailed
}
~~~

For Create, preserve the key-exists branch and use `Fail(c, configWriteErrorCode(err), err.Error())` in its fallback. Use the same call for Update. For Rollback, preserve the revision-not-found branch and use the classified code in its fallback.

- [ ] **Step 4: Format and verify GREEN**

Run:

~~~bash
gofmt -w internal/handler/config_center.go internal/handler/config_center_test.go
go test ./internal/handler
~~~

Expected: all handler tests pass.

- [ ] **Step 5: Commit**

~~~bash
git add internal/handler/config_center.go internal/handler/config_center_test.go
git commit -m "fix: report invalid config as parameter error"
~~~

### Task 4: Full Verification

**Files:**
- Verify all files changed in Tasks 1-3.

- [ ] **Step 1: Verify formatting**

Run: `test -z "$(gofmt -l internal/service/config_validator.go internal/service/config_validator_test.go internal/service/config_service.go internal/service/config_service_test.go internal/handler/config_center.go internal/handler/config_center_test.go)"`

Expected: exit code 0 with no output.

- [ ] **Step 2: Normalize dependencies and check diffs**

Run:

~~~bash
go mod tidy
git diff --check
~~~

Expected: module metadata is consistent and no whitespace errors are reported.

- [ ] **Step 3: Run all Go tests**

Run: `go test ./...`

Expected: every package passes.

- [ ] **Step 4: Run static analysis**

Run: `go vet ./...`

Expected: exit code 0 with no diagnostics.

- [ ] **Step 5: Review final scope**

Run:

~~~bash
git status --short
git diff --stat HEAD~3..HEAD
~~~

Expected: only the user's pre-existing `.gitignore` modification remains unstaged; implementation changes are limited to validation, integration, tests, and dependency metadata.
