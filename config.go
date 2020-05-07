package main

import (
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Errors that can occur during validation of configs.
var (
	ErrEmptyAppName      = errors.New("app name should be non empty")
	ErrEmptyLogLevel     = errors.New("log level should be non empty ")
	ErrEmptyHost         = errors.New("host should be non empty or be in IPv4 format")
	ErrPortRangeExceeded = errors.New("port should be in rage [1024..49151] inclusively")
)

// AppConfig is whole config for app.
type AppConfig struct {
	AppName  string
	HTTP     HTTPConfig
	Logger   logrus.Logger
	LogLevel string
}

// Validate validates whole app config.
func (c AppConfig) Validate() []error {
	var (
		errs []error
	)

	if len(c.AppName) == 0 {
		errs = append(errs, ErrEmptyAppName)
	}
	if err := c.HTTP.Validate(); err != nil {
		errs = append(errs, err)
	}
	if len(c.LogLevel) == 0 {
		errs = append(errs, ErrEmptyLogLevel)
	}
	return errs
}

// HTTPConfig is config for HTTP service.
type HTTPConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	ShutdownTimeout time.Duration `json:"shutdownTimeout"`
	CacheInterval   time.Duration `json:"cacheInterval"`
}

// Validate just validates http config.
func (c HTTPConfig) Validate() error {
	if len(c.Host) == 0 || len(strings.Split(c.Host, ".")) != 4 {
		return ErrEmptyHost
	}

	// ports below 1024 are using by system
	// ports above 49151 are using by clients
	if c.Port < 1024 || c.Port > 49152 {
		return ErrPortRangeExceeded
	}
	return nil
}
