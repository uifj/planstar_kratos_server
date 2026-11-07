package auth

import (
	"net/http"
	"strings"

	httpm "github.com/go-kratos/kratos/v3/transport/http"
)

// corsAllowMethods is the fixed verb set the console needs.
const corsAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"

// corsAllowHeaders lists the request headers a browser may ask to send.
const corsAllowHeaders = "Content-Type, Authorization, X-Requested-With"

// CORS returns a filter that enforces the configured browser origin allow-list.
// Cookies travel with every request, so the allow-list is exact and never
// widens to "*"; an unknown Origin is rejected with 403 even for safe methods.
//
// This filter owns cross-origin rejection for the whole HTTP server, which is
// why Middleware does not repeat the check.
func CORS(allowOrigins []string) httpm.FilterFunc {
	allowed := make(map[string]struct{}, len(allowOrigins))
	for _, origin := range allowOrigins {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					httpm.DefaultErrorEncoder(w, r, ErrPermissionDenied)
					return
				}
				header := w.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Credentials", "true")
				header.Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				header := w.Header()
				header.Set("Access-Control-Allow-Methods", corsAllowMethods)
				header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
				header.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
