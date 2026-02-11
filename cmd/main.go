package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vitistack/dnsdist-worker/internal/api/handlers"
	"github.com/vitistack/dnsdist-worker/internal/api/routes"
	"github.com/vitistack/dnsdist-worker/internal/config"
	"github.com/vitistack/dnsdist-worker/internal/worker"
	"github.com/vitistack/gslb-operator/pkg/auth"
	"github.com/vitistack/gslb-operator/pkg/auth/jwt"
	"github.com/vitistack/gslb-operator/pkg/bslog"
	"github.com/vitistack/gslb-operator/pkg/rest/middleware"
)

func main() {
	cfg := config.GetInstance()
	jwt.InitServiceTokenManager(cfg.JWT().Secret(), cfg.JWT().User())

	dnsdistWorker, err := worker.NewWorker(cfg.Server().DNSDistServers())
	if err != nil {
		bslog.Fatal("unable to create dnsdist worker", slog.String("reason", err.Error()))
	}
	background := context.Background()
	ctx, cancel := context.WithCancel(background) // global cancelling of running jobs

	dnsdistWorker.SynchronizeRemoteConfiguration(ctx) // start sync job of remote configuration

	distHandler, err := handlers.NewDNSDistHandler(dnsdistWorker)
	if err != nil {
		bslog.Fatal("unable to create dnsdist handler", slog.String("reason", err.Error()))
	}

	spoofsHandler := handlers.NewSpoofsHandler(dnsdistWorker)

	api := http.NewServeMux()
	api.HandleFunc(routes.POST_DNSDIST_READY, middleware.Chain(
		middleware.WithContextRequestID(),
		middleware.WithIncomingRequestLogging(slog.Default()),
	)(distHandler.NewServerReady))

	api.HandleFunc(routes.POST_SPOOFS, middleware.Chain(
		middleware.WithContextRequestID(),
		middleware.WithIncomingRequestLogging(slog.Default()),
		auth.WithTokenValidation(slog.Default()),
	)(spoofsHandler.CreateSpoof))

	api.HandleFunc(routes.DELETE_SPOOFS, middleware.Chain(
		middleware.WithContextRequestID(),
		middleware.WithIncomingRequestLogging(slog.Default()),
		auth.WithTokenValidation(slog.Default()),
	)(spoofsHandler.DeleteSpoof))

	// server for accepting http requests
	server := http.Server{
		Addr:    cfg.API().Port(),
		Handler: api,
	}

	serverErr := make(chan error, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	bslog.Info("starting service", slog.String("port", cfg.API().Port()))
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			serverErr <- fmt.Errorf("server failed: %s", err.Error())
		}
	}()

	select {
	case err := <-serverErr:
		bslog.Fatal("server crashed unexpectedly, no longer serving http", slog.String("reason", err.Error()))
	case <-quit:
		bslog.Info("gracefully shutting down...")
		cancel() // stopping all running jobs
	}

	serverCtx, cancel := context.WithTimeout(background, time.Second*5)
	defer cancel()

	if err := server.Shutdown(serverCtx); err != nil {
		panic("error shutting down server: " + err.Error())
	}
}
