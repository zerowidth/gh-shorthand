package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"github.com/zerowidth/gh-shorthand/internal/pkg/rpc"
)

// Run runs the gh-shorthand RPC server on the configured unix socket path
func Run(cfg config.Config) {
	if len(cfg.SocketPath) == 0 {
		log.Fatalf("no socket_path configured in %s", config.Filename)
	}

	if len(cfg.APIToken) == 0 {
		log.Fatalf("no api_token configured in %s", config.Filename)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	h := rpc.NewHandler(cfg)
	h.Mount(r)

	server := &http.Server{
		Handler:      r,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	sock, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		os.Remove(cfg.SocketPath)
	}()

	go func() {
		log.Printf("server started on %s\n", cfg.SocketPath)
		if err := server.Serve(sock); err != nil {
			if err != http.ErrServerClosed {
				log.Fatal("server error", err)
			}
		}
	}()

	<-sig

	log.Printf("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("server shutdown error", err)
	}
}
