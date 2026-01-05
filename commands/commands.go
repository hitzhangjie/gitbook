package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Command struct {
	Name        string
	Description string
	Options     []Option
	Exec        func(args []string, kwargs map[string]interface{}) error
}

type Option struct {
	Name        string
	Description string
	Defaults    interface{}
	Values      []string
}

// Help prints help for a list of commands
func Help(commands []Command) {
	for _, cmd := range commands {
		indentOutput(1, cmd.Name, cmd.Description)
		for _, option := range cmd.Options {
			after := []string{}

			if option.Defaults != nil {
				after = append(after, fmt.Sprintf("Default is %v", option.Defaults))
			}
			if len(option.Values) > 0 {
				after = append(after, fmt.Sprintf("Values are %s", strings.Join(option.Values, ", ")))
			}

			afterStr := ""
			if len(after) > 0 {
				afterStr = "(" + strings.Join(after, "; ") + ")"
			}

			optname := "--"
			if _, ok := option.Defaults.(bool); ok {
				optname += "[no-]"
			}
			optname += option.Name
			indentOutput(2, optname, option.Description+" "+afterStr)
		}
		fmt.Println()
	}
}

// Exec executes a command from a list with a specific set of args/kwargs
func Exec(commands []Command, commandName string, args []string, kwargs map[string]interface{}) error {
	var cmd *Command
	for i := range commands {
		cmdParts := strings.Fields(commands[i].Name)
		if len(cmdParts) > 0 && cmdParts[0] == commandName {
			cmd = &commands[i]
			break
		}
	}

	if cmd == nil {
		return fmt.Errorf("command %s doesn't exist, run \"gitbook help\" to list commands", commandName)
	}

	// Apply defaults
	for _, option := range cmd.Options {
		if _, exists := kwargs[option.Name]; !exists {
			kwargs[option.Name] = option.Defaults
		}

		if len(option.Values) > 0 {
			value, ok := kwargs[option.Name].(string)
			if ok {
				found := false
				for _, v := range option.Values {
					if v == value {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("invalid value for option \"%s\"", option.Name)
				}
			}
		}
	}

	return cmd.Exec(args, kwargs)
}

// ExecGitbookCommand executes a gitbook command by loading the gitbook module and calling it
func ExecGitbookCommand(gitbookPath, commandName, bookRoot string, args []string, kwargs map[string]interface{}) error {
	gitbookDir := gitbookPath
	if !filepath.IsAbs(gitbookDir) {
		var err error
		gitbookDir, err = filepath.Abs(gitbookDir)
		if err != nil {
			return err
		}
	}

	// Try to find gitbook CLI entry point
	// GitBook typically has bin/gitbook.js or can be executed via npx
	var gitbookBin string
	var useNpx bool

	// Check for bin/gitbook.js
	gitbookBin = filepath.Join(gitbookDir, "bin", "gitbook.js")
	if _, err := os.Stat(gitbookBin); err != nil {
		// Try cli.js
		gitbookBin = filepath.Join(gitbookDir, "cli.js")
		if _, err := os.Stat(gitbookBin); err != nil {
			// Try using npx with the gitbook package in the directory
			useNpx = true
			gitbookBin = "gitbook"
		}
	}

	// Build command arguments
	// The gitbook CLI expects: <command> [bookRoot] [args...] [flags...]
	cmdArgs := []string{}
	if !useNpx {
		cmdArgs = append(cmdArgs, gitbookBin)
	}
	cmdArgs = append(cmdArgs, commandName)

	// Add book root if provided and not already in args
	hasBookRoot := false
	for _, arg := range args {
		// Check if any arg matches the book root (as absolute or relative path)
		argAbs, _ := filepath.Abs(arg)
		bookRootAbs, _ := filepath.Abs(bookRoot)
		if arg == bookRoot || argAbs == bookRootAbs {
			hasBookRoot = true
			break
		}
	}
	if !hasBookRoot && bookRoot != "" && bookRoot != "." {
		absBookRoot, err := filepath.Abs(bookRoot)
		if err == nil {
			cmdArgs = append(cmdArgs, absBookRoot)
		} else {
			cmdArgs = append(cmdArgs, bookRoot)
		}
	}

	// Add positional arguments
	cmdArgs = append(cmdArgs, args...)

	// Add kwargs as flags
	for k, v := range kwargs {
		if v == nil {
			continue
		}
		// Skip gitbook version flag as it's handled by the CLI
		if k == "gitbook" || k == "v" {
			continue
		}
		switch val := v.(type) {
		case bool:
			if val {
				cmdArgs = append(cmdArgs, "--"+k)
			} else {
				cmdArgs = append(cmdArgs, "--no-"+k)
			}
		case string:
			if val != "" {
				cmdArgs = append(cmdArgs, "--"+k+"="+val)
			}
		default:
			cmdArgs = append(cmdArgs, "--"+k+"="+fmt.Sprintf("%v", val))
		}
	}

	// Execute gitbook command
	// Change to book root directory for execution
	originalDir, err := os.Getwd()
	if err != nil {
		return err
	}

	absBookRoot, err := filepath.Abs(bookRoot)
	if err != nil {
		absBookRoot = bookRoot
	}

	// Ensure book root exists
	if _, err := os.Stat(absBookRoot); err != nil {
		// If book root doesn't exist, use current directory
		absBookRoot = originalDir
	}

	if err := os.Chdir(absBookRoot); err != nil {
		// If can't change to book root, continue with current directory
		absBookRoot = originalDir
	}
	defer os.Chdir(originalDir)

	// Set NODE_PATH to include gitbook directory for module resolution
	env := os.Environ()
	nodePath := os.Getenv("NODE_PATH")
	if nodePath != "" {
		nodePath = gitbookDir + string(filepath.ListSeparator) + nodePath
	} else {
		nodePath = gitbookDir
	}
	env = append(env, "NODE_PATH="+nodePath)

	var cmd *exec.Cmd
	if useNpx {
		// Use npx to run gitbook from the directory
		cmd = exec.Command("npx", append([]string{"--prefix", gitbookDir, "gitbook"}, cmdArgs...)...)
	} else {
		cmd = exec.Command("node", cmdArgs...)
	}

	cmd.Dir = absBookRoot
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func indentOutput(n int, name, description string) {
	if n == 0 {
		n = 0
	}

	spaces := strings.Repeat("    ", n)
	padding := 32 - n*4 - len(name)
	if padding < 1 {
		padding = 1
	}

	fmt.Printf("%s%s%s%s\n", spaces, name, strings.Repeat(" ", padding), description)
}
