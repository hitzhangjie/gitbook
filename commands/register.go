package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// RegisterCommands registers all subcommands to the root command
func RegisterCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(NewVersionCommand())
	// GitBook commands (serve, build, pdf, epub, etc.)
	rootCmd.AddCommand(NewServeCommand())
	rootCmd.AddCommand(NewBuildCommand())
	rootCmd.AddCommand(NewPDFCommand())
	rootCmd.AddCommand(NewEPUBCommand())
	rootCmd.AddCommand(NewMOBICommand())
	rootCmd.AddCommand(NewInitCommand())
}

// getBookRoot returns the book root directory
func getBookRoot(args []string) string {
	if len(args) == 0 {
		v, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		return v
	}
	return args[0]
}

// runCommand handles GitBook commands registered as cobra commands
func runCommand(commandName string, fset *pflag.FlagSet, args []string) {
	bookRoot := getBookRoot(args)
	absBookRoot, err := filepath.Abs(bookRoot)
	if err != nil {
		absBookRoot = bookRoot
	}

	// Ensure book root exists
	if _, err := os.Stat(absBookRoot); err != nil {
		absBookRoot, _ = os.Getwd()
	}

	switch commandName {
	case "init":
		err = handleInit(absBookRoot, fset, args)
	case "build":
		err = handleBuild(absBookRoot, fset, args)
	case "serve":
		err = handleServe(absBookRoot, fset, args)
	case "pdf":
		err = handlePDF(absBookRoot, fset, args)
	case "epub":
		err = handleEPUB(absBookRoot, fset, args)
	case "mobi":
		err = handleMOBI(absBookRoot, fset, args)
	default:
		err = fmt.Errorf("unknown command: %s", commandName)
	}

	if err != nil {
		PrintError(err)
		os.Exit(1)
	}
}

// PrintError prints an error message
func PrintError(err error) {
	fmt.Println()
	color.Red(err.Error())
	if os.Getenv("DEBUG") != "" {
		fmt.Println(err)
	}
}
