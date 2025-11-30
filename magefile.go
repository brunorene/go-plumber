//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
var Default = Build

// Build builds the binary
func Build() error {
	fmt.Println("Building go-plumber...")
	return sh.Run("go", "build", "-o", "bin/go-plumber", "./cmd/go-plumber")
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	return sh.Rm("bin")
}

// Lint runs golangci-lint on the project with all available linters
func Lint() error {
	fmt.Println("Running golangci-lint with comprehensive linter set...")
	return sh.Run("golangci-lint", "run", "./...")
}

// Format formats the code using gofumpt and gci
func Format() error {
	fmt.Println("Formatting code with gofumpt...")
	if err := sh.Run("gofumpt", "-l", "-w", "."); err != nil {
		return err
	}

	fmt.Println("Organizing imports with gci...")
	return sh.Run("gci", "write", "--skip-generated", "-s", "standard", "-s", "default", "-s", "prefix(github.com/brunorene/go-plumber)", ".")
}

// FormatCheck checks if the code is properly formatted
func FormatCheck() error {
	fmt.Println("Checking code formatting with gofumpt...")
	if err := sh.Run("gofumpt", "-l", "-d", "."); err != nil {
		return err
	}

	fmt.Println("Checking import organization with gci...")
	return sh.Run("gci", "diff", "--skip-generated", "-s", "standard", "-s", "default", "-s", "prefix(github.com/brunorene/go-plumber)", ".")
}

// Test runs all tests
func Test() error {
	fmt.Println("Running tests...")
	return sh.Run("go", "test", "-v", "./...")
}

// Check runs formatting checks, linting, and tests
func Check() error {
	mg.Deps(FormatCheck, Lint, Test)
	return nil
}

// CheckAndFix runs formatting, linting, and tests (with auto-fix)
func CheckAndFix() error {
	mg.Deps(Format)
	return Check()
}

// Install installs the binary to $GOPATH/bin
func Install() error {
	fmt.Println("Installing go-plumber...")
	return sh.Run("go", "install", ".")
}

// Run builds and runs the application
func Run() error {
	mg.Deps(Build)
	fmt.Println("Running go-plumber...")
	return sh.Run("./bin/go-plumber")
}

// Dev runs the application without building a binary
func Dev() error {
	fmt.Println("Running in development mode...")
	return sh.Run("go", "run", ".")
}

// Setup creates necessary directories
func Setup() error {
	fmt.Println("Setting up project directories...")
	return os.MkdirAll("bin", 0o755)
}

// PreCommit runs all checks and formatting before committing
func PreCommit() error {
	fmt.Println("Running pre-commit checks...")
	mg.Deps(Format, Lint, Test)
	return nil
}
