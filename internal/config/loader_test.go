package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTownConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mayor", "town.json")

	original := &TownConfig{
		Type:      "town",
		Version:   1,
		Name:      "test-town",
		CreatedAt: time.Now().Truncate(time.Second),
	}

	if err := SaveTownConfig(path, original); err != nil {
		t.Fatalf("SaveTownConfig: %v", err)
	}

	loaded, err := LoadTownConfig(path)
	if err != nil {
		t.Fatalf("LoadTownConfig: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}
	if loaded.Type != original.Type {
		t.Errorf("Type = %q, want %q", loaded.Type, original.Type)
	}
}

func TestRigsConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mayor", "rigs.json")

	original := &RigsConfig{
		Version: 1,
		Rigs: map[string]RigEntry{
			"gastown": {
				GitURL:  "git@github.com:steveyegge/gastown.git",
				AddedAt: time.Now().Truncate(time.Second),
				BeadsConfig: &BeadsConfig{
					Repo:   "local",
					Prefix: "gt-",
				},
			},
		},
	}

	if err := SaveRigsConfig(path, original); err != nil {
		t.Fatalf("SaveRigsConfig: %v", err)
	}

	loaded, err := LoadRigsConfig(path)
	if err != nil {
		t.Fatalf("LoadRigsConfig: %v", err)
	}

	if len(loaded.Rigs) != 1 {
		t.Errorf("Rigs count = %d, want 1", len(loaded.Rigs))
	}

	rig, ok := loaded.Rigs["gastown"]
	if !ok {
		t.Fatal("missing 'gastown' rig")
	}
	if rig.BeadsConfig == nil || rig.BeadsConfig.Prefix != "gt-" {
		t.Errorf("BeadsConfig.Prefix = %v, want 'gt-'", rig.BeadsConfig)
	}
}

func TestAgentStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	original := &AgentState{
		Role:       "mayor",
		LastActive: time.Now().Truncate(time.Second),
		Session:    "abc123",
		Extra: map[string]any{
			"custom": "value",
		},
	}

	if err := SaveAgentState(path, original); err != nil {
		t.Fatalf("SaveAgentState: %v", err)
	}

	loaded, err := LoadAgentState(path)
	if err != nil {
		t.Fatalf("LoadAgentState: %v", err)
	}

	if loaded.Role != original.Role {
		t.Errorf("Role = %q, want %q", loaded.Role, original.Role)
	}
	if loaded.Session != original.Session {
		t.Errorf("Session = %q, want %q", loaded.Session, original.Session)
	}
}

func TestLoadTownConfigNotFound(t *testing.T) {
	_, err := LoadTownConfig("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestValidationErrors(t *testing.T) {
	// Missing name
	tc := &TownConfig{Type: "town", Version: 1}
	if err := validateTownConfig(tc); err == nil {
		t.Error("expected error for missing name")
	}

	// Wrong type
	tc = &TownConfig{Type: "wrong", Version: 1, Name: "test"}
	if err := validateTownConfig(tc); err == nil {
		t.Error("expected error for wrong type")
	}

	// Missing role
	as := &AgentState{}
	if err := validateAgentState(as); err == nil {
		t.Error("expected error for missing role")
	}
}

func TestRigConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	original := NewRigConfig()

	if err := SaveRigConfig(path, original); err != nil {
		t.Fatalf("SaveRigConfig: %v", err)
	}

	loaded, err := LoadRigConfig(path)
	if err != nil {
		t.Fatalf("LoadRigConfig: %v", err)
	}

	if loaded.Type != "rig" {
		t.Errorf("Type = %q, want 'rig'", loaded.Type)
	}
	if loaded.Version != CurrentRigConfigVersion {
		t.Errorf("Version = %d, want %d", loaded.Version, CurrentRigConfigVersion)
	}
	if loaded.MergeQueue == nil {
		t.Fatal("MergeQueue is nil")
	}
	if !loaded.MergeQueue.Enabled {
		t.Error("MergeQueue.Enabled = false, want true")
	}
	if loaded.MergeQueue.TargetBranch != "main" {
		t.Errorf("MergeQueue.TargetBranch = %q, want 'main'", loaded.MergeQueue.TargetBranch)
	}
	if loaded.MergeQueue.OnConflict != OnConflictAssignBack {
		t.Errorf("MergeQueue.OnConflict = %q, want %q", loaded.MergeQueue.OnConflict, OnConflictAssignBack)
	}
}

func TestRigConfigWithCustomMergeQueue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	original := &RigConfig{
		Type:    "rig",
		Version: 1,
		MergeQueue: &MergeQueueConfig{
			Enabled:              true,
			TargetBranch:         "develop",
			IntegrationBranches:  false,
			OnConflict:           OnConflictAutoRebase,
			RunTests:             true,
			TestCommand:          "make test",
			DeleteMergedBranches: false,
			RetryFlakyTests:      3,
			PollInterval:         "1m",
			MaxConcurrent:        2,
		},
	}

	if err := SaveRigConfig(path, original); err != nil {
		t.Fatalf("SaveRigConfig: %v", err)
	}

	loaded, err := LoadRigConfig(path)
	if err != nil {
		t.Fatalf("LoadRigConfig: %v", err)
	}

	mq := loaded.MergeQueue
	if mq.TargetBranch != "develop" {
		t.Errorf("TargetBranch = %q, want 'develop'", mq.TargetBranch)
	}
	if mq.OnConflict != OnConflictAutoRebase {
		t.Errorf("OnConflict = %q, want %q", mq.OnConflict, OnConflictAutoRebase)
	}
	if mq.TestCommand != "make test" {
		t.Errorf("TestCommand = %q, want 'make test'", mq.TestCommand)
	}
	if mq.RetryFlakyTests != 3 {
		t.Errorf("RetryFlakyTests = %d, want 3", mq.RetryFlakyTests)
	}
	if mq.PollInterval != "1m" {
		t.Errorf("PollInterval = %q, want '1m'", mq.PollInterval)
	}
	if mq.MaxConcurrent != 2 {
		t.Errorf("MaxConcurrent = %d, want 2", mq.MaxConcurrent)
	}
}

func TestRigConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *RigConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &RigConfig{
				Type:       "rig",
				Version:    1,
				MergeQueue: DefaultMergeQueueConfig(),
			},
			wantErr: false,
		},
		{
			name: "valid config without merge queue",
			config: &RigConfig{
				Type:    "rig",
				Version: 1,
			},
			wantErr: false,
		},
		{
			name: "wrong type",
			config: &RigConfig{
				Type:    "wrong",
				Version: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid on_conflict",
			config: &RigConfig{
				Type:    "rig",
				Version: 1,
				MergeQueue: &MergeQueueConfig{
					OnConflict: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid poll_interval",
			config: &RigConfig{
				Type:    "rig",
				Version: 1,
				MergeQueue: &MergeQueueConfig{
					PollInterval: "not-a-duration",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRigConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRigConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultMergeQueueConfig(t *testing.T) {
	cfg := DefaultMergeQueueConfig()

	if !cfg.Enabled {
		t.Error("Enabled should be true by default")
	}
	if cfg.TargetBranch != "main" {
		t.Errorf("TargetBranch = %q, want 'main'", cfg.TargetBranch)
	}
	if !cfg.IntegrationBranches {
		t.Error("IntegrationBranches should be true by default")
	}
	if cfg.OnConflict != OnConflictAssignBack {
		t.Errorf("OnConflict = %q, want %q", cfg.OnConflict, OnConflictAssignBack)
	}
	if !cfg.RunTests {
		t.Error("RunTests should be true by default")
	}
	if cfg.TestCommand != "go test ./..." {
		t.Errorf("TestCommand = %q, want 'go test ./...'", cfg.TestCommand)
	}
	if !cfg.DeleteMergedBranches {
		t.Error("DeleteMergedBranches should be true by default")
	}
	if cfg.RetryFlakyTests != 1 {
		t.Errorf("RetryFlakyTests = %d, want 1", cfg.RetryFlakyTests)
	}
	if cfg.PollInterval != "30s" {
		t.Errorf("PollInterval = %q, want '30s'", cfg.PollInterval)
	}
	if cfg.MaxConcurrent != 1 {
		t.Errorf("MaxConcurrent = %d, want 1", cfg.MaxConcurrent)
	}
}

func TestLoadRigConfigNotFound(t *testing.T) {
	_, err := LoadRigConfig("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
