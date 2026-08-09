package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"task-manager/internal/cache"
	"task-manager/internal/config"
	"task-manager/internal/db"
	"task-manager/internal/handlers"
	"task-manager/internal/middleware"
	redisclient "task-manager/internal/redis"
	"task-manager/internal/repository"
	"task-manager/internal/service"
	"task-manager/internal/telemetry"
)

func main() {
	cfg := config.Load()

	postgresDB, err := db.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer postgresDB.Close()

	redisClient, err := redisclient.NewRedis(cfg.RedisAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()

	telemetryShutdown, err := telemetry.Init(context.Background(), telemetry.Config{
		Enabled:          cfg.OTelEnabled,
		ServiceName:      cfg.OTelServiceName,
		ExporterEndpoint: cfg.OTelExporterEndpoint,
		ExporterInsecure: cfg.OTelExporterInsecure,
	})
	if err != nil {
		log.Fatal(err)
	}

	middleware.RegisterMetrics()

	authService := service.NewAuthService(cfg.JWTSecret)
	userRepo := repository.NewUserRepository(postgresDB)
	teamRepo := repository.NewTeamRepository(postgresDB)
	taskRepo := repository.NewTaskRepository(postgresDB)
	reportRepo := repository.NewReportRepository(postgresDB)
	taskPolicy := service.NewTaskPolicy(teamRepo)
	taskCache := cache.NewTaskCache(redisClient, 5*time.Minute)
	teamService := service.NewTeamService(teamRepo)
	taskService := service.NewTaskService(taskRepo, taskPolicy, taskCache)

	authHandler := handlers.NewAuthHandler(userRepo, authService)
	teamHandler := handlers.NewTeamHandler(teamService)
	taskHandler := handlers.NewTaskHandler(taskService)
	reportHandler := handlers.NewReportHandler(reportRepo)

	r := chi.NewRouter()

	r.Use(middleware.Metrics)
	r.Use(telemetry.RouteSpan)

	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.With(middleware.IPRateLimit(redisClient)).Post("/register", authHandler.Register)
		r.With(middleware.IPRateLimit(redisClient)).Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))
			r.Use(middleware.RateLimit(redisClient))

			r.Get("/me", authHandler.Me)

			r.Post("/teams", teamHandler.Create)
			r.Get("/teams", teamHandler.List)
			r.Get("/teams/{id}/members", teamHandler.Members)
			r.Post("/teams/{id}/invite", teamHandler.Invite)
			r.Delete("/teams/{id}/members/{user_id}", teamHandler.RemoveMember)
			r.Post("/teams/{id}/leave-requests", teamHandler.RequestLeave)
			r.Get("/teams/{id}/leave-requests", teamHandler.LeaveRequests)
			r.Get("/teams/{id}/leave-request", teamHandler.OwnLeaveRequest)
			r.Post("/teams/{id}/leave-requests/{request_id}/decision", teamHandler.ResolveLeaveRequest)

			r.Post("/tasks", taskHandler.Create)
			r.Get("/tasks", taskHandler.List)
			r.Get("/tasks/{id}", taskHandler.Get)
			r.Put("/tasks/{id}", taskHandler.Update)
			r.Get("/tasks/{id}/history", taskHandler.History)
			r.Post("/tasks/{id}/comments", taskHandler.CreateComment)
			r.Get("/tasks/{id}/comments", taskHandler.ListComments)

			r.Get("/reports/team-stats", reportHandler.TeamStats)
			r.Get("/reports/top-users", reportHandler.TopUsers)
			r.Get("/reports/invalid-assignees", reportHandler.InvalidAssignees)
		})
	})

	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: telemetry.HTTPHandler(r, cfg.OTelServiceName),
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Println("server started on port", cfg.AppPort)
		serverErrors <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server failed: %v", err)
		}
	}

	log.Println("shutting down server...")

	serverCtx, cancelServer := context.WithTimeout(
		context.Background(),
		10*time.Second)

	if err := server.Shutdown(serverCtx); err != nil {
		log.Printf("HTTP server shutdown failed: %v", err)
	}
	cancelServer()

	telemetryCtx, cancelTelemetry := context.WithTimeout(context.Background(), 10*time.Second)
	if err := telemetryShutdown(telemetryCtx); err != nil {
		log.Printf("telemetry shutdown failed: %v", err)
	}
	cancelTelemetry()

	log.Println("server stopped")
}
