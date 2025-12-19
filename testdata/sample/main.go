// Package sample is a test package for the code review assistant
package sample

import (
	"fmt"
	"os"
)

// main is the entry point
func main() {
	fmt.Println("Hello, World!")
	result := Add(1, 2)
	fmt.Printf("1 + 2 = %d\n", result)

	// Call a function with multiple returns
	sum, product := Calculate(5, 3)
	fmt.Printf("Sum: %d, Product: %d\n", sum, product)

	// Check environment
	if err := checkEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Add adds two integers
// This is a simple function with 2 parameters and 1 return value
func Add(a, b int) int {
	return a + b
}

// Calculate performs both addition and multiplication
// Returns multiple values
func Calculate(x, y int) (int, int) {
	sum := x + y
	product := x * y
	return sum, product
}

// checkEnvironment verifies the environment is set up correctly
func checkEnvironment() error {
	path := os.Getenv("PATH")
	if path == "" {
		return fmt.Errorf("PATH environment variable not set")
	}

	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME environment variable not set")
	}

	return nil
}
