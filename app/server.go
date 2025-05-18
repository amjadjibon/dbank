package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/amjadjibon/dbank/app/accounts"
	"github.com/amjadjibon/dbank/app/consumer"
	"github.com/amjadjibon/dbank/app/store"
	"github.com/amjadjibon/dbank/app/swagger"
	"github.com/amjadjibon/dbank/app/transactions"
	"github.com/amjadjibon/dbank/conf"
	dbankv1 "github.com/amjadjibon/dbank/gen/go/dbank/v1"
	"github.com/amjadjibon/dbank/pkg/amqpx"
	"github.com/amjadjibon/dbank/pkg/dbx"
	"github.com/amjadjibon/dbank/pkg/log"
	"github.com/amjadjibon/dbank/pkg/mongox"
	"github.com/amjadjibon/dbank/pkg/redisx"
)

type Server struct {
	logger         *slog.Logger
	grpcListener   net.Listener
	grpcServer     *grpc.Server
	httpServer     *http.Server
	consumer       *consumer.Consumer
	rabbitmqClient *amqpx.RabbitMQClient
}

func NewServer(
	ctx context.Context,
	cfg *conf.Config,
) (*Server, error) {
	logger := log.GetLogger(cfg.LogLevel)

	db, err := dbx.NewPostgres(cfg.DbURL,
		dbx.MaxPoolSize(10),
		dbx.ConnAttempts(10),
		dbx.ConnTimeout(1*time.Second),
	)
	if err != nil {
		return nil, err
	}
	if err = db.Pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.InfoContext(ctx, "connected to database",
		"db_url", cfg.DbURL,
	)

	redisClient, err := redisx.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	if err = redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}
	logger.InfoContext(ctx, "connected to Redis",
		"redis_url", cfg.RedisURL,
	)
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close Redis client",
				"error", err,
			)
		}
	}()

	mongoClient, err := mongox.NewMongoClient(ctx, cfg.MongoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err = mongoClient.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	logger.InfoContext(ctx, "connected to MongoDB",
		"mongo_url", cfg.MongoURL,
	)

	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			logger.ErrorContext(ctx, "failed to disconnect MongoDB client",
				"error", err,
			)
		}
	}()

	// Initialize RabbitMQ client
	rabbitmqClient, err := amqpx.NewRabbitMQClient(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Initialize RabbitMQ exchanges
	if err := rabbitmqClient.EnsureExchange(amqpx.TransactionExchange, "topic"); err != nil {
		return nil, fmt.Errorf("failed to declare RabbitMQ exchange: %w", err)
	}

	logger.InfoContext(ctx, "connected to RabbitMQ",
		"rabbitmq_url", cfg.RabbitMQURL,
	)

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	store := store.NewStore(db, logger)
	accountsService := accounts.NewService(logger, store)
	transactionsService := transactions.NewService(logger, store, rabbitmqClient)

	dbankv1.RegisterAccountServiceServer(grpcServer, accountsService)
	dbankv1.RegisterTransactionServiceServer(grpcServer, transactionsService)

	reflection.Register(grpcServer)

	mux := runtime.NewServeMux()
	err = dbankv1.RegisterAccountServiceHandlerServer(ctx, mux, accountsService)
	if err != nil {
		return nil, err
	}

	err = dbankv1.RegisterTransactionServiceHandlerServer(ctx, mux, transactionsService)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	router.HandleFunc("/dbank/*", func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	})

	// Add Swagger UI routes
	router.Get("/swagger/", swagger.UI)
	router.Get("/swagger/v1/openapiv2.json", swagger.APIv1)

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

	// Initialize the consumer
	messageConsumer := consumer.NewConsumer(rabbitmqClient, logger)

	// Register transaction event handlers
	messageConsumer.RegisterHandler(amqpx.TransactionSuccessRoute, consumer.ProcessSuccessfulTransaction(logger))
	messageConsumer.RegisterHandler(amqpx.TransactionFailureRoute, consumer.ProcessFailedTransaction(logger))

	return &Server{
		logger:         logger,
		grpcListener:   grpcListener,
		grpcServer:     grpcServer,
		httpServer:     httpServer,
		consumer:       messageConsumer,
		rabbitmqClient: rabbitmqClient,
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

	// Start RabbitMQ consumer
	go func() {
		s.logger.InfoContext(ctx, "starting RabbitMQ consumer...")

		if err := s.consumer.Start(ctx); err != nil {
			errCh <- errors.Join(err, errors.New("failed to start RabbitMQ consumer"))
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

	// Stop the RabbitMQ consumer
	s.consumer.Stop(ctx)

	// Close RabbitMQ connection
	if s.rabbitmqClient != nil {
		if err := s.rabbitmqClient.Close(); err != nil {
			s.logger.ErrorContext(ctx, "failed to close RabbitMQ client", "error", err)
		}
		s.logger.InfoContext(ctx, "RabbitMQ client closed gracefully.")
	}

	return nil
}
