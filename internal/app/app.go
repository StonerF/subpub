package app

import (
	"log/slog"

	grpcapp "github.com/StonerF/subpub/internal/app/grpc"
	"github.com/StonerF/subpub/internal/service"
	"github.com/StonerF/subpub/internal/subpub"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
) *App {

	SP := subpub.NewSubPub()

	SPService := service.NewSubPubService(SP, log)
	grpcApp := grpcapp.New(log, SPService, grpcPort)

	return &App{
		GRPCServer: grpcApp,
	}
}
