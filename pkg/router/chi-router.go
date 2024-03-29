package router

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	maxRT  = 5 * time.Second   // max time to read request from the client
	maxWR  = 10 * time.Second  // max time to write response to the client
	maxTKA = 120 * time.Second // max time for connections using TCP Keep-Alive
)

type chiRouter struct {
	mux *chi.Mux
}

// NewChiRouter initializes and returns new Router
func NewChiRouter() Router {
	return &chiRouter{
		mux: setUpChi(),
	}
}

func (c *chiRouter) POST(uri string, f func(w http.ResponseWriter, r *http.Request)) {
	c.mux.Post(uri, f)
}

func (c *chiRouter) GET(uri string, f func(w http.ResponseWriter, r *http.Request)) {
	c.mux.Get(uri, f)
}

func (c *chiRouter) PUT(uri string, f func(w http.ResponseWriter, r *http.Request)) {
	c.mux.Put(uri, f)
}

func (c *chiRouter) DELETE(uri string, f func(w http.ResponseWriter, r *http.Request)) {
	c.mux.Delete(uri, f)
}

func (c *chiRouter) SERVE(port string) {

	s := http.Server{
		Addr:         port,
		Handler:      c.mux,
		ReadTimeout:  maxRT,
		WriteTimeout: maxWR,
		IdleTimeout:  maxTKA,
	}

	log.Printf("starting http server on port %s", port)
	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("failed to start server: ", err)
		}
	}()
	log.Print("server started")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
	log.Print("signal closing server received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Print("server shutdown failed: ", err)
	}
	log.Println("server shutdown completed")
}

func setUpChi() *chi.Mux {
	r := chi.NewRouter()
	setMiddlewares(r)
	return r
}

func setMiddlewares(r *chi.Mux) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
}
