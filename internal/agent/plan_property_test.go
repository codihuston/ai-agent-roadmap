package agent

import (
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Validates: Requirements 8.2, 8.3**
// Property 18: Plan JSON Round-Trip Preserves Data
// For any valid Plan object, serializing to JSON with ToJSON() and then parsing
// with ParsePlan() SHALL produce a Plan equivalent to the original.

// genNonEmptyAlphaString generates non-empty alphanumeric strings
// that are safe for JSON serialization.
func genNonEmptyAlphaString() gopter.Gen {
	return gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "a" // Ensure non-empty
		}
		return s
	})
}

// genPlanStep generates a valid PlanStep with empty parameters.
func genPlanStep() gopter.Gen {
	return gopter.CombineGens(
		genNonEmptyAlphaString(), // Description
		genNonEmptyAlphaString(), // Action
	).Map(func(values []interface{}) PlanStep {
		return PlanStep{
			Description: values[0].(string),
			Action:      values[1].(string),
			Parameters:  make(map[string]interface{}),
		}
	})
}

// genPlanStepWithParams generates a PlanStep with string parameters.
func genPlanStepWithParams() gopter.Gen {
	return gopter.CombineGens(
		genNonEmptyAlphaString(), // Description
		genNonEmptyAlphaString(), // Action
		gen.MapOf(genNonEmptyAlphaString(), genNonEmptyAlphaString()), // Parameters
	).Map(func(values []interface{}) PlanStep {
		params := make(map[string]interface{})
		for k, v := range values[2].(map[string]string) {
			params[k] = v
		}
		return PlanStep{
			Description: values[0].(string),
			Action:      values[1].(string),
			Parameters:  params,
		}
	})
}

// genStepCount generates a step count between 1 and 5.
func genStepCount() gopter.Gen {
	return gen.IntRange(1, 5)
}

// genPlan generates a valid Plan with 1-5 steps.
func genPlan() gopter.Gen {
	return gopter.CombineGens(
		genNonEmptyAlphaString(), // Goal
		genStepCount(),           // Number of steps
	).FlatMap(func(values interface{}) gopter.Gen {
		vals := values.([]interface{})
		goal := vals[0].(string)
		stepCount := vals[1].(int)

		// Generate exactly stepCount steps
		stepGens := make([]gopter.Gen, stepCount)
		for i := 0; i < stepCount; i++ {
			stepGens[i] = genPlanStep()
		}

		return gopter.CombineGens(stepGens...).Map(func(steps []interface{}) *Plan {
			planSteps := make([]PlanStep, len(steps))
			for i, s := range steps {
				planSteps[i] = s.(PlanStep)
			}
			return &Plan{
				Goal:  goal,
				Steps: planSteps,
			}
		})
	}, reflect.TypeOf(&Plan{}))
}

// genPlanWithParams generates a valid Plan with parameters in steps.
func genPlanWithParams() gopter.Gen {
	return gopter.CombineGens(
		genNonEmptyAlphaString(), // Goal
		genStepCount(),           // Number of steps
	).FlatMap(func(values interface{}) gopter.Gen {
		vals := values.([]interface{})
		goal := vals[0].(string)
		stepCount := vals[1].(int)

		// Generate exactly stepCount steps with params
		stepGens := make([]gopter.Gen, stepCount)
		for i := 0; i < stepCount; i++ {
			stepGens[i] = genPlanStepWithParams()
		}

		return gopter.CombineGens(stepGens...).Map(func(steps []interface{}) *Plan {
			planSteps := make([]PlanStep, len(steps))
			for i, s := range steps {
				planSteps[i] = s.(PlanStep)
			}
			return &Plan{
				Goal:  goal,
				Steps: planSteps,
			}
		})
	}, reflect.TypeOf(&Plan{}))
}

func TestProperty_PlanRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.Rng.Seed(42) // Reproducible tests

	properties := gopter.NewProperties(parameters)

	properties.Property("Plan round-trip preserves goal", prop.ForAll(
		func(plan *Plan) bool {
			jsonStr, err := plan.ToJSON()
			if err != nil {
				t.Logf("ToJSON error: %v", err)
				return false
			}

			parsed, err := ParsePlan(jsonStr)
			if err != nil {
				t.Logf("ParsePlan error: %v", err)
				return false
			}

			return parsed.Goal == plan.Goal
		},
		genPlan(),
	))

	properties.Property("Plan round-trip preserves step count", prop.ForAll(
		func(plan *Plan) bool {
			jsonStr, err := plan.ToJSON()
			if err != nil {
				return false
			}

			parsed, err := ParsePlan(jsonStr)
			if err != nil {
				return false
			}

			return len(parsed.Steps) == len(plan.Steps)
		},
		genPlan(),
	))

	properties.Property("Plan round-trip preserves step descriptions and actions", prop.ForAll(
		func(plan *Plan) bool {
			jsonStr, err := plan.ToJSON()
			if err != nil {
				return false
			}

			parsed, err := ParsePlan(jsonStr)
			if err != nil {
				return false
			}

			for i, step := range parsed.Steps {
				if step.Description != plan.Steps[i].Description {
					return false
				}
				if step.Action != plan.Steps[i].Action {
					return false
				}
			}

			return true
		},
		genPlan(),
	))

	properties.Property("Plan round-trip preserves step parameters", prop.ForAll(
		func(plan *Plan) bool {
			jsonStr, err := plan.ToJSON()
			if err != nil {
				return false
			}

			parsed, err := ParsePlan(jsonStr)
			if err != nil {
				return false
			}

			for i, step := range parsed.Steps {
				if !reflect.DeepEqual(step.Parameters, plan.Steps[i].Parameters) {
					return false
				}
			}

			return true
		},
		genPlanWithParams(),
	))

	properties.Property("Plan round-trip produces equivalent plan", prop.ForAll(
		func(plan *Plan) bool {
			jsonStr, err := plan.ToJSON()
			if err != nil {
				return false
			}

			parsed, err := ParsePlan(jsonStr)
			if err != nil {
				return false
			}

			// Check goal
			if parsed.Goal != plan.Goal {
				return false
			}

			// Check steps count
			if len(parsed.Steps) != len(plan.Steps) {
				return false
			}

			// Check each step
			for i, step := range parsed.Steps {
				orig := plan.Steps[i]
				if step.Description != orig.Description {
					return false
				}
				if step.Action != orig.Action {
					return false
				}
				if !reflect.DeepEqual(step.Parameters, orig.Parameters) {
					return false
				}
			}

			return true
		},
		genPlanWithParams(),
	))

	properties.TestingRun(t)
}
