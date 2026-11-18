package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "config-check",
		Short: "Validate configuration without contacting dependencies (values withheld)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := parseEnvFlag()
			if err != nil {
				return fmt.Errorf("environment must be dev or prod")
			}
			if _, err := loadCfgLight(env); err != nil {
				return fmt.Errorf("configuration invalid: check required variables and formats; values withheld")
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Configuration valid (no services contacted)")
			return err
		},
	})
}
