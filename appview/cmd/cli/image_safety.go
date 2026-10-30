package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/imagesafety"
)

func init() {
	imageSafety := &cobra.Command{Use: "image-safety", Short: "Image safety development tools"}
	imageSafety.AddCommand(&cobra.Command{
		Use: "dev-catch-up", Short: "Requeue unresolved local images for development auto-approval",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := parseEnvFlag()
			if err != nil {
				return err
			}
			// Refuse before loading production configuration or opening a connection.
			if env != app.EnvDev {
				return fmt.Errorf("image-safety dev-catch-up is dev-only")
			}
			cfg, err := loadCfgLight(env)
			if err != nil {
				return err
			}
			if cfg.ImageSafety.Mode != imagesafety.ScannerModeDevelopmentAutoApprove || !cfg.ImageSafety.Ready() {
				return fmt.Errorf("image-safety dev-catch-up requires IMAGE_SAFETY_SCANNER_MODE=dev-auto-approve")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()
			pool, err := db.Connect(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer pool.Close()
			count, err := imagesafety.CatchUpDevelopment(ctx, pool, cfg.ImageSafety)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Requeued %d image-dependent projections for development auto-approval.\n", count)
			return err
		},
	})
	rootCmd.AddCommand(imageSafety)
}
