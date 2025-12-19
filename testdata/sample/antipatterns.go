package sample

import "fmt"

// This file contains functions that demonstrate various anti-patterns
// for testing the anti-pattern detection feature

// TooManyParameters has 7 parameters (exceeds threshold of 5)
func TooManyParameters(a, b, c, d, e, f, g int) int {
	return a + b + c + d + e + f + g
}

// AcceptableParameters has exactly 5 parameters (at threshold)
func AcceptableParameters(a, b, c, d, e int) int {
	return a + b + c + d + e
}

// DeeplyNested has nesting depth of 5 (exceeds threshold of 4)
func DeeplyNested(x int) string {
	if x > 0 { // depth 1
		if x > 10 { // depth 2
			if x > 20 { // depth 3
				if x > 30 { // depth 4
					if x > 40 { // depth 5
						return "very deep"
					}
					return "deep 4"
				}
				return "deep 3"
			}
			return "deep 2"
		}
		return "deep 1"
	}
	return "not deep"
}

// ModerateNesting has nesting depth of 3 (acceptable)
func ModerateNesting(x int) string {
	if x > 0 { // depth 1
		if x > 10 { // depth 2
			if x > 20 { // depth 3
				return "moderately deep"
			}
			return "medium"
		}
		return "shallow"
	}
	return "not nested"
}

// TooManyReturns has 6 return statements (exceeds threshold of 3)
func TooManyReturns(status string, value int) string {
	if status == "ready" {
		return "ready"
	}
	if status == "pending" {
		return "pending"
	}
	if status == "error" {
		return "error"
	}
	if value > 100 {
		return "high"
	}
	if value < 0 {
		return "negative"
	}
	return "normal"
}

// AcceptableReturns has 3 return statements (at threshold)
func AcceptableReturns(x int) string {
	if x > 0 {
		return "positive"
	}
	if x < 0 {
		return "negative"
	}
	return "zero"
}

// MagicNumbersFunction contains several magic numbers
func MagicNumbersFunction(x int) int {
	result := x * 42      // magic number
	result += 100         // magic number
	result /= 7           // magic number
	threshold := 365      // magic number

	if result > threshold {
		result -= 86400   // magic number (seconds in a day)
	}

	// These should NOT be flagged (0, 1, -1 are allowed)
	if x == 0 || x == 1 || x == -1 {
		return result
	}

	return result + 3.14159  // magic number (float)
}

// NoMagicNumbers uses constants (should not be flagged)
func NoMagicNumbers(x int) int {
	const (
		multiplier = 42
		addend     = 100
	)

	result := x * multiplier
	result += addend

	return result
}

// DuplicateErrorHandling has 8 error checks (exceeds threshold of 5)
func DuplicateErrorHandling() error {
	var err error

	err = step1()
	if err != nil {
		return err
	}

	err = step2()
	if err != nil {
		return err
	}

	err = step3()
	if err != nil {
		return err
	}

	err = step4()
	if err != nil {
		return err
	}

	err = step5()
	if err != nil {
		return err
	}

	err = step6()
	if err != nil {
		return err
	}

	err = step7()
	if err != nil {
		return err
	}

	err = step8()
	if err != nil {
		return err
	}

	return nil
}

// AcceptableErrorHandling has 4 error checks (acceptable)
func AcceptableErrorHandling() error {
	var err error

	err = step1()
	if err != nil {
		return err
	}

	err = step2()
	if err != nil {
		return err
	}

	err = step3()
	if err != nil {
		return err
	}

	err = step4()
	if err != nil {
		return err
	}

	return nil
}

// AllAntiPatterns combines multiple anti-patterns in one function
func AllAntiPatterns(a, b, c, d, e, f int) error {
	// Too many parameters (6)
	var err error

	// Deep nesting
	if a > 0 {
		if b > 0 {
			if c > 0 {
				if d > 0 {
					if e > 0 {
						// Magic numbers
						result := a * 42 + b * 100

						// Duplicate error handling
						err = step1()
						if err != nil {
							return err
						}

						err = step2()
						if err != nil {
							return err
						}

						err = step3()
						if err != nil {
							return err
						}

						err = step4()
						if err != nil {
							return err
						}

						err = step5()
						if err != nil {
							return err
						}

						err = step6()
						if err != nil {
							return err
						}

						// Too many returns
						if result > 1000 {
							return fmt.Errorf("too high")
						}
						if result < -1000 {
							return fmt.Errorf("too low")
						}
						if result == 0 {
							return fmt.Errorf("zero")
						}
						if result%2 == 0 {
							return fmt.Errorf("even")
						}
					}
				}
			}
		}
	}

	return nil
}

// Helper functions for error handling examples
func step1() error { return nil }
func step2() error { return nil }
func step3() error { return nil }
func step4() error { return nil }
func step5() error { return nil }
func step6() error { return nil }
func step7() error { return nil }
func step8() error { return nil }
