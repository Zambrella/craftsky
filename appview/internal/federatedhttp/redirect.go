package federatedhttp

import "net/http"

func checkRedirect(profile Profile, policy *Policy) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, via []*http.Request) error {
		if request == nil || request.URL == nil || len(via) == 0 || len(via) > profile.MaxRedirects {
			return &Error{Kind: KindRedirectRejected, Purpose: profile.Purpose}
		}
		target, err := policy.ValidateURL(request.Context(), request.URL.String())
		if err != nil {
			kind := Classify(err)
			if kind == KindDestinationRejected {
				kind = KindRedirectRejected
			}
			return &Error{Kind: kind, Purpose: profile.Purpose, Cause: err}
		}
		origin, err := parseHTTPSURL(via[0].URL.String())
		if err != nil || !sameOrigin(origin, target) {
			return &Error{Kind: KindRedirectRejected, Purpose: profile.Purpose, Cause: err}
		}
		return nil
	}
}
