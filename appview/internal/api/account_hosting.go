package api

import (
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Provider links are based on the resolved PDS, never the mutable handle.
// /account is the reference implementation's convention; providers may differ.
func accountHostingInfo(did syntax.DID, handle syntax.Handle, endpoint string) WhoAmIResponse {
	info := WhoAmIResponse{DID: did, Handle: handle}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return info
	}
	info.PDSURL = u.Scheme + "://" + u.Host
	info.HostingProvider = u.Host
	info.AccountManagementURL = info.PDSURL + "/account"
	host := strings.ToLower(u.Hostname())
	if (u.Port() == "" || u.Port() == "443") && (host == "bsky.social" || strings.HasSuffix(host, ".host.bsky.network")) {
		info.HostingProvider = "Bluesky"
		info.AccountManagementURL = "https://bsky.social/account"
		info.PasswordChangeURL = "https://bsky.app/settings/account"
	}
	return info
}
