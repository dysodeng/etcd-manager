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
	Err    error
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
