package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/data"
	"github.com/AustinMusiku/Materix-go/internal/logger"
	_ "github.com/lib/pq"
)

type config struct {
	port int
	env  string
	log  struct {
		minLevel logger.Level
	}
	db struct {
		dsn string
	}
	jwt struct {
		secret string
	}
	cors struct {
		allowedOrigins []string
	}
	limiter struct {
		rps     int
		wl      int
		enabled bool
	}
}

type application struct {
	config config
	logger *logger.Logger
	models data.Models
	wg     sync.WaitGroup
}

func main() {
	config := configure()
	if config.env == "production" {
		config.log.minLevel = logger.LevelInfo
	}
	logger := logger.New(os.Stdout, config.log.minLevel)

	db, err := openDB(config)
	if err != nil {
		logger.Fatal(err, nil)
	}
	defer db.Close()
	logger.Info("Database connection pool established", nil)

	app := &application{
		config: config,
		logger: logger,
		models: data.NewModels(db),
		wg:     sync.WaitGroup{},
	}

	err = app.serve()
	if err != nil {
		logger.Fatal(err, nil)
	}
}

func (app *application) serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		ErrorLog:     log.New(app.logger, "", 0),
		Handler:      app.initRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	app.logger.Info("Server is starting", map[string]string{
		"addr": server.Addr,
		"env":  app.config.env,
	})

	shutdownErr := make(chan error)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		intercepted := <-sigChan

		app.logger.Info("Server is shutting down", map[string]string{
			"signal": intercepted.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			shutdownErr <- err
		}

		app.logger.Info("Waiting for background processes to finish", nil)

		app.wg.Wait()
		shutdownErr <- nil
	}()

	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownErr
	if err != nil {
		return fmt.Errorf("failed to shutdown server gracefully: %w", err)
	}

	app.logger.Info("Server stopped", nil)

	return nil
}

func configure() config {
	var config config

	defaultPort := 4000
	if os.Getenv("PORT") != "" {
		p, err := strconv.Atoi(os.Getenv("PORT"))
		if err == nil {
			defaultPort = p
		}
	}

	flag.IntVar(&config.port, "port", defaultPort, "Application service port")
	flag.StringVar(&config.env, "env", "development", "Environment (development|staging|production)")

	flag.IntVar((*int)(&config.log.minLevel), "log-level", int(logger.LevelDebug), "Minimum log level (0=DEBUG, 1=INFO, 2=WARN, 3=ERROR, 4=FATAL)")

	flag.StringVar(&config.db.dsn, "db-dsn", os.Getenv("DATABASE_URL"), "PostgreSQL DSN")

	flag.StringVar(&config.jwt.secret, "jwt-secret", os.Getenv("JWT_SECRET"), "JWT secret key")

	flag.Func("cors-allowed-origins", "CORS allowed origins", func(val string) error {
		config.cors.allowedOrigins = strings.Fields(val)
		return nil
	})

	flag.IntVar(&config.limiter.rps, "limiter-rps", 10, "Rate limiter requests per second")
	flag.IntVar(&config.limiter.wl, "limiter-wl", 1, "Rate limiter window length in seconds")
	flag.BoolVar(&config.limiter.enabled, "limiter-enabled", false, "Enable rate limiter")

	flag.Parse()

	return config
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
