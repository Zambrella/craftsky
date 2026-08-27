package linkpreview

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"net/url"
)

const maxRedirects = 5

const (
	userAgent   = "CraftskyLinkPreview/1.0 (+https://craftsky.social)"
	pageAccept  = "text/html, application/xhtml+xml"
	imageAccept = "image/jpeg, image/png, image/webp"
)

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type HTTPDoer interface {
	Do(request *http.Request) (*http.Response, error)
}

type Fetcher struct {
	resolver Resolver
	doer     HTTPDoer
}

func NewFetcher(resolver Resolver, doer HTTPDoer) *Fetcher {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &Fetcher{resolver: resolver, doer: doer}
}

// Fetch follows redirects manually so every next hop is validated before an
// HTTP request is issued. The caller owns the successful response body.
func (f *Fetcher) Fetch(ctx context.Context, raw string) (*http.Response, *url.URL, error) {
	return f.fetch(ctx, raw, pageAccept)
}

func (f *Fetcher) FetchPage(ctx context.Context, raw string) (*http.Response, *url.URL, error) {
	return f.fetch(ctx, raw, pageAccept)
}

func (f *Fetcher) FetchImage(ctx context.Context, raw string) (*http.Response, *url.URL, error) {
	return f.fetch(ctx, raw, imageAccept)
}

func (f *Fetcher) fetch(ctx context.Context, raw, accept string) (*http.Response, *url.URL, error) {
	current, err := f.validateTarget(ctx, raw)
	if err != nil {
		return nil, nil, err
	}
	for redirects := 0; ; {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, current.String(), nil)
		if err != nil {
			return nil, nil, ErrNotAllowed
		}
		request.Header.Set("User-Agent", userAgent)
		request.Header.Set("Accept", accept)
		response, err := f.doer.Do(request)
		if err != nil {
			return nil, nil, err
		}
		if !isRedirect(response.StatusCode) {
			return response, current, nil
		}
		location := response.Header.Get("Location")
		response.Body.Close()
		if redirects >= maxRedirects || location == "" {
			return nil, nil, ErrNotAllowed
		}
		locationURL, err := url.Parse(location)
		if err != nil {
			return nil, nil, ErrNotAllowed
		}
		redirectFragment := locationURL.Fragment
		resolved := current.ResolveReference(locationURL)
		next, err := f.validateTarget(ctx, resolved.String())
		if err != nil {
			return nil, nil, err
		}
		if redirectFragment != "" {
			next.Fragment = redirectFragment
			next.RawFragment = locationURL.RawFragment
		}
		current = next
		redirects++
	}
}

func (f *Fetcher) validateTarget(ctx context.Context, raw string) (*url.URL, error) {
	target, err := ValidateURL(raw)
	if err != nil {
		return nil, err
	}
	host := target.Hostname()
	if literal, err := netip.ParseAddr(host); err == nil {
		_, err = ValidateAddresses([]netip.Addr{literal})
		return target, err
	}
	addresses, err := f.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if _, err := ValidateAddresses(addresses); err != nil {
		return nil, err
	}
	return target, nil
}

func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}
