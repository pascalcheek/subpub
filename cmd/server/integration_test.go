package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "subpub/internal/grpc/proto/gen"
)

func TestIntegration(t *testing.T) {
	go main()
	time.Sleep(500 * time.Millisecond)

	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPubSubClient(conn)

	t.Run("publish", func(t *testing.T) {
		_, err := client.Publish(context.Background(), &pb.PublishRequest{
			Key:  "test",
			Data: "integration test",
		})
		assert.NoError(t, err)
	})

	t.Run("subscribe", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: "test"})
		assert.NoError(t, err)

		go func() {
			time.Sleep(100 * time.Millisecond)
			client.Publish(context.Background(), &pb.PublishRequest{
				Key:  "test",
				Data: "stream test",
			})
		}()

		msg, err := stream.Recv()
		assert.NoError(t, err)
		assert.Equal(t, "stream test", msg.Data)
	})
}
