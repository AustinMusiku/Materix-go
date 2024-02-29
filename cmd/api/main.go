package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/logger"
)

type config struct {
	port int
	env  string
	log  struct {
		minLevel logger.Level
	}
}

type application struct {
	config config
	logger *logger.Logger
}

func main() {
	config := configure()
	if config.env == "production" {
		config.log.minLevel = logger.LevelInfo
	}
	logger := logger.New(os.Stdout, config.log.minLevel)
	logger.Debug("Starting the application", nil)

	app := &application{
		config: config,
		logger: logger,
	}

	err := app.serve()
	if err != nil {
		logger.Fatal(err, nil)
	}
}

func (app *application) serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		ErrorLog:     log.New(app.logger, "", 0),
		Handler:      initRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	app.logger.Info("Server is starting", map[string]string{
		"addr": server.Addr,
		"env":  app.config.env,
	})

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	} else if err != nil {
		return err
	}

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

	flag.Parse()

	return config
}
