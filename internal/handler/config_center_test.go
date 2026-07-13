package handler

import (
	"errors"
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
