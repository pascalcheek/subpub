package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"subpub/internal/config"
	"subpub/internal/grpc/service"
	"subpub/internal/subpub"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Initialize pubsub bus and gRPC service
	bus := subpub.New()
	grpcService := service.New(bus)
	grpcServer := grpc.NewServer()

	// Register the service with the server
	// Note: Replace with your actual registration method from generated code
	// For example: pb.RegisterPubSubServer(grpcServer, grpcService)
	grpcService.Register(grpcServer)

	// Convert port number to string
	portStr := strconv.Itoa(cfg.GRPC.Port)
	lis, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		logger.Info("starting gRPC server", zap.Int("port", cfg.GRPC.Port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("failed to serve", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	waitForShutdown(logger, grpcServer, bus)
}

func waitForShutdown(
	logger *zap.Logger,
	server *grpc.Server,
	bus subpub.SubPub,
) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("received signal, shutting down", zap.String("signal", sig.String()))

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown components
	if err := bus.Close(ctx); err != nil {
		logger.Error("failed to close bus", zap.Error(err))
	}
	server.GracefulStop()
}
