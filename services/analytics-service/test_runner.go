package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	fmt.Println("🧪 Analytics Service Test Runner")
	fmt.Println("================================")

	// Get the current directory
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Define test configurations
	testConfigs := []TestConfig{
		{
			Name:        "Unit Tests",
			Pattern:     "./...",
			Flags:       []string{"-v", "-race", "-cover"},
			Timeout:     "5m",
			Description: "Run all unit tests with race detection and coverage",
		},
		{
			Name:        "Integration Tests",
			Pattern:     "./...",
			Flags:       []string{"-v", "-tags=integration"},
			Timeout:     "10m",
			Description: "Run integration tests",
		},
		{
			Name:        "Benchmark Tests",
			Pattern:     "./...",
			Flags:       []string{"-bench=.", "-benchmem"},
			Timeout:     "5m",
			Description: "Run benchmark tests",
		},
		{
			Name:        "Coverage Report",
			Pattern:     "./...",
			Flags:       []string{"-coverprofile=coverage.out", "-covermode=atomic"},
			Timeout:     "5m",
			Description: "Generate coverage report",
		},
	}

	// Check if we're in the right directory
	if !strings.HasSuffix(pwd, "analytics-service") {
		fmt.Println("❌ Please run this from the analytics-service directory")
		os.Exit(1)
	}

	// Run tests
	totalTests := len(testConfigs)
	passed := 0
	failed := 0

	for i, config := range testConfigs {
		fmt.Printf("\n📋 Test Suite %d/%d: %s\n", i+1, totalTests, config.Name)
		fmt.Printf("📝 %s\n", config.Description)
		fmt.Println(strings.Repeat("-", 60))

		if runTest(config) {
			fmt.Printf("✅ %s - PASSED\n", config.Name)
			passed++
		} else {
			fmt.Printf("❌ %s - FAILED\n", config.Name)
			failed++
		}
	}

	// Generate HTML coverage report if coverage.out exists
	if _, err := os.Stat("coverage.out"); err == nil {
		fmt.Println("\n📊 Generating HTML coverage report...")
		cmd := exec.Command("go", "tool", "cover", "-html=coverage.out", "-o=coverage.html")
		if err := cmd.Run(); err == nil {
			fmt.Println("✅ Coverage report generated: coverage.html")
		} else {
			fmt.Printf("⚠️  Failed to generate HTML coverage report: %v\n", err)
		}
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 Test Summary")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Test Suites: %d\n", totalTests)
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)

	if failed > 0 {
		fmt.Printf("\n❌ %d test suite(s) failed\n", failed)
		os.Exit(1)
	} else {
		fmt.Printf("\n🎉 All test suites passed!\n")
	}

	// Additional checks
	fmt.Println("\n🔍 Additional Checks")
	fmt.Println(strings.Repeat("-", 60))

	// Check for go.mod and go.sum
	checkGoMod()

	// Check for proper package structure
	checkPackageStructure()

	// Check for test coverage
	checkTestCoverage()

	fmt.Println("\n✨ Test run completed successfully!")
}

type TestConfig struct {
	Name        string
	Pattern     string
	Flags       []string
	Timeout     string
	Description string
}

func runTest(config TestConfig) bool {
	args := []string{"test"}
	args = append(args, config.Flags...)

	if config.Timeout != "" {
		args = append(args, "-timeout", config.Timeout)
	}

	args = append(args, config.Pattern)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("⏱️  Duration: %v\n", duration)
		return false
	}

	fmt.Printf("⏱️  Duration: %v\n", duration)
	return true
}

func checkGoMod() {
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("⚠️  go.mod not found")
	} else {
		fmt.Println("✅ go.mod exists")
	}

	if _, err := os.Stat("go.sum"); os.IsNotExist(err) {
		fmt.Println("⚠️  go.sum not found - run 'go mod tidy'")
	} else {
		fmt.Println("✅ go.sum exists")
	}
}

func checkPackageStructure() {
	requiredDirs := []string{
		"cmd",
		"internal/handlers",
		"internal/service",
		"internal/repository",
		"proto",
	}

	fmt.Println("\n📁 Package Structure:")
	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("⚠️  Missing directory: %s\n", dir)
		} else {
			fmt.Printf("✅ %s/\n", dir)
		}
	}

	// Check for test files
	testFiles := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  Error scanning for test files: %v\n", err)
	} else {
		fmt.Printf("\n🧪 Test Files Found: %d\n", len(testFiles))
		for _, file := range testFiles {
			fmt.Printf("   📄 %s\n", file)
		}
	}
}

func checkTestCoverage() {
	if _, err := os.Stat("coverage.out"); os.IsNotExist(err) {
		fmt.Println("\n📊 No coverage file found")
		return
	}

	// Parse coverage data
	cmd := exec.Command("go", "tool", "cover", "-func=coverage.out")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Failed to parse coverage: %v\n", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		// Last line usually contains total coverage
		lastLine := lines[len(lines)-2] // -2 because last line is empty
		if strings.Contains(lastLine, "total:") {
			fmt.Printf("\n📊 %s\n", lastLine)
		}
	}
}