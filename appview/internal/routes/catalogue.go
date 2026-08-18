package routes

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	pathpkg "path"
	"sort"
	"strings"
	"unicode"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type V1Catalogue struct {
	policies map[string]RoutePolicy
	routes   []*catalogueRoute
}

type catalogueRoute struct {
	pattern     string
	segments    []catalogueSegment
	methods     map[string]RoutePolicy
	specificity int
}

type catalogueSegment struct {
	literal  string
	wildcard bool
}

type V1Match struct {
	PathPattern string
	Methods     []string
	Policies    map[string]RoutePolicy
}

func NewV1Catalogue(policies []RoutePolicy) (*V1Catalogue, error) {
	catalogue := &V1Catalogue{policies: make(map[string]RoutePolicy)}
	byPattern := make(map[string]*catalogueRoute)
	normalizedPatterns := make(map[string]string)
	for _, policy := range policies {
		if !policy.AccessClass.Valid() {
			return nil, fmt.Errorf("invalid access class for %s %s", policy.Method, policy.PathPattern)
		}
		if !policy.RateClass.Valid() {
			return nil, fmt.Errorf("invalid rate class for %s %s", policy.Method, policy.PathPattern)
		}
		if !policy.BodyKind.Valid() {
			return nil, fmt.Errorf("invalid body kind for %s %s", policy.Method, policy.PathPattern)
		}
		if !validHTTPMethod(policy.Method) {
			return nil, fmt.Errorf("invalid HTTP method %q", policy.Method)
		}
		segments, normalized, specificity, err := compilePathPattern(policy.PathPattern)
		if err != nil {
			return nil, err
		}
		key := policyKey(policy.Method, policy.PathPattern)
		if _, exists := catalogue.policies[key]; exists {
			return nil, fmt.Errorf("duplicate v1 route policy %s", key)
		}
		catalogue.policies[key] = policy
		if prior, exists := normalizedPatterns[normalized]; exists && prior != policy.PathPattern {
			return nil, fmt.Errorf("conflicting v1 path patterns %s and %s", prior, policy.PathPattern)
		}
		normalizedPatterns[normalized] = policy.PathPattern

		route := byPattern[policy.PathPattern]
		if route == nil {
			route = &catalogueRoute{
				pattern:     policy.PathPattern,
				segments:    segments,
				methods:     make(map[string]RoutePolicy),
				specificity: specificity,
			}
			byPattern[policy.PathPattern] = route
			catalogue.routes = append(catalogue.routes, route)
		}
		route.methods[policy.Method] = policy
	}
	return catalogue, nil
}

func (c *V1Catalogue) Match(path string) (V1Match, bool) {
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	var best *catalogueRoute
	for _, route := range c.routes {
		if !route.matches(segments) {
			continue
		}
		if best == nil || route.specificity > best.specificity {
			best = route
		}
	}
	if best == nil {
		return V1Match{}, false
	}
	methods := make([]string, 0, len(best.methods))
	policies := make(map[string]RoutePolicy, len(best.methods))
	for method, policy := range best.methods {
		methods = append(methods, method)
		policies[method] = policy
	}
	sort.Strings(methods)
	return V1Match{PathPattern: best.pattern, Methods: methods, Policies: policies}, true
}

func (c *V1Catalogue) AllowedMethods(path string) ([]string, bool) {
	match, ok := c.Match(path)
	if !ok {
		return nil, false
	}
	return append([]string(nil), match.Methods...), true
}

func (c *V1Catalogue) RoutingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !looksLikeV1Path(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		path, err := canonicalV1Path(r.URL)
		if err != nil {
			middleware.RejectBodyWithoutDrain(w, r)
			observability.RecordRoutePattern(r.Context(), "/v1/*")
			envelope.WriteError(w, http.StatusNotFound, "not_found", "route not found", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		match, ok := c.Match(path)
		if !ok {
			middleware.RejectBodyWithoutDrain(w, r)
			observability.RecordRoutePattern(r.Context(), "/v1/*")
			envelope.WriteError(w, http.StatusNotFound, "not_found", "route not found", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		observability.RecordRoutePattern(r.Context(), match.PathPattern)
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := match.Policies[r.Method]; !ok {
			middleware.RejectBodyWithoutDrain(w, r)
			w.Header().Set("Allow", strings.Join(match.Methods, ", "))
			envelope.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed for this route", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (r *catalogueRoute) matches(pathSegments []string) bool {
	if len(pathSegments) != len(r.segments) {
		return false
	}
	for index, segment := range r.segments {
		if segment.wildcard {
			if pathSegments[index] == "" {
				return false
			}
			continue
		}
		if pathSegments[index] != segment.literal {
			return false
		}
	}
	return true
}

func compilePathPattern(pattern string) ([]catalogueSegment, string, int, error) {
	if !strings.HasPrefix(pattern, "/v1/") || strings.HasSuffix(pattern, "/") || strings.Contains(pattern, "//") {
		return nil, "", 0, fmt.Errorf("non-canonical v1 path pattern %q", pattern)
	}
	parts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	segments := make([]catalogueSegment, 0, len(parts))
	normalized := make([]string, 0, len(parts))
	specificity := len(parts)
	for index, part := range parts {
		if part == "." || part == ".." || part == "" {
			return nil, "", 0, fmt.Errorf("invalid v1 path pattern segment in %q", pattern)
		}
		if strings.HasPrefix(part, "{") || strings.HasSuffix(part, "}") {
			if len(part) < 3 || part[0] != '{' || part[len(part)-1] != '}' || strings.Contains(part[1:len(part)-1], "{") {
				return nil, "", 0, fmt.Errorf("invalid v1 wildcard segment in %q", pattern)
			}
			segments = append(segments, catalogueSegment{wildcard: true})
			normalized = append(normalized, "{}")
			continue
		}
		segments = append(segments, catalogueSegment{literal: part})
		normalized = append(normalized, part)
		// Earlier literal segments dominate wildcard matches at the same depth.
		specificity += 1 << (min(len(parts)-index, 20))
	}
	return segments, strings.Join(normalized, "/"), specificity, nil
}

func validHTTPMethod(method string) bool {
	if method == "" || method == http.MethodOptions || method != strings.ToUpper(method) {
		return false
	}
	for _, character := range method {
		if character > unicode.MaxASCII || !(unicode.IsLetter(character) || character == '-') {
			return false
		}
	}
	return true
}

func canonicalV1Path(value *url.URL) (string, error) {
	if value == nil {
		return "", errors.New("URL is required")
	}
	escaped := value.EscapedPath()
	if value.RawPath != "" {
		decodedRaw, err := url.PathUnescape(value.RawPath)
		if err != nil || decodedRaw != value.Path {
			return "", errors.New("invalid escaped path")
		}
		escaped = value.RawPath
	}
	lowerEscaped := strings.ToLower(escaped)
	for _, forbidden := range []string{"%2f", "%5c", "%00"} {
		if strings.Contains(lowerEscaped, forbidden) {
			return "", errors.New("encoded path separator")
		}
	}
	decoded, err := url.PathUnescape(escaped)
	if err != nil || decoded != value.Path || containsEncodedTriplet(decoded) {
		return "", errors.New("ambiguous escaped path")
	}
	if !strings.HasPrefix(decoded, "/v1/") || strings.HasPrefix(decoded, "//") || strings.Contains(decoded, "//") ||
		strings.ContainsRune(decoded, '\\') || strings.ContainsRune(decoded, '\x00') || strings.HasSuffix(decoded, "/") {
		return "", errors.New("non-canonical v1 path")
	}
	for _, segment := range strings.Split(strings.TrimPrefix(decoded, "/"), "/") {
		if segment == "." || segment == ".." || segment == "" {
			return "", errors.New("non-canonical v1 segment")
		}
	}
	return decoded, nil
}

func looksLikeV1Path(path string) bool {
	trimmed := strings.TrimLeft(path, "/")
	for strings.HasPrefix(trimmed, "./") {
		trimmed = strings.TrimPrefix(trimmed, "./")
	}
	if trimmed == "v1" || strings.HasPrefix(trimmed, "v1/") {
		return true
	}
	cleaned := pathpkg.Clean("/" + strings.TrimLeft(path, "/"))
	return cleaned == "/v1" || strings.HasPrefix(cleaned, "/v1/")
}

func containsEncodedTriplet(value string) bool {
	for index := 0; index+2 < len(value); index++ {
		if value[index] == '%' && isHex(value[index+1]) && isHex(value[index+2]) {
			return true
		}
	}
	return false
}

func isHex(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}

func policyKey(method, path string) string { return method + " " + path }

type PolicyMux struct {
	*http.ServeMux
	catalogue  *V1Catalogue
	registered map[string]struct{}
	extra      []string
}

func NewPolicyMux(mux *http.ServeMux, catalogue *V1Catalogue) *PolicyMux {
	return &PolicyMux{ServeMux: mux, catalogue: catalogue, registered: make(map[string]struct{})}
}

func (m *PolicyMux) Handle(pattern string, handler http.Handler) {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		if pattern == "/v1" || strings.HasPrefix(pattern, "/v1/") {
			m.extra = append(m.extra, "* "+pattern)
		}
	} else if path == "/v1" || strings.HasPrefix(path, "/v1/") {
		key := policyKey(method, path)
		if _, exists := m.catalogue.policies[key]; !exists {
			m.extra = append(m.extra, key)
		} else {
			m.registered[key] = struct{}{}
		}
	}
	m.ServeMux.Handle(pattern, handler)
}

func (m *PolicyMux) Validate() error {
	missing := make([]string, 0)
	for key := range m.catalogue.policies {
		if _, ok := m.registered[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(m.extra)
	if len(missing) == 0 && len(m.extra) == 0 {
		return nil
	}
	return fmt.Errorf("v1 handler-policy mismatch: missing handlers [%s]; handlers without policies [%s]",
		strings.Join(missing, ", "), strings.Join(m.extra, ", "))
}
