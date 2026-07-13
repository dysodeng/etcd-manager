package service

import (
	"errors"
	"fmt"
)

var (
	ErrKeyExists          = errors.New("key already exists")
	ErrKeyNotFound        = errors.New("key not found")
	ErrConfigConflict     = errors.New("configuration changed concurrently")
	ErrConfigPersistence  = errors.New("configuration revision persistence failed")
	ErrConfigInconsistent = errors.New("configuration state may be inconsistent")
	ErrRevisionNotFound   = errors.New("revision not found")
)

type ConfigPersistenceError struct {
	Operation       string
	Err             error
	Compensated     bool
	CompensationErr error
}

func (e *ConfigPersistenceError) Error() string {
	if e.Compensated {
		return fmt.Sprintf("%s: revision persistence failed; etcd change compensated: %v", e.Operation, e.Err)
	}
	if e.CompensationErr != nil {
		return fmt.Sprintf("%s: revision persistence failed and compensation errored: %v: %v", e.Operation, e.Err, e.CompensationErr)
	}
	return fmt.Sprintf("%s: revision persistence failed and compensation comparison failed: %v", e.Operation, e.Err)
}

func (e *ConfigPersistenceError) Unwrap() error {
	return e.Err
}

func (e *ConfigPersistenceError) Is(target error) bool {
	switch target {
	case ErrConfigInconsistent:
		return e.CompensationErr != nil || !e.Compensated
	case ErrConfigPersistence:
		return e.Compensated && e.CompensationErr == nil
	default:
		return false
	}
}
