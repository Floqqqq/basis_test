package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"task-manager/internal/config"
	"task-manager/internal/db"
	"task-manager/internal/notifications"
	"task-manager/internal/repository"
	"task-manager/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Load()
	if cfg.SMTPHost == "" || cfg.SMTPPort <= 0 || cfg.SMTPFrom == "" {
		return fmt.Errorf("SMTP_HOST, SMTP_PORT and SMTP_FROM are required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	postgresDB, err := db.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer postgresDB.Close()

	telemetryShutdown, err := telemetry.Init(ctx, telemetry.Config{
		Enabled:          cfg.OTelEnabled,
		ServiceName:      cfg.OTelServiceName,
		ExporterEndpoint: cfg.OTelExporterEndpoint,
		ExporterInsecure: cfg.OTelExporterInsecure,
	})
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := telemetryShutdown(shutdownCtx); err != nil {
			log.Printf("telemetry shutdown failed: %v", err)
		}
	}()

	notifications.RegisterMetrics()
	metricsServer := &http.Server{
		Addr:              ":" + cfg.WorkerMetricsPort,
		Handler:           promhttp.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	metricsErrors := make(chan error, 1)
	go func() {
		metricsErrors <- metricsServer.ListenAndServe()
	}()

	outbox := repository.NewOutboxRepository(postgresDB)
	data := repository.NewNotificationRepository(postgresDB)
	sender := notifications.NewSMTPEmailSender(notifications.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
	processor := notifications.NewNotificationService(notifications.NewNotificationBuilder(data), sender)
	worker := notifications.NewWorker(outbox, processor, 20, 5, 2*time.Second)

	workerErrors := make(chan error, 1)
	go func() { workerErrors <- worker.Run(ctx) }()
	log.Println("notification worker started")

	workerFinished := false
	var runErr error
	select {
	case <-ctx.Done():
	case err := <-workerErrors:
		workerFinished = true
		if err != nil {
			runErr = fmt.Errorf("notification worker failed: %w", err)
		}
	case err := <-metricsErrors:
		if err != nil && err != http.ErrServerClosed {
			runErr = fmt.Errorf("worker metrics server failed: %w", err)
		}
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown worker metrics server: %w", err)
	}
	if !workerFinished {
		select {
		case err := <-workerErrors:
			if err != nil && runErr == nil {
				runErr = fmt.Errorf("notification worker failed: %w", err)
			}
		case <-shutdownCtx.Done():
			return fmt.Errorf("notification worker shutdown: %w", shutdownCtx.Err())
		}
	}
	log.Println("notification worker stopped")
	return runErr
}
