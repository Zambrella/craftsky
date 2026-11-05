package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/ipfs/go-cid"
	"github.com/spf13/cobra"
	"social.craftsky/appview/internal/app"
	"time"
)

// Development tools are deliberately unavailable against production.
func init() {
	parent := &cobra.Command{Use: "video-dev", Short: "Bunny development media tools"}
	for _, name := range []string{"import", "status", "cleanup"} {
		command := &cobra.Command{Use: name, Args: cobra.NoArgs}
		command.Flags().String("did", "", "Owner DID")
		command.Flags().String("cid", "", "Existing published video blob CID")
		command.Flags().String("job", "", "Imported video job ID")
		command.Flags().Bool("apply", false, "Queue unreferenced development copies older than seven days for deletion and CDN purge; otherwise preview")
		command.RunE = func(cmd *cobra.Command, _ []string) error {
			if envFlag != "dev" {
				return errors.New("video-dev is development-only")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Minute)
			defer cancel()
			deps, cleanup, err := loadDeps(ctx)
			if err != nil {
				return err
			}
			defer cleanup()
			if deps.Config.Env != app.EnvDev || deps.BunnyVideo == nil {
				return errors.New("Bunny development configuration is required")
			}
			var result any
			switch cmd.Name() {
			case "cleanup":
				apply, _ := cmd.Flags().GetBool("apply")
				count, err := deps.BunnyVideo.Cleanup(ctx, apply)
				if err != nil {
					return err
				}
				result = map[string]any{"apply": apply, "count": count}
			default:
				raw, _ := cmd.Flags().GetString("did")
				owner, err := syntax.ParseDID(raw)
				if err != nil {
					return err
				}
				if cmd.Name() == "import" {
					raw, _ := cmd.Flags().GetString("cid")
					parsed, err := cid.Parse(raw)
					if err != nil || parsed.String() != raw {
						return errors.New("invalid canonical video CID")
					}
					id, err := deps.ImportBunnyVideo(ctx, owner, syntax.CID(raw))
					if err != nil {
						return err
					}
					result = map[string]string{"jobId": id}
				} else {
					id, _ := cmd.Flags().GetString("job")
					status, err := deps.BunnyVideo.ImportedStatus(ctx, owner, id)
					if err != nil {
						return err
					}
					result = status
				}
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		parent.AddCommand(command)
	}
	rootCmd.AddCommand(parent)
}
