package web

import (
	"net/http"
	"net/url"
	"strings"
)

// SecurityMiddleware wraps an http.Handler with OWASP security headers and fine-grained CORS validation.
func SecurityMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	isOriginAllowed := func(origin string) bool {
		parsedOrigin, err := parseOrigin(origin)
		if err != nil {
			return false
		}
		for _, allowed := range allowedOrigins {
			allowed = strings.TrimSpace(allowed)
			if allowed == "*" || strings.EqualFold(allowed, origin) {
				return true
			}
			if wildcardOriginMatches(parsedOrigin, allowed) {
				return true
			}
		}
		return false
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		if origin != "" {
			allowed = isOriginAllowed(origin)
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Upstream-Url, X-Upstream-Key, X-API-Key, X-Goog-Api-Key, Anthropic-Version")
			if strings.EqualFold(r.Header.Get("Access-Control-Request-Private-Network"), "true") {
				w.Header().Set("Access-Control-Allow-Private-Network", "true")
			}
		}

		// Security Headers (OWASP recommendations)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if r.Method == http.MethodOptions {
			if origin != "" && !allowed {
				http.Error(w, "CORS origin denied", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseOrigin(raw string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, http.ErrNotSupported
	}
	if parsed.Path != "" {
		return nil, http.ErrNotSupported
	}
	return parsed, nil
}

func wildcardOriginMatches(origin *url.URL, allowed string) bool {
	if !strings.HasSuffix(allowed, "*") {
		return false
	}
	if strings.HasSuffix(allowed, ":*") {
		base, err := url.ParseRequestURI(strings.TrimSuffix(allowed, ":*"))
		return err == nil && strings.EqualFold(origin.Scheme, base.Scheme) && strings.EqualFold(origin.Hostname(), base.Hostname())
	}
	if strings.HasSuffix(allowed, "://*") {
		return strings.EqualFold(origin.Scheme, strings.TrimSuffix(allowed, "://*")) && origin.Hostname() != ""
	}
	return false
}
