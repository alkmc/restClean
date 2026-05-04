package handler

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// NewRouter initializes and returns a new standard library based ServeMux
func NewRouter() *http.ServeMux {
	return http.NewServeMux()
}

// MountRoutes assigns endpoints to the multiplexer and wraps it with middlewares
func MountRoutes(mux *http.ServeMux, h Handler) http.Handler {
	mux.HandleFunc("POST /product", h.Add)
	mux.HandleFunc("GET /product", h.Get)
	mux.HandleFunc("GET /product/{id}", h.GetByID)
	mux.HandleFunc("PUT /product/{id}", h.Update)
	mux.HandleFunc("DELETE /product/{id}", h.Delete)

	return recoverMiddleware(loggerMiddleware(mux))
}

// loggerMiddleware logs incoming HTTP requests
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// We could wrap w to get the status code, but for a simple logger this is sufficient
		log.Printf("request started: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("request completed: %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
	})
}

// recoverMiddleware catches panics and prevents the server from crashing
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v\n%s", err, debug.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
