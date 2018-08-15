package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/kardianos/service"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"github.com/zerowidth/gh-shorthand/internal/pkg/rpc"
)

type server struct {
	cfg  config.Config
	stop chan interface{}
	exit chan interface{}
}

// Service returns a service for this rpc server.
func Service(cfg config.Config) service.Service {
	sc := service.Config{
		Name:        "gh-shorthand",
		DisplayName: "gh-shorthand",
		Description: "GitHub autocompletion tools for Alfred",
		Arguments:   []string{"server", "run"},
		Option: service.KeyValue{
			"UserService": true, // run as current user, not root
			"RunAtLoad":   true, // run at boot
		},
	}

	server := server{
		cfg:  cfg,
		stop: make(chan interface{}), // stop the server
		exit: make(chan interface{}), // server's ready to exit
	}

	svc, err := service.New(&server, &sc)
	if err != nil {
		log.Fatalf("couldn't create daemon: %s", err.Error())
	}
	return svc
}

func (s *server) Start(service.Service) error {
	if len(s.cfg.SocketPath) == 0 {
		return fmt.Errorf("no socket_path configured in %s", config.Filename)
	}

	if len(s.cfg.APIToken) == 0 {
		return fmt.Errorf("no api_token configured in %s", config.Filename)
	}

	go s.run()

	return nil
}

func (s *server) Stop(service.Service) error {
	// these two channels allows the main goroutine to shut down cleanly.
	close(s.stop)
	<-s.exit
	return nil
}

// run the gh-shorthand RPC server on the configured unix socket path
func (s *server) run() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	h := rpc.NewHandler(s.cfg)
	h.Mount(r)

	server := &http.Server{
		Handler:      r,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}

	sock, err := net.Listen("unix", s.cfg.SocketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		os.Remove(s.cfg.SocketPath)
	}()

	go func() {
		log.Printf("server started on %s\n", s.cfg.SocketPath)
		if err := server.Serve(sock); err != nil {
			if err != http.ErrServerClosed {
				log.Fatal("server error", err)
			}
		}
	}()

	// wait for service to be stopped
	<-s.stop

	log.Printf("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("server shutdown error", err)
	}

	close(s.exit) // signal service.Stop that we're done
}
