package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/hitzhangjie/gitbook/commands"
	"github.com/hitzhangjie/gitbook/config"
	"github.com/hitzhangjie/gitbook/local"
	"github.com/hitzhangjie/gitbook/manager"
	"github.com/hitzhangjie/gitbook/registry"
)

var (
	gitbookVersion string
	debug          bool
	bookRoot       string
	cliVersion     = "2.3.2" // Match original gitbook-cli version
)

var rootCmd = &cobra.Command{
	Use:   "gitbook",
	Short: "CLI to generate books and documentation using gitbook",
	Long:  "The GitBook command line interface.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize gitbook-cli
		if err := config.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing gitbook-cli: %v\n", err)
			os.Exit(1)
		}

		// Determine book root
		// Commands that don't take book root as first argument
		noBookRootCommands := []string{
			"ls", "ls-remote", "list", "list-remote",
			"fetch", "update", "uninstall",
			"alias", "current", "help", "version",
		}
		isNoBookRootCmd := false
		for _, c := range noBookRootCommands {
			if cmd.Name() == c {
				isNoBookRootCmd = true
				break
			}
		}
		if len(args) > 0 && !isNoBookRootCmd {
			bookRoot = args[0]
		} else {
			var err error
			bookRoot, err = os.Getwd()
			if err != nil {
				bookRoot = "."
			}
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&gitbookVersion, "gitbook", "v", "", "specify GitBook version to use")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable verbose error")
	rootCmd.PersistentFlags().StringVarP(&bookRoot, "book", "b", "", "book root directory")

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Display running versions of gitbook and gitbook-cli",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("CLI version:", cliVersion)
			v, err := manager.EnsureVersion(bookRoot, gitbookVersion, true)
			if err != nil {
				printError(err)
				os.Exit(1)
			}
			fmt.Println("GitBook version:", printGitbookVersion(v))
		},
	})

	// ls command
	rootCmd.AddCommand(&cobra.Command{
		Use:     "ls",
		Short:   "List versions installed locally",
		Aliases: []string{"list"},
		Run: func(cmd *cobra.Command, args []string) {
			versions, err := local.Versions()
			if err != nil {
				printError(err)
				os.Exit(1)
			}

			if len(versions) > 0 {
				fmt.Println("GitBook Versions Installed:")
				fmt.Println()
				for i, v := range versions {
					text := v.Name
					if v.Name != v.Version {
						text += " [" + v.Version + "]"
					}
					if v.Link != "" {
						text += " (alias of " + v.Link + ")"
					}

					marker := " "
					if i == 0 {
						marker = "*"
					}
					fmt.Printf("   %s %s\n", marker, text)
				}
				fmt.Println()
				fmt.Println("Run \"gitbook update\" to update to the latest version.")
			} else {
				fmt.Println("There is no versions installed")
				fmt.Println("You can install the latest version using: \"gitbook fetch\"")
			}
		},
	})

	// current command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "current",
		Short: "Display currently activated version",
		Run: func(cmd *cobra.Command, args []string) {
			v, err := manager.EnsureVersion(bookRoot, gitbookVersion, true)
			if err != nil {
				printError(err)
				os.Exit(1)
			}
			fmt.Println("GitBook version is", printGitbookVersion(v))
		},
	})

	// ls-remote command
	rootCmd.AddCommand(&cobra.Command{
		Use:     "ls-remote",
		Short:   "List remote versions available for install",
		Aliases: []string{"list-remote"},
		Run: func(cmd *cobra.Command, args []string) {
			available, err := registry.Versions()
			if err != nil {
				printError(err)
				os.Exit(1)
			}

			fmt.Println("Available GitBook Versions:")
			fmt.Println()
			fmt.Println("    ", strings.Join(available.Versions, ", "))
			fmt.Println()
			fmt.Println("Tags:")
			fmt.Println()
			for tagName, version := range available.Tags {
				fmt.Printf("    %s: %s\n", tagName, version)
			}
			fmt.Println()
		},
	})

	// fetch command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "fetch [version]",
		Short: "Download and install a <version>",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			version := "*"
			if len(args) > 0 {
				version = args[0]
			}

			installedVersion, err := registry.Install(version, false)
			if err != nil {
				printError(err)
				os.Exit(1)
			}

			fmt.Println()
			color.Green("GitBook %s has been installed", installedVersion)
		},
	})

	// alias command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "alias [folder] [version]",
		Short: "Set an alias named <version> pointing to <folder>",
		Args:  cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			var folder, version string
			if len(args) > 0 {
				folder = args[0]
			} else {
				var err error
				folder, err = os.Getwd()
				if err != nil {
					printError(err)
					os.Exit(1)
				}
			}

			if len(args) > 1 {
				version = args[1]
			} else {
				version = "latest"
			}

			absFolder, err := filepath.Abs(folder)
			if err != nil {
				printError(err)
				os.Exit(1)
			}

			if err := local.Link(version, absFolder); err != nil {
				printError(err)
				os.Exit(1)
			}

			color.Green("GitBook %s point to %s", version, absFolder)
		},
	})

	// uninstall command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "uninstall [version]",
		Short: "Uninstall a version",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				printError(fmt.Errorf("no version specified"))
				os.Exit(1)
			}

			if err := local.Remove(args[0]); err != nil {
				printError(err)
				os.Exit(1)
			}

			color.Green("GitBook %s has been uninstalled.", args[0])
		},
	})

	// update command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "update [tag]",
		Short: "Update to the latest version of GitBook",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			tag := "latest"
			if len(args) > 0 {
				tag = args[0]
			}

			version, err := manager.UpdateVersion(tag)
			if err != nil {
				printError(err)
				os.Exit(1)
			}

			if version == "" {
				fmt.Println("No update found!")
			} else {
				fmt.Println()
				color.Green("GitBook has been updated to %s", version)
			}
		},
	})

	// help command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "help",
		Short: "List commands for GitBook",
		Run: func(cmd *cobra.Command, args []string) {
			gitbookPath, err := manager.EnsureAndLoad(bookRoot, gitbookVersion)
			if err != nil {
				printError(err)
				os.Exit(1)
			}

			// Load gitbook commands - this is a simplified version
			// In the original, it loads commands from the gitbook module
			// For now, we'll just show a message that they should use gitbook help from the installed version
			fmt.Println("GitBook commands are loaded from the installed GitBook version.")
			fmt.Println("To see available commands, GitBook will be executed with 'help'.")
			fmt.Println()

			// Try to execute gitbook help
			kwargs := make(map[string]interface{})
			if err := commands.ExecGitbookCommand(gitbookPath, "help", bookRoot, []string{}, kwargs); err != nil {
				fmt.Printf("Note: Could not load GitBook commands: %v\n", err)
				fmt.Println("Make sure GitBook is properly installed using 'gitbook fetch'")
			}
		},
	})

	// GitBook commands (serve, build, pdf, epub, etc.)
	// serve command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "serve [book]",
		Short: "Serve the book on a local server",
		Long:  "Start a local server to preview your book",
		Run: func(cmd *cobra.Command, args []string) {
			handleGitbookCommandFromCobra("serve", args, cmd)
		},
	})

	// build command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "build [book] [output]",
		Short: "Build a gitbook from a directory",
		Long:  "Build a static website using gitbook",
		Run: func(cmd *cobra.Command, args []string) {
			handleGitbookCommandFromCobra("build", args, cmd)
		},
	})

	// pdf command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "pdf [book] [output]",
		Short: "Build a pdf from a book",
		Long:  "Generate a PDF file from your book",
		Run: func(cmd *cobra.Command, args []string) {
			handleGitbookCommandFromCobra("pdf", args, cmd)
		},
	})

	// epub command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "epub [book] [output]",
		Short: "Build an epub from a book",
		Long:  "Generate an EPUB file from your book",
		Run: func(cmd *cobra.Command, args []string) {
			handleGitbookCommandFromCobra("epub", args, cmd)
		},
	})

	// mobi command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "mobi [book] [output]",
		Short: "Build a mobi from a book",
		Long:  "Generate a MOBI file from your book",
		Run: func(cmd *cobra.Command, args []string) {
			handleGitbookCommandFromCobra("mobi", args, cmd)
		},
	})

	// init command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "init [book]",
		Short: "Setup and initialize a book",
		Long:  "Initialize a book structure in the current directory or specified directory",
		Run: func(cmd *cobra.Command, args []string) {
			handleGitbookCommandFromCobra("init", args, cmd)
		},
	})

	// install command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "install [book]",
		Short: "Install plugins for a book",
		Long:  "Install all plugins dependencies for a book",
		Run: func(cmd *cobra.Command, args []string) {
			handleGitbookCommandFromCobra("install", args, cmd)
		},
	})
}

func main() {
	// Check for version flag first
	for i, arg := range os.Args {
		if arg == "-V" || arg == "--version" {
			// Initialize
			if err := config.Init(); err != nil {
				printError(err)
				os.Exit(1)
			}

			// Determine book root
			bookRoot := "."
			if len(os.Args) > i+1 && !strings.HasPrefix(os.Args[i+1], "-") {
				bookRoot = os.Args[i+1]
			} else {
				var err error
				bookRoot, err = os.Getwd()
				if err != nil {
					bookRoot = "."
				}
			}

			// Extract gitbook version from flags
			gitbookVer := ""
			for j, a := range os.Args {
				if (a == "-v" || a == "--gitbook") && j+1 < len(os.Args) {
					gitbookVer = os.Args[j+1]
					break
				}
			}

			fmt.Println("CLI version:", cliVersion)
			v, err := manager.EnsureVersion(bookRoot, gitbookVer, true)
			if err != nil {
				printError(err)
				os.Exit(1)
			}
			fmt.Println("GitBook version:", printGitbookVersion(v))
			os.Exit(0)
		}
	}

	// Handle catch-all command for gitbook commands (build, serve, etc.)
	rootCmd.SetArgs(os.Args[1:])

	// Check if this is a gitbook command (not a subcommand)
	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		// Check if it's not a known subcommand (CLI commands or registered GitBook commands)
		knownCommands := []string{
			"ls", "ls-remote", "list", "list-remote",
			"fetch", "update", "uninstall",
			"alias", "current", "help", "version",
			"serve", "build", "pdf", "epub", "mobi", "init", "install",
		}
		isKnownCommand := false
		for _, cmd := range knownCommands {
			if firstArg == cmd {
				isKnownCommand = true
				break
			}
		}
		if !isKnownCommand && !strings.HasPrefix(firstArg, "-") {
			// This is likely an unregistered gitbook command, use catch-all handler
			handleGitbookCommand(firstArg, os.Args[2:])
			return
		}
	}

	if err := rootCmd.Execute(); err != nil {
		printError(err)
		os.Exit(1)
	}
}

// handleGitbookCommandFromCobra handles GitBook commands registered as cobra commands
func handleGitbookCommandFromCobra(commandName string, args []string, cmd *cobra.Command) {
	// Initialize
	if err := config.Init(); err != nil {
		printError(err)
		os.Exit(1)
	}

	// Determine book root
	bookRoot := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		bookRoot = args[0]
		args = args[1:]
	} else {
		var err error
		bookRoot, err = os.Getwd()
		if err != nil {
			bookRoot = "."
		}
	}

	// Parse kwargs from args (flags like --port, --host, etc.)
	kwargs := make(map[string]interface{})
	filteredArgs := []string{}
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if strings.HasPrefix(key, "no-") {
				key = strings.TrimPrefix(key, "no-")
				kwargs[key] = false
			} else if strings.Contains(key, "=") {
				parts := strings.SplitN(key, "=", 2)
				kwargs[parts[0]] = parts[1]
			} else {
				// Check if next arg is a value
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
					kwargs[key] = args[i+1]
					skipNext = true
				} else {
					kwargs[key] = true
				}
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Handle short flags like -v, -d
			flag := arg[1:]
			if len(flag) == 1 {
				// Single character flag, check if next arg is value
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					kwargs[flag] = args[i+1]
					skipNext = true
				} else {
					kwargs[flag] = true
				}
			} else {
				// Multiple flags or flag with value (e.g., -v=2.0.1)
				if strings.Contains(flag, "=") {
					parts := strings.SplitN(flag, "=", 2)
					kwargs[parts[0]] = parts[1]
				} else {
					kwargs[flag] = true
				}
			}
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// Handle --gitbook or -v flag from persistent flags
	if gitbookVersion != "" {
		kwargs["gitbook"] = gitbookVersion
	} else if v, ok := kwargs["gitbook"].(string); ok {
		gitbookVersion = v
	} else if v, ok := kwargs["v"].(string); ok {
		gitbookVersion = v
	} else if v, ok := kwargs["v"].(bool); ok && v {
		// -v without value, use empty string (will use book.json or latest)
		gitbookVersion = ""
	}

	// Ensure and load gitbook
	gitbookPath, err := manager.EnsureAndLoad(bookRoot, gitbookVersion)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Execute the command
	if err := commands.ExecGitbookCommand(gitbookPath, commandName, bookRoot, filteredArgs, kwargs); err != nil {
		printError(err)
		os.Exit(1)
	}
}

func handleGitbookCommand(commandName string, args []string) {
	// Initialize
	if err := config.Init(); err != nil {
		printError(err)
		os.Exit(1)
	}

	// Determine book root
	bookRoot := "."
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		bookRoot = args[0]
		args = args[1:]
	} else {
		var err error
		bookRoot, err = os.Getwd()
		if err != nil {
			bookRoot = "."
		}
	}

	// Parse kwargs from args
	kwargs := make(map[string]interface{})
	filteredArgs := []string{}
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if strings.HasPrefix(key, "no-") {
				key = strings.TrimPrefix(key, "no-")
				kwargs[key] = false
			} else if strings.Contains(key, "=") {
				parts := strings.SplitN(key, "=", 2)
				kwargs[parts[0]] = parts[1]
			} else {
				// Check if next arg is a value
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
					kwargs[key] = args[i+1]
					skipNext = true
				} else {
					kwargs[key] = true
				}
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Handle short flags like -v, -d
			flag := arg[1:]
			if len(flag) == 1 {
				// Single character flag, check if next arg is value
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					kwargs[flag] = args[i+1]
					skipNext = true
				} else {
					kwargs[flag] = true
				}
			} else {
				// Multiple flags or flag with value (e.g., -v=2.0.1)
				if strings.Contains(flag, "=") {
					parts := strings.SplitN(flag, "=", 2)
					kwargs[parts[0]] = parts[1]
				} else {
					kwargs[flag] = true
				}
			}
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// Handle --gitbook or -v flag
	if v, ok := kwargs["gitbook"].(string); ok {
		gitbookVersion = v
	} else if v, ok := kwargs["v"].(string); ok {
		gitbookVersion = v
	} else if v, ok := kwargs["v"].(bool); ok && v {
		// -v without value, use empty string (will use book.json or latest)
		gitbookVersion = ""
	}

	// Ensure and load gitbook
	gitbookPath, err := manager.EnsureAndLoad(bookRoot, gitbookVersion)
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	// Execute the command
	if err := commands.ExecGitbookCommand(gitbookPath, commandName, bookRoot, filteredArgs, kwargs); err != nil {
		printError(err)
		os.Exit(1)
	}
}

func printGitbookVersion(v *local.VersionInfo) string {
	actualVersion := ""
	if v.Name != v.Version {
		actualVersion = " (" + v.Version + ")"
	}
	return v.Name + actualVersion
}

func printError(err error) {
	fmt.Println()
	color.Red(err.Error())
	if debug || os.Getenv("DEBUG") != "" {
		fmt.Println(err)
	}
}
