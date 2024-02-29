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
}

func main() {
	app := &application{
		config: configure(),
	}

	err := app.serve()
	if err != nil {
		fmt.Println(err)
	}
}

func (app *application) serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		ErrorLog:     log.New(os.Stdout, "", 0),
		Handler:      initRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Server is starting on port %d...\n", app.config.port)

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
