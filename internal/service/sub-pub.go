package service

import (
	"context"
	"log/slog"

	"github.com/StonerF/subpub/internal/lib/logger/sl"
	"github.com/StonerF/subpub/internal/model"
	"github.com/StonerF/subpub/internal/subpub"
)

type SubPubService struct {
	log *slog.Logger
	Sp  subpub.SubPub
}

func NewSubPubService(sp subpub.SubPub, log *slog.Logger) *SubPubService {
	return &SubPubService{
		Sp:  sp,
		log: log,
	}
}

func (S *SubPubService) Publish(ctx context.Context, topic string, data interface{}) error {

	const op = "Service.Publish"

	log := S.log.With(
		slog.String("op", op),
		slog.String("topic", topic),
		slog.String("data", data.(string)),
	)

	log.Info("attempting to publish message")

	err := S.Sp.Publish(topic, data)

	if err != nil {
		S.log.Error("failed to publish message", sl.Err(err))

		return err
	}

	return nil
}

func (S *SubPubService) Subscribe(ctx context.Context, topic string) (chan *model.Message, error) {

	const op = "Service.Subscribe"

	log := S.log.With(
		slog.String("op", op),
		slog.String("topic", topic),
	)

	log.Info("attempting to subscribe")

	sub, err := S.Sp.Subscribe(topic, func(msg interface{}) {})
	if err != nil {
		S.log.Error("failed to subscribe", sl.Err(err))
		return nil, err
	}

	return sub.GetChannel(), nil

}

func (S *SubPubService) Close(ctx context.Context) error {

	const op = "Service.Close"

	log := S.log.With(
		slog.String("op", op),
	)

	log.Info("attempting to close")

	err := S.Sp.Close(ctx)
	if err != nil {
		S.log.Error("failed to close", sl.Err(err))
		return err
	}
	return nil

}
