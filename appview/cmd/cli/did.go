package main

import (
	"errors"

	"github.com/spf13/cobra"
)

var didResolveCmd = &cobra.Command{
	Use:   "did-resolve HANDLE",
	Short: "Resolve a handle to a DID (stub until identity resolver lands)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("did-resolve: not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(didResolveCmd)
}
