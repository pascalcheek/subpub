package service

import (
	"context"
	"google.golang.org/grpc"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/emptypb"
	pb "subpub/internal/grpc/proto/gen"
	"subpub/internal/subpub"
)

type PubSubService struct {
	pb.UnimplementedPubSubServer
	bus     subpub.SubPub
	mu      sync.Mutex
	streams map[string][]pb.PubSub_SubscribeServer
}

func New(bus subpub.SubPub) *PubSubService {
	return &PubSubService{
		bus:     bus,
		streams: make(map[string][]pb.PubSub_SubscribeServer),
	}
}

func (s *PubSubService) Subscribe(req *pb.SubscribeRequest, stream pb.PubSub_SubscribeServer) error {
	key := req.GetKey()

	sub, err := s.bus.Subscribe(key, func(msg interface{}) {
		data, ok := msg.(string)
		if !ok {
			return
		}

		if err := stream.Send(&pb.Event{Data: data}); err != nil {
			s.removeStream(key, stream)
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}

	s.addStream(key, stream)

	<-stream.Context().Done()
	sub.Unsubscribe()
	s.removeStream(key, stream)

	return nil
}

func (s *PubSubService) Publish(ctx context.Context, req *pb.PublishRequest) (*emptypb.Empty, error) {
	if err := s.bus.Publish(req.GetKey(), req.GetData()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *PubSubService) addStream(key string, stream pb.PubSub_SubscribeServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[key] = append(s.streams[key], stream)
}

func (s *PubSubService) removeStream(key string, stream pb.PubSub_SubscribeServer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	streams, ok := s.streams[key]
	if !ok {
		return
	}

	for i, st := range streams {
		if st == stream {
			s.streams[key] = append(streams[:i], streams[i+1:]...)
			break
		}
	}

	if len(s.streams[key]) == 0 {
		delete(s.streams, key)
	}
}

func (s *PubSubService) Register(server *grpc.Server) {
	pb.RegisterPubSubServer(server, s)
}
