package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	// ErrNotFound indicates the config file does not exist.
	ErrNotFound = errors.New("config file not found")

	// ErrInvalidVersion indicates an unsupported schema version.
	ErrInvalidVersion = errors.New("unsupported config version")

	// ErrInvalidType indicates an unexpected config type.
	ErrInvalidType = errors.New("invalid config type")

	// ErrMissingField indicates a required field is missing.
	ErrMissingField = errors.New("missing required field")
)

// LoadTownConfig loads and validates a town configuration file.
func LoadTownConfig(path string) (*TownConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var config TownConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := validateTownConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveTownConfig saves a town configuration to a file.
func SaveTownConfig(path string, config *TownConfig) error {
	if err := validateTownConfig(config); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// LoadRigsConfig loads and validates a rigs registry file.
func LoadRigsConfig(path string) (*RigsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var config RigsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := validateRigsConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveRigsConfig saves a rigs registry to a file.
func SaveRigsConfig(path string, config *RigsConfig) error {
	if err := validateRigsConfig(config); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// LoadAgentState loads an agent state file.
func LoadAgentState(path string) (*AgentState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	if err := validateAgentState(&state); err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveAgentState saves an agent state to a file.
func SaveAgentState(path string, state *AgentState) error {
	if err := validateAgentState(state); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}

	return nil
}

// validateTownConfig validates a TownConfig.
func validateTownConfig(c *TownConfig) error {
	if c.Type != "town" && c.Type != "" {
		return fmt.Errorf("%w: expected type 'town', got '%s'", ErrInvalidType, c.Type)
	}
	if c.Version > CurrentTownVersion {
		return fmt.Errorf("%w: got %d, max supported %d", ErrInvalidVersion, c.Version, CurrentTownVersion)
	}
	if c.Name == "" {
		return fmt.Errorf("%w: name", ErrMissingField)
	}
	return nil
}

// validateRigsConfig validates a RigsConfig.
func validateRigsConfig(c *RigsConfig) error {
	if c.Version > CurrentRigsVersion {
		return fmt.Errorf("%w: got %d, max supported %d", ErrInvalidVersion, c.Version, CurrentRigsVersion)
	}
	if c.Rigs == nil {
		c.Rigs = make(map[string]RigEntry)
	}
	return nil
}

// validateAgentState validates an AgentState.
func validateAgentState(s *AgentState) error {
	if s.Role == "" {
		return fmt.Errorf("%w: role", ErrMissingField)
	}
	return nil
}

// LoadRigConfig loads and validates a rig configuration file.
func LoadRigConfig(path string) (*RigConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var config RigConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := validateRigConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveRigConfig saves a rig configuration to a file.
func SaveRigConfig(path string, config *RigConfig) error {
	if err := validateRigConfig(config); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// validateRigConfig validates a RigConfig.
func validateRigConfig(c *RigConfig) error {
	if c.Type != "rig" && c.Type != "" {
		return fmt.Errorf("%w: expected type 'rig', got '%s'", ErrInvalidType, c.Type)
	}
	if c.Version > CurrentRigConfigVersion {
		return fmt.Errorf("%w: got %d, max supported %d", ErrInvalidVersion, c.Version, CurrentRigConfigVersion)
	}

	// Validate merge queue config if present
	if c.MergeQueue != nil {
		if err := validateMergeQueueConfig(c.MergeQueue); err != nil {
			return err
		}
	}

	return nil
}

// ErrInvalidOnConflict indicates an invalid on_conflict strategy.
var ErrInvalidOnConflict = errors.New("invalid on_conflict strategy")

// validateMergeQueueConfig validates a MergeQueueConfig.
func validateMergeQueueConfig(c *MergeQueueConfig) error {
	// Validate on_conflict strategy
	if c.OnConflict != "" && c.OnConflict != OnConflictAssignBack && c.OnConflict != OnConflictAutoRebase {
		return fmt.Errorf("%w: got '%s', want '%s' or '%s'",
			ErrInvalidOnConflict, c.OnConflict, OnConflictAssignBack, OnConflictAutoRebase)
	}

	// Validate poll_interval if specified
	if c.PollInterval != "" {
		if _, err := time.ParseDuration(c.PollInterval); err != nil {
			return fmt.Errorf("invalid poll_interval: %w", err)
		}
	}

	// Validate non-negative values
	if c.RetryFlakyTests < 0 {
		return fmt.Errorf("%w: retry_flaky_tests must be non-negative", ErrMissingField)
	}
	if c.MaxConcurrent < 0 {
		return fmt.Errorf("%w: max_concurrent must be non-negative", ErrMissingField)
	}

	return nil
}

// NewRigConfig creates a new RigConfig with defaults.
func NewRigConfig() *RigConfig {
	return &RigConfig{
		Type:       "rig",
		Version:    CurrentRigConfigVersion,
		MergeQueue: DefaultMergeQueueConfig(),
	}
}
