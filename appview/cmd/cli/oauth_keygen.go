package main

import (
	"fmt"
	"io"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/spf13/cobra"
)

// oauthKeygenCmd generates a P-256 private key and prints its multibase
// encoding to stdout. Paste the output into your prod-style .env as
// OAUTH_CLIENT_SECRET_KEY. Never commit the key.
func oauthKeygenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "oauth-keygen",
		Short: "Generate a P-256 private key for OAuth confidential-client auth",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runOAuthKeygen(cmd.OutOrStdout())
		},
	}
}

func runOAuthKeygen(w io.Writer) error {
	priv, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	// Note: indigo's PrivateKeyP256.Multibase() returns string (no error).
	_, err = fmt.Fprintln(w, priv.Multibase())
	return err
}
