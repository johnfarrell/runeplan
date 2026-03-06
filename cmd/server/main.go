package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/johnfarrell/runeplan/application/catalog"
	appgoal "github.com/johnfarrell/runeplan/application/goal"
	appuser "github.com/johnfarrell/runeplan/application/user"
	"github.com/johnfarrell/runeplan/config"
	"github.com/johnfarrell/runeplan/infrastructure/hiscores"
	"github.com/johnfarrell/runeplan/infrastructure/postgres"
	"github.com/johnfarrell/runeplan/interfaces/handler"
	"github.com/johnfarrell/runeplan/interfaces/middleware"
	"github.com/johnfarrell/runeplan/logger"
	"github.com/johnfarrell/runeplan/static"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.App)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// Run migrations (use pgx5:// scheme for golang-migrate pgx/v5 driver)
	migrateURL := strings.Replace(cfg.DB.URL, "postgres://", "pgx5://", 1)
	if err := postgres.RunMigrations(migrateURL); err != nil {
		log.Sugar().Fatalf("migrations: %v", err)
	}
	log.Info("migrations applied")

	// Connect pgxpool
	pool, err := postgres.Connect(context.Background(), cfg.DB.URL)
	if err != nil {
		log.Sugar().Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	log.Info("database connected")

	// Repositories
	catalogRepo := postgres.NewCatalogRepository(pool)
	goalRepo := postgres.NewGoalRepository(pool)
	userRepo := postgres.NewUserRepository(pool)

	// Services
	catalogSvc := catalog.NewService(catalogRepo)
	goalSvc := appgoal.NewService(goalRepo)
	hiscoresClient := hiscores.NewClient(cfg.Hiscores.BaseURL, cfg.Hiscores.Timeout)
	userSvc := appuser.NewService(hiscoresClient, userRepo)

	// Router
	r := mux.NewRouter()
	r.Use(middleware.Logging(log))
	r.Use(middleware.DevAuth)

	// Static assets
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.FS(static.FS))),
	)

	// Pages
	r.Handle("/", http.RedirectHandler("/browse", http.StatusFound))
	r.Handle("/browse", handler.BrowseHandler(catalogSvc)).Methods(http.MethodGet)
	r.Handle("/browse/catalog/{id}", handler.CatalogDetailHandler(catalogSvc)).Methods(http.MethodGet)
	r.Handle("/planner", handler.PlannerHandler(goalSvc)).Methods(http.MethodGet)
	r.Handle("/profile", handler.ProfileHandler()).Methods(http.MethodGet)

	// HTMX fragments
	r.Handle("/htmx/goals/activate", handler.ActivateGoalHandler(goalSvc)).Methods(http.MethodPost)
	r.Handle("/htmx/goals/{id}/complete", handler.CompleteGoalHandler(goalSvc)).Methods(http.MethodPost)
	r.Handle("/htmx/requirements/{id}/toggle", handler.ToggleRequirementHandler(goalSvc)).Methods(http.MethodPost)
	r.Handle("/htmx/skills", handler.SkillsHandler(goalRepo, catalogRepo)).Methods(http.MethodGet)
	r.Handle("/htmx/sync", handler.SyncHandler(userSvc)).Methods(http.MethodPost)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Sugar().Infof("listening on :%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Sugar().Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Sugar().Errorf("shutdown: %v", err)
	}
}
