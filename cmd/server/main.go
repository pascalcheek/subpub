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

	bus := subpub.New()
	grpcService := service.New(bus)
	grpcServer := grpc.NewServer()

	grpcService.Register(grpcServer)

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := bus.Close(ctx); err != nil {
		logger.Error("failed to close bus", zap.Error(err))
	}
	server.GracefulStop()
}
