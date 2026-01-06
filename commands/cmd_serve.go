package commands

import (
	"github.com/hitzhangjie/gitbook/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewServeCommand creates the serve command
func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve [book]",
		Short: "Serve the book on a local server",
		Long:  "Start a local server to preview your book",
		Run: func(cmd *cobra.Command, args []string) {
			runCommand("serve", cmd.Flags(), args)
		},
	}
}

func handleServe(bookRoot string, fset *pflag.FlagSet, args []string) error {
	srv, err := server.NewServer(bookRoot)
	if err != nil {
		return err
	}

	return srv.Start()
}
