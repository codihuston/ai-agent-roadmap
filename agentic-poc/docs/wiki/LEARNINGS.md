# Learnings Document

This document captures design decisions, challenges, and observations encountered during the development of the agentic system POC.

## Table of Contents

1. [JSON Serialization](#json-serialization)
2. [Property-Based Testing](#property-based-testing)

---

## JSON Serialization

### Challenge: nil vs empty map in JSON round-trips

**Context**: When deserializing JSON into Go structs, optional fields like `Parameters map[string]interface{}` can end up as `nil` if not present in the JSON, but the original struct may have had an empty map `map[string]interface{}{}`.

**Problem**: `reflect.DeepEqual(nil, map[string]interface{}{})` returns `false`, causing round-trip tests to fail even though the semantic meaning is the same.

**Solution**: Normalize `nil` maps to empty maps during parsing:

```go
// In ParsePlan, after unmarshaling:
for i := range plan.Steps {
    if plan.Steps[i].Parameters == nil {
        plan.Steps[i].Parameters = make(map[string]interface{})
    }
}
```

**Trade-off**: This adds a small overhead but ensures consistent behavior and makes testing simpler.

---

## Property-Based Testing

### Challenge: Generating valid test data with gopter

**Context**: Using `gopter` for property-based testing requires careful generator construction to avoid high discard rates.

**Problem**: Using `SuchThat` filters on slice generators (e.g., requiring at least 1 element) causes many generated values to be discarded, leading to "gave up" errors.

**Solution**: Use `FlatMap` to first generate the count, then generate exactly that many elements:

```go
// Instead of:
gen.SliceOf(genStep()).SuchThat(func(s []Step) bool { return len(s) >= 1 })

// Use:
gen.IntRange(1, 5).FlatMap(func(count interface{}) gopter.Gen {
    n := count.(int)
    stepGens := make([]gopter.Gen, n)
    for i := 0; i < n; i++ {
        stepGens[i] = genStep()
    }
    return gopter.CombineGens(stepGens...)
}, reflect.TypeOf([]Step{}))
```

**Lesson**: Design generators to produce valid data directly rather than filtering invalid data.

### Challenge: Non-empty string generation

**Context**: Required fields like `Goal`, `Description`, and `Action` must be non-empty strings.

**Problem**: `gen.AlphaString()` can generate empty strings, and using `SuchThat` to filter them increases discard rate.

**Solution**: Map empty strings to a default value:

```go
func genNonEmptyAlphaString() gopter.Gen {
    return gen.AlphaString().Map(func(s string) string {
        if len(s) == 0 {
            return "a" // Ensure non-empty
        }
        return s
    })
}
```

---

## Categories

- **Parsing**: JSON serialization, nil vs empty handling
- **Testing**: Property-based testing with gopter, generator design
