package cors

import (
	"net/http"
	"slices"

	"go.uber.org/zap"
)

func CORSMiddleware(next http.HandlerFunc, logger *zap.Logger, allowOrigin []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if slices.Contains(allowOrigin, "*") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if slices.Contains(allowOrigin, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			logger.Warn("CORS request from disallowed origin", zap.String("origin", origin))
			http.Error(w, "CORS not allowed", http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}
