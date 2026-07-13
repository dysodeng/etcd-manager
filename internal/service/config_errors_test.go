package service

import (
	"errors"
	"testing"
)

func TestConfigConsistencyErrorsSupportErrorsIs(t *testing.T) {
	persistenceErr := errors.New("database unavailable")
	compensated := &ConfigPersistenceError{
		Operation:   "update",
		Err:         persistenceErr,
		Compensated: true,
	}
	if !errors.Is(compensated, ErrConfigPersistence) {
		t.Fatalf("error = %v, want ErrConfigPersistence", compensated)
	}
	if !errors.Is(compensated, persistenceErr) {
		t.Fatalf("error = %v, want wrapped persistence error", compensated)
	}

	compensationErr := errors.New("etcd unavailable")
	inconsistent := &ConfigPersistenceError{
		Operation:       "update",
		Err:             persistenceErr,
		CompensationErr: compensationErr,
	}
	if !errors.Is(inconsistent, ErrConfigInconsistent) {
		t.Fatalf("error = %v, want ErrConfigInconsistent", inconsistent)
	}

	compareMiss := &ConfigPersistenceError{
		Operation: "update",
		Err:       persistenceErr,
	}
	if !errors.Is(compareMiss, ErrConfigInconsistent) {
		t.Fatalf("error = %v, want ErrConfigInconsistent", compareMiss)
	}
}
