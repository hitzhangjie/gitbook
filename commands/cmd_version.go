package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display running versions of gitbook and gitbook-cli",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("gitbook is an utility rewritten in Go, which is based on gitbook-cli 2.3.2")
			fmt.Println("gitbook version: v0.0.1")
		},
	}
}
