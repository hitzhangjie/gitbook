package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hitzhangjie/gitbook/builder"
	"github.com/hitzhangjie/gitbook/ebook"
	"github.com/hitzhangjie/gitbook/initcmd"
	"github.com/hitzhangjie/gitbook/plugin"
	"github.com/hitzhangjie/gitbook/server"
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

// ExecGitbookCommand executes a gitbook command using pure Go implementation
func ExecGitbookCommand(gitbookPath, commandName, bookRoot string, args []string, kwargs map[string]interface{}) error {
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
		return handleInit(absBookRoot, args, kwargs)
	case "build":
		return handleBuild(absBookRoot, args, kwargs)
	case "serve":
		return handleServe(absBookRoot, args, kwargs)
	case "install":
		return handleInstall(absBookRoot, args, kwargs)
	case "pdf":
		return handlePDF(absBookRoot, args, kwargs)
	case "epub":
		return handleEPUB(absBookRoot, args, kwargs)
	case "mobi":
		return handleMOBI(absBookRoot, args, kwargs)
	case "help":
		return handleHelp()
	default:
		return fmt.Errorf("unknown command: %s", commandName)
	}
}

func handleInit(bookRoot string, args []string, kwargs map[string]interface{}) error {
	initDir := bookRoot
	if len(args) > 0 {
		initDir = args[0]
	}
	return initcmd.Init(initDir)
}

func handleBuild(bookRoot string, args []string, kwargs map[string]interface{}) error {
	outputDir := ""
	if len(args) > 0 {
		outputDir = args[0]
	}
	if outputDir == "" {
		if output, ok := kwargs["output"].(string); ok {
			outputDir = output
		}
	}

	builder, err := builder.NewBuilder(bookRoot, outputDir)
	if err != nil {
		return err
	}

	return builder.Build()
}

func handleServe(bookRoot string, args []string, kwargs map[string]interface{}) error {
	port := 4000
	host := "localhost"

	if p, ok := kwargs["port"].(string); ok {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}
	if h, ok := kwargs["host"].(string); ok {
		host = h
	}

	srv, err := server.NewServer(bookRoot, port, host)
	if err != nil {
		return err
	}

	return srv.Start()
}

func handleInstall(bookRoot string, args []string, kwargs map[string]interface{}) error {
	return plugin.Install(bookRoot)
}

func handlePDF(bookRoot string, args []string, kwargs map[string]interface{}) error {
	outputPath := "book.pdf"
	if len(args) > 0 {
		outputPath = args[0]
	}

	outputDir := filepath.Join(bookRoot, "_book")
	gen, err := ebook.NewGenerator(bookRoot, outputDir, "pdf")
	if err != nil {
		return err
	}

	return gen.Generate(outputPath)
}

func handleEPUB(bookRoot string, args []string, kwargs map[string]interface{}) error {
	outputPath := "book.epub"
	if len(args) > 0 {
		outputPath = args[0]
	}

	outputDir := filepath.Join(bookRoot, "_book")
	gen, err := ebook.NewGenerator(bookRoot, outputDir, "epub")
	if err != nil {
		return err
	}

	return gen.Generate(outputPath)
}

func handleMOBI(bookRoot string, args []string, kwargs map[string]interface{}) error {
	outputPath := "book.mobi"
	if len(args) > 0 {
		outputPath = args[0]
	}

	outputDir := filepath.Join(bookRoot, "_book")
	gen, err := ebook.NewGenerator(bookRoot, outputDir, "mobi")
	if err != nil {
		return err
	}

	return gen.Generate(outputPath)
}

func handleHelp() error {
	fmt.Println("GitBook Commands:")
	fmt.Println()
	fmt.Println("  init          Initialize a GitBook project")
	fmt.Println("  build         Build a static website")
	fmt.Println("  serve         Start a local server to preview your book")
	fmt.Println("  install       Install plugins for a book")
	fmt.Println("  pdf           Generate a PDF file from your book")
	fmt.Println("  epub          Generate an EPUB file from your book")
	fmt.Println("  mobi          Generate a MOBI file from your book")
	fmt.Println()
	return nil
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
