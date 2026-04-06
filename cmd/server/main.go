package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"finance-backend-challenge/internal/config"
	"finance-backend-challenge/internal/db"
	"finance-backend-challenge/internal/domain/dashboard"
	"finance-backend-challenge/internal/domain/record"
	"finance-backend-challenge/internal/domain/user"
	"finance-backend-challenge/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("database connection established")

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	log.Println("database migration completed")

	// User domain
	userRepo := user.NewPostgresRepository(database)
	userService := user.NewService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
	userHandler := user.NewHandler(userService)

	// Record domain
	recordRepo := record.NewPostgresRepository(database)
	recordService := record.NewService(recordRepo)
	recordHandler := record.NewHandler(recordService)

	// Dashboard domain
	dashboardService := dashboard.NewService(database)
	dashboardHandler := dashboard.NewHandler(dashboardService)

	authMiddleware := middleware.Authenticate(userService)
	adminOnlyMiddleware := middleware.AdminOnly()
	analystAbove := middleware.AnalystAndAbove()
	anyRoleMiddleware := middleware.AnyRole()

	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	r.Route("/api/v1", func(r chi.Router) {
		userHandler.RegisterRoutes(r, authMiddleware, adminOnlyMiddleware)
		recordHandler.RegisterRoutes(r, authMiddleware, anyRoleMiddleware, adminOnlyMiddleware)
		dashboardHandler.RegisterRoutes(r, authMiddleware, analystAbove)
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("server starting on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
