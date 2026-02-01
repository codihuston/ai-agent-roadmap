// Package main provides the entry point for the agentic system CLI.
package main

import (
	"flag"
	"fmt"
	"os"

	"agentic-poc/internal/cli"
	"agentic-poc/internal/provider"
)

func main() {
	// Define command-line flags
	mode := flag.String("mode", "single", "Mode to run: 'single' for single-agent mode, 'multi' for multi-agent mode")
	basePath := flag.String("path", ".", "Base path for file operations")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	// Validate mode
	if *mode != "single" && *mode != "multi" {
		fmt.Fprintf(os.Stderr, "Error: invalid mode '%s'. Use 'single' or 'multi'.\n", *mode)
		printUsage()
		os.Exit(1)
	}

	// Create the LLM provider
	llmProvider, err := provider.NewClaudeProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating LLM provider: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure ANTHROPIC_API_KEY environment variable is set.")
		os.Exit(1)
	}

	// Create the CLI
	cliInstance := cli.NewCLI(llmProvider)
	cliInstance.SetBasePath(*basePath)

	// Run the appropriate mode
	var runErr error
	switch *mode {
	case "single":
		runErr = cliInstance.RunSingleAgentMode()
	case "multi":
		runErr = cliInstance.RunMultiAgentMode()
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", runErr)
		os.Exit(1)
	}
}

// printUsage prints the usage information.
func printUsage() {
	fmt.Println("Agentic System POC")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agent [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -mode string")
	fmt.Println("        Mode to run: 'single' for single-agent mode, 'multi' for multi-agent mode (default \"single\")")
	fmt.Println("  -path string")
	fmt.Println("        Base path for file operations (default \".\")")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  ANTHROPIC_API_KEY    API key for Claude (required)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Run in single-agent mode (default)")
	fmt.Println("  agent")
	fmt.Println()
	fmt.Println("  # Run in multi-agent mode")
	fmt.Println("  agent -mode multi")
	fmt.Println()
	fmt.Println("  # Run with a specific base path for file operations")
	fmt.Println("  agent -mode single -path /tmp/workspace")
}
