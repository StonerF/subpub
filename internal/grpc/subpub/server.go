package subpubgrpc

import (
	"context"

	pubsubv1 "github.com/StonerF/subpub/gen/go"
	"github.com/StonerF/subpub/internal/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type serverAPI struct {
	pubsubv1.UnimplementedPubSubServer
	SubPubSl SubPubSl
}

type SubPubSl interface {
	Publish(ctx context.Context, topic string, data interface{}) error
	Subscribe(ctx context.Context, topic string) (chan *model.Message, error)
	Close(ctx context.Context) error
}

func (s *serverAPI) Publish(
	ctx context.Context,
	in *pubsubv1.PublishRequest,
) (*emptypb.Empty, error) {
	// TODO
	if in.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "topic is nil")
	}
	if in.Data == "" {
		return nil, status.Error(codes.InvalidArgument, "data is nil")
	}

	err := s.SubPubSl.Publish(ctx, in.Key, in.Data)

	if err != nil {
		return nil, status.Error(codes.Unknown, "error in Publish method")
	}

	return &emptypb.Empty{}, nil
}

func (s *serverAPI) Subscribe(
	in *pubsubv1.SubscribeRequest,
	stream pubsubv1.PubSub_SubscribeServer,
) error {

	for {

		if stream.Context().Err() != nil {
			s.SubPubSl.Close(stream.Context())
			return stream.Context().Err()
		}

		ch, err := s.SubPubSl.Subscribe(stream.Context(), in.Key)
		if err != nil {
			return err
		}

		select {
		case ms := <-ch:
			err = stream.Send(&pubsubv1.Event{Data: ms.Data.(string)})
			if err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}

	}
}

func Register(gRPC *grpc.Server, SubPub SubPubSl) {
	pubsubv1.RegisterPubSubServer(gRPC, &serverAPI{SubPubSl: SubPub})
}
