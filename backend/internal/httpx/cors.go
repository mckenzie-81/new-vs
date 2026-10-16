package httpx

import "net/http"

// CORS allows requests from a single configured origin (the frontend's base
// URL). There's only ever one legitimate caller in this architecture - the
// one Next.js app - so there's no need for a wildcard or a list.
func CORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		// Needed for the session cookie to actually travel with
		// cross-origin fetches (frontend and backend are different
		// origins in dev, and different subdomains in production).
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
