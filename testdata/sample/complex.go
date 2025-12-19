package sample

// This file contains functions with varying levels of cyclomatic complexity
// for testing the complexity analysis feature

// SimpleFunction has CC=1 (no branches)
func SimpleFunction() int {
	return 42
}

// LowComplexity has CC=2 (one if statement)
func LowComplexity(x int) string {
	if x > 0 {
		return "positive"
	}
	return "non-positive"
}

// ModerateComplexity has CC=5
func ModerateComplexity(x, y int) string {
	if x > 0 && y > 0 { // +1 for if, +1 for &&
		return "both positive"
	} else if x > 0 { // +1 for else-if
		return "x positive"
	} else if y > 0 { // +1 for else-if
		return "y positive"
	}
	return "both non-positive"
}

// HighComplexity has CC=12 (exceeds threshold of 10)
func HighComplexity(value int, flag bool, status string) (string, error) {
	if value < 0 { // +1
		return "", nil
	}

	if flag && value > 100 { // +1 for if, +1 for &&
		return "high value", nil
	}

	switch status { // switch itself doesn't add, cases do
	case "active": // +1
		if value > 50 { // +1
			return "active-high", nil
		}
		return "active-low", nil
	case "pending": // +1
		return "pending", nil
	case "inactive": // +1
		return "inactive", nil
	default:
		// default doesn't add complexity
	}

	for i := 0; i < value; i++ { // +1 for loop
		if i%2 == 0 { // +1
			continue
		}
	}

	if value > 10 || value < -10 { // +1 for if, +1 for ||
		return "extreme", nil
	}

	return "normal", nil
}

// WithSwitch has CC=4 (1 + 3 case clauses)
func WithSwitch(day string) string {
	switch day {
	case "Monday": // +1
		return "Start of week"
	case "Friday": // +1
		return "End of work week"
	case "Saturday", "Sunday": // +1 (counted as one case)
		return "Weekend"
	default:
		return "Midweek"
	}
}

// WithNestedLoops has CC=4
func WithNestedLoops(matrix [][]int) int {
	sum := 0
	for _, row := range matrix { // +1
		for _, val := range row { // +1
			if val > 0 { // +1
				sum += val
			}
		}
	}
	return sum
}

// WithLogicalOperators has CC=5
func WithLogicalOperators(a, b, c, d bool) bool {
	if a && b { // +1 for if, +1 for &&
		return true
	}

	if c || d { // +1 for if, +1 for ||
		return true
	}

	return false
}

// VeryHighComplexity has CC=15 (exceeds threshold significantly)
func VeryHighComplexity(n int, flags map[string]bool) int {
	result := 0

	if n < 0 { // +1
		return -1
	}

	if flags["debug"] && flags["verbose"] { // +1 for if, +1 for &&
		n *= 2
	}

	for i := 0; i < n; i++ { // +1
		if i%2 == 0 { // +1
			if i%4 == 0 { // +1
				result += 4
			} else { // counted in the if above
				result += 2
			}
		} else if i%3 == 0 { // +1
			result += 3
		}

		if flags["skip"] || flags["fast"] { // +1 for if, +1 for ||
			continue
		}
	}

	switch result {
	case 0: // +1
		return -1
	case 1: // +1
		return 1
	case 2: // +1
		return 2
	default:
		// +0
	}

	if flags["double"] { // +1
		result *= 2
	}

	if result > 100 && result < 1000 { // +1 for if, +1 for &&
		result /= 2
	}

	return result
}

// WithTypeSwitch has CC=4
func WithTypeSwitch(val interface{}) string {
	switch v := val.(type) { // type switch
	case int: // +1
		return "integer"
	case string: // +1
		return "string"
	case bool: // +1
		return "boolean"
	default:
		return "unknown"
	}
}

// WithSelect has CC=3
func WithSelect(ch1, ch2 <-chan int) int {
	select {
	case val := <-ch1: // +1
		return val
	case val := <-ch2: // +1
		return val
	default:
		return 0
	}
}
