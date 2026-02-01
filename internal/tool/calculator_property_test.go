package tool

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestCalculatorProperty_Correctness validates Property 7:
// "For any valid arithmetic operation (add, subtract, multiply, divide) with numeric
// operands a and b, the CalculatorTool SHALL return the mathematically correct result.
// Division by zero SHALL return an error."
//
// **Validates: Requirements 3.5**
func TestCalculatorProperty_Correctness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	calc := NewCalculatorTool()
	ctx := context.Background()

	// Property: Addition produces mathematically correct results
	properties.Property("add returns a + b", prop.ForAll(
		func(a, b float64) bool {
			args := map[string]interface{}{
				"operation": "add",
				"a":         a,
				"b":         b,
			}
			result, err := calc.Execute(ctx, args)
			if err != nil {
				return false
			}
			if !result.Success {
				return false
			}
			got, parseErr := strconv.ParseFloat(result.Output, 64)
			if parseErr != nil {
				return false
			}
			expected := a + b
			return floatEquals(got, expected)
		},
		genFiniteFloat(),
		genFiniteFloat(),
	))

	// Property: Subtraction produces mathematically correct results
	properties.Property("subtract returns a - b", prop.ForAll(
		func(a, b float64) bool {
			args := map[string]interface{}{
				"operation": "subtract",
				"a":         a,
				"b":         b,
			}
			result, err := calc.Execute(ctx, args)
			if err != nil {
				return false
			}
			if !result.Success {
				return false
			}
			got, parseErr := strconv.ParseFloat(result.Output, 64)
			if parseErr != nil {
				return false
			}
			expected := a - b
			return floatEquals(got, expected)
		},
		genFiniteFloat(),
		genFiniteFloat(),
	))

	// Property: Multiplication produces mathematically correct results
	properties.Property("multiply returns a * b", prop.ForAll(
		func(a, b float64) bool {
			args := map[string]interface{}{
				"operation": "multiply",
				"a":         a,
				"b":         b,
			}
			result, err := calc.Execute(ctx, args)
			if err != nil {
				return false
			}
			if !result.Success {
				return false
			}
			got, parseErr := strconv.ParseFloat(result.Output, 64)
			if parseErr != nil {
				return false
			}
			expected := a * b
			return floatEquals(got, expected)
		},
		genFiniteFloat(),
		genFiniteFloat(),
	))

	// Property: Division produces mathematically correct results for non-zero divisor
	properties.Property("divide returns a / b for non-zero b", prop.ForAll(
		func(a, b float64) bool {
			args := map[string]interface{}{
				"operation": "divide",
				"a":         a,
				"b":         b,
			}
			result, err := calc.Execute(ctx, args)
			if err != nil {
				return false
			}
			if !result.Success {
				return false
			}
			got, parseErr := strconv.ParseFloat(result.Output, 64)
			if parseErr != nil {
				return false
			}
			expected := a / b
			return floatEquals(got, expected)
		},
		genFiniteFloat(),
		genNonZeroFloat(),
	))

	// Property: Division by zero returns an error
	properties.Property("divide by zero returns error", prop.ForAll(
		func(a float64) bool {
			args := map[string]interface{}{
				"operation": "divide",
				"a":         a,
				"b":         0.0,
			}
			result, err := calc.Execute(ctx, args)
			if err != nil {
				return false
			}
			// Should fail with division by zero error
			return !result.Success && result.Error == "division by zero"
		},
		genFiniteFloat(),
	))

	properties.TestingRun(t)
}

// genFiniteFloat generates finite float64 values (no NaN or Inf).
func genFiniteFloat() gopter.Gen {
	return gen.Float64().SuchThat(func(f float64) bool {
		return !math.IsNaN(f) && !math.IsInf(f, 0)
	})
}

// genNonZeroFloat generates non-zero finite float64 values.
func genNonZeroFloat() gopter.Gen {
	return gen.Float64().SuchThat(func(f float64) bool {
		return f != 0 && !math.IsNaN(f) && !math.IsInf(f, 0)
	})
}

// floatEquals compares two float64 values with tolerance for floating point errors.
func floatEquals(a, b float64) bool {
	// Handle special cases
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsInf(a, 1) && math.IsInf(b, 1) {
		return true
	}
	if math.IsInf(a, -1) && math.IsInf(b, -1) {
		return true
	}

	// For very small numbers, use absolute tolerance
	const epsilon = 1e-10
	if math.Abs(a) < epsilon && math.Abs(b) < epsilon {
		return math.Abs(a-b) < epsilon
	}

	// For larger numbers, use relative tolerance
	const relTolerance = 1e-9
	diff := math.Abs(a - b)
	largest := math.Max(math.Abs(a), math.Abs(b))
	return diff <= largest*relTolerance
}

// TestCalculatorProperty_OperationCoverage ensures all operations are tested
func TestCalculatorProperty_OperationCoverage(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	calc := NewCalculatorTool()
	ctx := context.Background()

	operations := []string{"add", "subtract", "multiply", "divide"}

	// Property: All valid operations succeed with valid inputs
	properties.Property("all operations succeed with valid non-zero inputs", prop.ForAll(
		func(opIdx int, a, b float64) bool {
			op := operations[opIdx%len(operations)]
			args := map[string]interface{}{
				"operation": op,
				"a":         a,
				"b":         b,
			}
			result, err := calc.Execute(ctx, args)
			if err != nil {
				return false
			}
			return result.Success
		},
		gen.IntRange(0, 3),
		genFiniteFloat(),
		genNonZeroFloat(),
	))

	// Property: Invalid operations always fail
	properties.Property("invalid operations fail", prop.ForAll(
		func(op string, a, b float64) bool {
			args := map[string]interface{}{
				"operation": op,
				"a":         a,
				"b":         b,
			}
			result, err := calc.Execute(ctx, args)
			if err != nil {
				return false
			}
			return !result.Success
		},
		gen.OneConstOf("modulo", "power", "sqrt", "invalid", ""),
		genFiniteFloat(),
		genFiniteFloat(),
	))

	properties.TestingRun(t)
}

// TestCalculatorProperty_Commutativity tests mathematical properties
func TestCalculatorProperty_Commutativity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	calc := NewCalculatorTool()
	ctx := context.Background()

	// Property: Addition is commutative (a + b = b + a)
	properties.Property("addition is commutative", prop.ForAll(
		func(a, b float64) bool {
			args1 := map[string]interface{}{"operation": "add", "a": a, "b": b}
			args2 := map[string]interface{}{"operation": "add", "a": b, "b": a}

			result1, _ := calc.Execute(ctx, args1)
			result2, _ := calc.Execute(ctx, args2)

			if !result1.Success || !result2.Success {
				return false
			}

			v1, _ := strconv.ParseFloat(result1.Output, 64)
			v2, _ := strconv.ParseFloat(result2.Output, 64)
			return floatEquals(v1, v2)
		},
		genFiniteFloat(),
		genFiniteFloat(),
	))

	// Property: Multiplication is commutative (a * b = b * a)
	properties.Property("multiplication is commutative", prop.ForAll(
		func(a, b float64) bool {
			args1 := map[string]interface{}{"operation": "multiply", "a": a, "b": b}
			args2 := map[string]interface{}{"operation": "multiply", "a": b, "b": a}

			result1, _ := calc.Execute(ctx, args1)
			result2, _ := calc.Execute(ctx, args2)

			if !result1.Success || !result2.Success {
				return false
			}

			v1, _ := strconv.ParseFloat(result1.Output, 64)
			v2, _ := strconv.ParseFloat(result2.Output, 64)
			return floatEquals(v1, v2)
		},
		genFiniteFloat(),
		genFiniteFloat(),
	))

	properties.TestingRun(t)
}

func init() {
	// Suppress unused import warning
	_ = fmt.Sprint
}
