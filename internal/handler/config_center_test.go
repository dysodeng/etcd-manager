package handler

import (
	"errors"
	"testing"

	"github.com/dysodeng/etcd-manager/internal/service"
)

func TestConfigWriteErrorCode(t *testing.T) {
	validationErr := &service.ConfigValidationError{
		Format: "YAML",
		Err:    errors.New("line 2: unexpected end"),
	}
	if got := configWriteErrorCode(validationErr); got != CodeParamInvalid {
		t.Fatalf("validation code = %d, want %d", got, CodeParamInvalid)
	}
	if got := configWriteErrorCode(errors.New("etcd unavailable")); got != CodeEtcdOpFailed {
		t.Fatalf("generic code = %d, want %d", got, CodeEtcdOpFailed)
	}
}
