package agent

import (
	"strings"
	"testing"
)

func TestPlan_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid plan with single step",
			plan: &Plan{
				Goal: "Create a hello world file",
				Steps: []PlanStep{
					{
						Description: "Write hello.txt",
						Action:      "write_file",
						Parameters: map[string]interface{}{
							"path":    "hello.txt",
							"content": "Hello, World!",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid plan with multiple steps",
			plan: &Plan{
				Goal: "Build a calculator",
				Steps: []PlanStep{
					{
						Description: "Create main file",
						Action:      "write_file",
						Parameters: map[string]interface{}{
							"path": "main.go",
						},
					},
					{
						Description: "Create test file",
						Action:      "write_file",
						Parameters: map[string]interface{}{
							"path": "main_test.go",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid plan with step without parameters",
			plan: &Plan{
				Goal: "Read a file",
				Steps: []PlanStep{
					{
						Description: "Read the config",
						Action:      "read_file",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil plan",
			plan:    nil,
			wantErr: true,
			errMsg:  "cannot serialize nil plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.plan.ToJSON()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ToJSON() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ToJSON() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ToJSON() unexpected error: %v", err)
				return
			}

			if got == "" {
				t.Error("ToJSON() returned empty string for valid plan")
			}
		})
	}
}

func TestParsePlan(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid plan JSON",
			json: `{
				"goal": "Create a hello world file",
				"steps": [
					{
						"description": "Write hello.txt",
						"action": "write_file",
						"parameters": {"path": "hello.txt"}
					}
				]
			}`,
			wantErr: false,
		},
		{
			name: "valid plan with multiple steps",
			json: `{
				"goal": "Build something",
				"steps": [
					{"description": "Step 1", "action": "action1"},
					{"description": "Step 2", "action": "action2"}
				]
			}`,
			wantErr: false,
		},
		{
			name:    "empty string",
			json:    "",
			wantErr: true,
			errMsg:  "cannot parse empty JSON string",
		},
		{
			name:    "invalid JSON",
			json:    "{invalid json}",
			wantErr: true,
			errMsg:  "failed to parse plan JSON",
		},
		{
			name:    "missing goal",
			json:    `{"steps": [{"description": "Step 1", "action": "action1"}]}`,
			wantErr: true,
			errMsg:  "missing required field: goal",
		},
		{
			name:    "empty goal",
			json:    `{"goal": "", "steps": [{"description": "Step 1", "action": "action1"}]}`,
			wantErr: true,
			errMsg:  "missing required field: goal",
		},
		{
			name:    "missing steps",
			json:    `{"goal": "Do something"}`,
			wantErr: true,
			errMsg:  "must have at least one step",
		},
		{
			name:    "empty steps array",
			json:    `{"goal": "Do something", "steps": []}`,
			wantErr: true,
			errMsg:  "must have at least one step",
		},
		{
			name:    "step missing description",
			json:    `{"goal": "Do something", "steps": [{"action": "action1"}]}`,
			wantErr: true,
			errMsg:  "step 1 is missing required field: description",
		},
		{
			name:    "step missing action",
			json:    `{"goal": "Do something", "steps": [{"description": "Step 1"}]}`,
			wantErr: true,
			errMsg:  "step 1 is missing required field: action",
		},
		{
			name:    "second step missing description",
			json:    `{"goal": "Do something", "steps": [{"description": "Step 1", "action": "a1"}, {"action": "a2"}]}`,
			wantErr: true,
			errMsg:  "step 2 is missing required field: description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePlan(tt.json)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePlan() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ParsePlan() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ParsePlan() unexpected error: %v", err)
				return
			}

			if got == nil {
				t.Error("ParsePlan() returned nil for valid JSON")
				return
			}

			if got.Goal == "" {
				t.Error("ParsePlan() returned plan with empty goal")
			}

			if len(got.Steps) == 0 {
				t.Error("ParsePlan() returned plan with no steps")
			}
		})
	}
}

func TestPlan_RoundTrip(t *testing.T) {
	original := &Plan{
		Goal: "Test round-trip serialization",
		Steps: []PlanStep{
			{
				Description: "First step",
				Action:      "action1",
				Parameters: map[string]interface{}{
					"key1": "value1",
					"key2": float64(42),
				},
			},
			{
				Description: "Second step",
				Action:      "action2",
			},
		},
	}

	// Serialize
	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	// Deserialize
	parsed, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("ParsePlan() failed: %v", err)
	}

	// Verify
	if parsed.Goal != original.Goal {
		t.Errorf("Goal mismatch: got %q, want %q", parsed.Goal, original.Goal)
	}

	if len(parsed.Steps) != len(original.Steps) {
		t.Fatalf("Steps count mismatch: got %d, want %d", len(parsed.Steps), len(original.Steps))
	}

	for i, step := range parsed.Steps {
		origStep := original.Steps[i]
		if step.Description != origStep.Description {
			t.Errorf("Step %d description mismatch: got %q, want %q", i, step.Description, origStep.Description)
		}
		if step.Action != origStep.Action {
			t.Errorf("Step %d action mismatch: got %q, want %q", i, step.Action, origStep.Action)
		}
	}
}
