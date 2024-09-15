package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/amjadjibon/dbank/conf"
	usersv1 "github.com/amjadjibon/dbank/gen/go/users/v1"
	"github.com/amjadjibon/dbank/log"
	"github.com/amjadjibon/dbank/server/users"
)

type Server struct {
	logger       *slog.Logger
	grpcListener net.Listener
	grpcServer   *grpc.Server
	httpServer   *http.Server
}

func NewServer(
	ctx context.Context,
	cfg *conf.Config,
) (*Server, error) {
	logger := log.GetLogger(cfg.LogLevel)

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	usersService := users.NewService()
	usersv1.RegisterUsersServiceServer(grpcServer, usersService)
	reflection.Register(grpcServer)

	mux := runtime.NewServeMux()
	err := usersv1.RegisterUsersServiceHandlerServer(ctx, mux, usersService)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	router.HandleFunc("/api/*", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.ReplaceAll(r.URL.Path, "/api", "")
		mux.ServeHTTP(w, r)
	})

	httpAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:         httpAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	grpcAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.GrpcPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, err
	}

	return &Server{
		logger:       logger,
		grpcListener: grpcListener,
		grpcServer:   grpcServer,
		httpServer:   httpServer,
	}, nil
}

// Start runs both the gRPC and HTTP servers concurrently.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error)

	go func() {
		s.logger.InfoContext(ctx, "starting gRPC server...",
			"addr", s.grpcListener.Addr(),
		)

		if err := s.grpcServer.Serve(s.grpcListener); err != nil {
			errCh <- errors.Join(err, errors.New("failed to serve gRPC"))
		}
	}()

	// Start HTTP server (gRPC-Gateway)
	go func() {
		s.logger.InfoContext(ctx, "starting HTTP server...",
			"addr", s.httpServer.Addr,
		)

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- errors.Join(err, errors.New("failed to serve HTTP"))
		}
	}()

	// Channel to listen for interrupt signals (for graceful shutdown)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		s.logger.InfoContext(ctx, "received signal, shutting down...",
			"signal", sig,
		)
	}

	return s.Shutdown(ctx)
}

// Shutdown gracefully shuts down both gRPC and HTTP servers.
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return errors.Join(err,
			errors.New("failed to shutdown HTTP server"),
		)
	}
	s.logger.InfoContext(ctx, "HTTP server stopped gracefully.")

	s.grpcServer.GracefulStop()
	s.logger.InfoContext(ctx, "gRPC server stopped gracefully.")
	return nil
}
