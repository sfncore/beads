package beads

import "testing"

func TestBuiltinMolecules(t *testing.T) {
	molecules := BuiltinMolecules()

	if len(molecules) != 14 {
		t.Errorf("expected 14 built-in molecules, got %d", len(molecules))
	}

	// Verify each molecule can be parsed and validated
	for _, mol := range molecules {
		t.Run(mol.Title, func(t *testing.T) {
			// Check required fields
			if mol.ID == "" {
				t.Error("molecule missing ID")
			}
			if mol.Title == "" {
				t.Error("molecule missing Title")
			}
			if mol.Description == "" {
				t.Error("molecule missing Description")
			}

			// Parse the molecule steps
			steps, err := ParseMoleculeSteps(mol.Description)
			if err != nil {
				t.Fatalf("failed to parse molecule steps: %v", err)
			}

			if len(steps) == 0 {
				t.Error("molecule has no steps")
			}

			// Validate the molecule as if it were an issue
			issue := &Issue{
				Type:        "molecule",
				Title:       mol.Title,
				Description: mol.Description,
			}

			if err := ValidateMolecule(issue); err != nil {
				t.Errorf("molecule validation failed: %v", err)
			}
		})
	}
}

func TestEngineerInBoxMolecule(t *testing.T) {
	mol := EngineerInBoxMolecule()

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 5 steps: design, implement, review, test, submit
	if len(steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(steps))
	}

	// Verify step refs
	expectedRefs := []string{"design", "implement", "review", "test", "submit"}
	for i, expected := range expectedRefs {
		if steps[i].Ref != expected {
			t.Errorf("step %d: expected ref %q, got %q", i, expected, steps[i].Ref)
		}
	}

	// Verify dependencies
	// design has no deps
	if len(steps[0].Needs) != 0 {
		t.Errorf("design should have no deps, got %v", steps[0].Needs)
	}

	// implement needs design
	if len(steps[1].Needs) != 1 || steps[1].Needs[0] != "design" {
		t.Errorf("implement should need design, got %v", steps[1].Needs)
	}

	// review needs implement
	if len(steps[2].Needs) != 1 || steps[2].Needs[0] != "implement" {
		t.Errorf("review should need implement, got %v", steps[2].Needs)
	}

	// test needs implement
	if len(steps[3].Needs) != 1 || steps[3].Needs[0] != "implement" {
		t.Errorf("test should need implement, got %v", steps[3].Needs)
	}

	// submit needs review and test
	if len(steps[4].Needs) != 2 {
		t.Errorf("submit should need 2 deps, got %v", steps[4].Needs)
	}
}

func TestQuickFixMolecule(t *testing.T) {
	mol := QuickFixMolecule()

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 3 steps: implement, test, submit
	if len(steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(steps))
	}

	expectedRefs := []string{"implement", "test", "submit"}
	for i, expected := range expectedRefs {
		if steps[i].Ref != expected {
			t.Errorf("step %d: expected ref %q, got %q", i, expected, steps[i].Ref)
		}
	}
}

func TestResearchMolecule(t *testing.T) {
	mol := ResearchMolecule()

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 2 steps: investigate, document
	if len(steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(steps))
	}

	expectedRefs := []string{"investigate", "document"}
	for i, expected := range expectedRefs {
		if steps[i].Ref != expected {
			t.Errorf("step %d: expected ref %q, got %q", i, expected, steps[i].Ref)
		}
	}

	// document needs investigate
	if len(steps[1].Needs) != 1 || steps[1].Needs[0] != "investigate" {
		t.Errorf("document should need investigate, got %v", steps[1].Needs)
	}
}

func TestInstallGoBinaryMolecule(t *testing.T) {
	mol := InstallGoBinaryMolecule()

	if mol.ID != "mol-install-go-binary" {
		t.Errorf("expected ID 'mol-install-go-binary', got %q", mol.ID)
	}

	if mol.Title != "Install Go Binary" {
		t.Errorf("expected Title 'Install Go Binary', got %q", mol.Title)
	}

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 1 step: install
	if len(steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(steps))
	}

	if steps[0].Ref != "install" {
		t.Errorf("expected ref 'install', got %q", steps[0].Ref)
	}

	// install has no deps
	if len(steps[0].Needs) != 0 {
		t.Errorf("install should have no deps, got %v", steps[0].Needs)
	}
}

func TestPolecatWorkMolecule(t *testing.T) {
	mol := PolecatWorkMolecule()

	if mol.ID != "mol-polecat-work" {
		t.Errorf("expected ID 'mol-polecat-work', got %q", mol.ID)
	}

	if mol.Title != "Polecat Work" {
		t.Errorf("expected Title 'Polecat Work', got %q", mol.Title)
	}

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 8 steps: load-context, implement, self-review, verify-tests,
	// rebase-main, submit-merge, exit-decision, request-shutdown
	if len(steps) != 8 {
		t.Errorf("expected 8 steps, got %d", len(steps))
	}

	expectedRefs := []string{
		"load-context", "implement", "self-review", "verify-tests",
		"rebase-main", "submit-merge", "exit-decision", "request-shutdown",
	}
	for i, expected := range expectedRefs {
		if i >= len(steps) {
			t.Errorf("missing step %d: expected %q", i, expected)
			continue
		}
		if steps[i].Ref != expected {
			t.Errorf("step %d: expected ref %q, got %q", i, expected, steps[i].Ref)
		}
	}

	// Verify key dependencies
	// load-context has no deps
	if len(steps[0].Needs) != 0 {
		t.Errorf("load-context should have no deps, got %v", steps[0].Needs)
	}

	// implement needs load-context
	if len(steps[1].Needs) != 1 || steps[1].Needs[0] != "load-context" {
		t.Errorf("implement should need load-context, got %v", steps[1].Needs)
	}

	// rebase-main needs self-review and verify-tests
	if len(steps[4].Needs) != 2 {
		t.Errorf("rebase-main should need 2 deps, got %v", steps[4].Needs)
	}

	// request-shutdown needs exit-decision
	if len(steps[7].Needs) != 1 || steps[7].Needs[0] != "exit-decision" {
		t.Errorf("request-shutdown should need exit-decision, got %v", steps[7].Needs)
	}
}

func TestDeaconPatrolMolecule(t *testing.T) {
	mol := DeaconPatrolMolecule()

	if mol.ID != "mol-deacon-patrol" {
		t.Errorf("expected ID 'mol-deacon-patrol', got %q", mol.ID)
	}

	if mol.Title != "Deacon Patrol" {
		t.Errorf("expected Title 'Deacon Patrol', got %q", mol.Title)
	}

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 8 steps: inbox-check, trigger-pending-spawns, health-scan, plugin-run,
	// orphan-check, session-gc, context-check, loop-or-exit
	if len(steps) != 8 {
		t.Errorf("expected 8 steps, got %d", len(steps))
	}

	expectedRefs := []string{
		"inbox-check", "trigger-pending-spawns", "health-scan", "plugin-run",
		"orphan-check", "session-gc", "context-check", "loop-or-exit",
	}
	for i, expected := range expectedRefs {
		if i >= len(steps) {
			t.Errorf("missing step %d: expected %q", i, expected)
			continue
		}
		if steps[i].Ref != expected {
			t.Errorf("step %d: expected ref %q, got %q", i, expected, steps[i].Ref)
		}
	}

	// Verify key dependencies
	// inbox-check has no deps (first step)
	if len(steps[0].Needs) != 0 {
		t.Errorf("inbox-check should have no deps, got %v", steps[0].Needs)
	}

	// trigger-pending-spawns needs inbox-check
	if len(steps[1].Needs) != 1 || steps[1].Needs[0] != "inbox-check" {
		t.Errorf("trigger-pending-spawns should need inbox-check, got %v", steps[1].Needs)
	}

	// loop-or-exit needs context-check
	if len(steps[7].Needs) != 1 || steps[7].Needs[0] != "context-check" {
		t.Errorf("loop-or-exit should need context-check, got %v", steps[7].Needs)
	}
}

func TestWitnessPatrolMolecule(t *testing.T) {
	mol := WitnessPatrolMolecule()

	if mol.ID != "mol-witness-patrol" {
		t.Errorf("expected ID 'mol-witness-patrol', got %q", mol.ID)
	}

	if mol.Title != "Witness Patrol" {
		t.Errorf("expected Title 'Witness Patrol', got %q", mol.Title)
	}

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 9 steps using Christmas Ornament pattern:
	// PREFLIGHT: inbox-check, check-refinery, load-state
	// DISCOVERY: survey-workers (bonds mol-polecat-arm dynamically)
	// CLEANUP: aggregate, save-state, generate-summary, context-check, burn-or-loop
	if len(steps) != 9 {
		t.Errorf("expected 9 steps, got %d", len(steps))
	}

	expectedRefs := []string{
		"inbox-check", "check-refinery", "load-state", "survey-workers",
		"aggregate", "save-state", "generate-summary", "context-check", "burn-or-loop",
	}
	for i, expected := range expectedRefs {
		if i >= len(steps) {
			t.Errorf("missing step %d: expected %q", i, expected)
			continue
		}
		if steps[i].Ref != expected {
			t.Errorf("step %d: expected ref %q, got %q", i, expected, steps[i].Ref)
		}
	}

	// Verify key dependencies
	// inbox-check has no deps (first step)
	if len(steps[0].Needs) != 0 {
		t.Errorf("inbox-check should have no deps, got %v", steps[0].Needs)
	}

	// check-refinery needs inbox-check
	if len(steps[1].Needs) != 1 || steps[1].Needs[0] != "inbox-check" {
		t.Errorf("check-refinery should need inbox-check, got %v", steps[1].Needs)
	}

	// load-state needs check-refinery
	if len(steps[2].Needs) != 1 || steps[2].Needs[0] != "check-refinery" {
		t.Errorf("load-state should need check-refinery, got %v", steps[2].Needs)
	}

	// aggregate needs survey-workers (fanout gate)
	if len(steps[4].Needs) != 1 || steps[4].Needs[0] != "survey-workers" {
		t.Errorf("aggregate should need survey-workers, got %v", steps[4].Needs)
	}

	// aggregate should have WaitsFor: all-children
	if len(steps[4].WaitsFor) != 1 || steps[4].WaitsFor[0] != "all-children" {
		t.Errorf("aggregate should WaitsFor all-children, got %v", steps[4].WaitsFor)
	}

	// burn-or-loop needs context-check
	if len(steps[8].Needs) != 1 || steps[8].Needs[0] != "context-check" {
		t.Errorf("burn-or-loop should need context-check, got %v", steps[8].Needs)
	}
}

func TestPolecatArmMolecule(t *testing.T) {
	mol := PolecatArmMolecule()

	if mol.ID != "mol-polecat-arm" {
		t.Errorf("expected ID 'mol-polecat-arm', got %q", mol.ID)
	}

	if mol.Title != "Polecat Arm" {
		t.Errorf("expected Title 'Polecat Arm', got %q", mol.Title)
	}

	steps, err := ParseMoleculeSteps(mol.Description)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Should have 5 steps: capture, assess, load-history, decide, execute
	if len(steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(steps))
	}

	expectedRefs := []string{"capture", "assess", "load-history", "decide", "execute"}
	for i, expected := range expectedRefs {
		if i >= len(steps) {
			t.Errorf("missing step %d: expected %q", i, expected)
			continue
		}
		if steps[i].Ref != expected {
			t.Errorf("step %d: expected ref %q, got %q", i, expected, steps[i].Ref)
		}
	}

	// Verify dependencies form a chain
	// capture has no deps (first step)
	if len(steps[0].Needs) != 0 {
		t.Errorf("capture should have no deps, got %v", steps[0].Needs)
	}

	// assess needs capture
	if len(steps[1].Needs) != 1 || steps[1].Needs[0] != "capture" {
		t.Errorf("assess should need capture, got %v", steps[1].Needs)
	}

	// load-history needs assess
	if len(steps[2].Needs) != 1 || steps[2].Needs[0] != "assess" {
		t.Errorf("load-history should need assess, got %v", steps[2].Needs)
	}

	// decide needs load-history
	if len(steps[3].Needs) != 1 || steps[3].Needs[0] != "load-history" {
		t.Errorf("decide should need load-history, got %v", steps[3].Needs)
	}

	// execute needs decide
	if len(steps[4].Needs) != 1 || steps[4].Needs[0] != "decide" {
		t.Errorf("execute should need decide, got %v", steps[4].Needs)
	}
}
