package main

import (
	"context"
	"golang.org/x/sync/errgroup"
	"os"
	"time"

	"github.com/namsral/flag"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	defaultAppName = "users_counter_app"

	defaultHost            = "0.0.0.0"
	defaultPort            = 9000
	defaultShutdownTimeout = 10 * time.Second
	defaultCacheInterval   = 5 * time.Minute

	defaultLogLevel = "info"
)

// App is the struct that describes whole app.
// Any service (f.e. DataService) should be just added as a field of that structure
type App struct {
	AppConfig
	http *HTTPService
}

var flagErrorHandling = flag.ContinueOnError

// NewApp reads params into a config and creates new App, based on that config
func NewApp() (*App, error) {
	flagset := flag.NewFlagSetWithEnvPrefix(defaultAppName, "", flagErrorHandling)

	config := AppConfig{}

	flagset.StringVar(&config.AppName, "app-name", defaultAppName, "Service name.")
	flagset.StringVar(&config.LogLevel, "log-level", defaultLogLevel, "Log level (debug, info, warn, error)")

	// HTTP
	flagset.StringVar(&config.HTTP.Host, "host", defaultHost, "Host part of listening address.")
	flagset.IntVar(&config.HTTP.Port, "port", defaultPort, "Listening port.")
	flagset.DurationVar(&config.HTTP.ShutdownTimeout, "shutdown-timeout", defaultShutdownTimeout, "Shutdown timeout for http service.")
	flagset.DurationVar(&config.HTTP.CacheInterval, "cache-interval", defaultCacheInterval, "Cache interval for a session.")

	if err := flagset.Parse(os.Args[1:]); err != nil {
		return nil, errors.Wrap(err, "parsing flags")
	}
	if errs := config.Validate(); len(errs) > 0 {
		return nil, errors.Errorf("invalid flag(s): %s", errs)
	}
	return NewAppFrom(config)
}

// NewAppFrom creates new App and init all related to the App things.
func NewAppFrom(config AppConfig) (*App, error) {
	a := &App{
		AppConfig: config,
	}

	if err := a.initializeLogger(); err != nil {
		return nil, errors.Wrap(err, "initializing logger")
	}
	log.Info("parsed config")
	if err := a.initializeHTTPService(); err != nil {
		return nil, errors.Wrap(err, "initializing http service")
	}
	if err := a.http.initRoutes(); err != nil {
		return nil, errors.Wrap(err, "initializing routes")
	}
	return a, nil
}

// Run just runs the application
// ErrorGroup is just a way of graceful start/shutdown of some components
// Here we just start http service
func (a *App) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return errors.Wrap(a.http.Run(ctx), "http service")
	})
	return eg.Wait()
}

func (a *App) initializeHTTPService() error {
	hs, err := NewHTTPService(a.HTTP)
	a.http = hs
	return err
}

func (a *App) initializeLogger() error {
	level, err := log.ParseLevel(a.LogLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logger := log.New()
	logger.SetLevel(level)

	return nil
}
