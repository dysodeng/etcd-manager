package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/service"
)

func TestConfigWriteErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{name: "validation", err: &service.ConfigValidationError{Format: "YAML", Err: errors.New("line 2: unexpected end")}, code: CodeParamInvalid},
		{name: "key exists", err: service.ErrKeyExists, code: CodeKeyExists},
		{name: "key not found", err: service.ErrKeyNotFound, code: CodeKeyNotFound},
		{name: "conflict", err: service.ErrConfigConflict, code: CodeConfigConflict},
		{name: "list limit", err: service.ErrConfigListLimitExceeded, code: CodeConfigLimitExceeded},
		{name: "forbidden", err: domain.ErrEnvironmentForbidden, code: CodeForbidden},
		{name: "revision not found", err: service.ErrRevisionNotFound, code: CodeRevisionNotFound},
		{name: "persistence compensated", err: &service.ConfigPersistenceError{Err: errors.New("db"), Compensated: true}, code: CodeInternalError},
		{name: "inconsistent", err: &service.ConfigPersistenceError{Err: errors.New("db"), CompensationErr: errors.New("etcd")}, code: CodeInternalError},
		{name: "generic", err: errors.New("etcd unavailable"), code: CodeEtcdOpFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configWriteErrorCode(tt.err); got != tt.code {
				t.Fatalf("code = %d, want %d", got, tt.code)
			}
		})
	}
}

func TestReadConfigImportBodyAcceptsMaximumSize(t *testing.T) {
	body := strings.Repeat("x", int(MaxConfigImportBytes))
	request := httptest.NewRequest("POST", "/api/v1/configs/import", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	got, err := readConfigImportBody(recorder, request)

	if err != nil {
		t.Fatalf("readConfigImportBody() error = %v", err)
	}
	if len(got) != len(body) {
		t.Fatalf("body length = %d, want %d", len(got), len(body))
	}
}

func TestReadConfigImportBodyRejectsOversizedBody(t *testing.T) {
	request := httptest.NewRequest(
		"POST",
		"/api/v1/configs/import",
		strings.NewReader(strings.Repeat("x", int(MaxConfigImportBytes)+1)),
	)
	recorder := httptest.NewRecorder()

	_, err := readConfigImportBody(recorder, request)

	var maxBytesErr *http.MaxBytesError
	if !errors.As(err, &maxBytesErr) {
		t.Fatalf("error = %v, want *http.MaxBytesError", err)
	}
}
