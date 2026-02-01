// Package agent implements the core agent loop and specialized agents.
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
)

// PlanStep represents a single step in a plan.
type PlanStep struct {
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Plan represents a structured plan created by the Architect agent.
type Plan struct {
	Goal  string     `json:"goal"`
	Steps []PlanStep `json:"steps"`
}

// ToJSON serializes the Plan to a JSON string.
func (p *Plan) ToJSON() (string, error) {
	if p == nil {
		return "", errors.New("cannot serialize nil plan")
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize plan: %w", err)
	}

	return string(data), nil
}

// ParsePlan deserializes a JSON string into a Plan.
func ParsePlan(jsonStr string) (*Plan, error) {
	if jsonStr == "" {
		return nil, errors.New("cannot parse empty JSON string")
	}

	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	// Validate required fields
	if plan.Goal == "" {
		return nil, errors.New("plan is missing required field: goal")
	}

	if len(plan.Steps) == 0 {
		return nil, errors.New("plan must have at least one step")
	}

	for i, step := range plan.Steps {
		if step.Description == "" {
			return nil, fmt.Errorf("step %d is missing required field: description", i+1)
		}
		if step.Action == "" {
			return nil, fmt.Errorf("step %d is missing required field: action", i+1)
		}
		// Normalize nil parameters to empty map for consistency
		if plan.Steps[i].Parameters == nil {
			plan.Steps[i].Parameters = make(map[string]interface{})
		}
	}

	return &plan, nil
}
