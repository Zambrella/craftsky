// Command cli is the Craftsky App View's companion CLI: ops tasks,
// smoke tests, and stubs for not-yet-implemented subsystems.
//
// Usage:
//
//	cli [subcommand] --env <dev|prod> [flags]
//
// --env is a persistent flag on the root command; every subcommand
// inherits it. Default is "dev" so local iteration just works.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

// envFlag is the value of --env for the current invocation, populated
// by cobra before any subcommand's RunE runs.
var envFlag string

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "Craftsky App View ops and smoke-test CLI",
	Long: `cli is a companion tool to the appview server. It provides:
  * migrate — apply or inspect database migrations
  * ping    — check DB connectivity
  * request — hit the running server as the dev DID
  * did-resolve — stub pending real impl`,
}

func main() {
	rootCmd.PersistentFlags().StringVar(&envFlag, "env", "dev", `environment: "dev" or "prod"`)
	rootCmd.AddCommand(tapCmd)
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints "Error: ..." itself; we just ensure non-zero exit.
		os.Exit(1)
	}
}
