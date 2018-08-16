package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/kardianos/service"
	"github.com/zerowidth/gh-shorthand/internal/pkg/config"
	"github.com/zerowidth/gh-shorthand/internal/pkg/rpc"
)

type server struct {
	cfg  config.Config
	stop chan interface{} // external "stop the server" signal
	done chan interface{} // internal "all done, exit" signal
}

// Service returns a service for this rpc server.
func Service(cfg config.Config) service.Service {
	server := server{
		cfg:  cfg,
		stop: make(chan interface{}),
		done: make(chan interface{}),
	}

	sc := service.Config{
		Name:        "gh-shorthand",
		DisplayName: "gh-shorthand",
		Description: "GitHub autocompletion tools for Alfred",
		Arguments:   []string{"server", "run"},
		Option: service.KeyValue{
			"UserService": true, // run as current user, not root
			"RunAtLoad":   true, // run at boot

			// override the default runwait to select on server.err channel, so an
			// error from within the server will terminate the service.
			"RunWait": func() {
				var sigChan = make(chan os.Signal, 3)
				signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
				select {
				case <-sigChan:
				case <-server.done:
				}
			},
		},
	}

	svc, err := service.New(&server, &sc)
	if err != nil {
		log.Fatalf("couldn't create daemon: %s", err.Error())
	}
	return svc
}

func (s *server) Start(svc service.Service) error {
	if len(s.cfg.SocketPath) == 0 {
		return fmt.Errorf("no socket_path configured in %s", config.Filename)
	}

	if len(s.cfg.APIToken) == 0 {
		return fmt.Errorf("no api_token configured in %s", config.Filename)
	}

	logger, err := svc.Logger(nil)
	if err != nil {
		return err
	}

	go func() {
		if err := s.run(logger); err != nil {
			close(s.done)
		}
	}()

	return nil
}

func (s *server) Stop(service.Service) error {
	close(s.stop)
	// let logs from the goroutine make it through:
	<-time.After(100 * time.Millisecond)
	return nil
}

// struct to wrap a service logger with an interface for middleware to use
type mwLogger struct {
	logger service.Logger
}

func (ml mwLogger) Print(v ...interface{}) {
	_ = ml.logger.Info(v...)
}

// run the gh-shorthand RPC server on the configured unix socket path
func (s *server) run(logger service.Logger) error {
	r := chi.NewRouter()

	formatter := middleware.DefaultLogFormatter{
		Logger:  mwLogger{logger: logger},
		NoColor: !service.Interactive(), // double negative! interactive means color.
	}
	r.Use(middleware.RequestLogger(&formatter))

	h := rpc.NewHandler(s.cfg, logger)
	h.Mount(r)

	server := &http.Server{
		Handler:      r,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}

	sock, err := net.Listen("unix", s.cfg.SocketPath)
	if err != nil {
		_ = logger.Error(err)
		return err
	}
	defer func() {
		os.Remove(s.cfg.SocketPath)
	}()

	go func() {
		_ = logger.Infof("server started on %s\n", s.cfg.SocketPath)
		if err := server.Serve(sock); err != nil {
			if err != http.ErrServerClosed {
				_ = logger.Error("server error", err)
				return
			}
		}
		defer close(s.done)
	}()

	// wait for service to be stopped
	<-s.stop

	_ = logger.Infof("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		_ = logger.Error("server shutdown error", err)
	}

	return nil
}
