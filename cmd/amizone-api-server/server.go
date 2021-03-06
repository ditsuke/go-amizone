package main

import (
	"context"
	"flag"
	"github.com/ditsuke/go-amizone/server"
	"github.com/joho/godotenv"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

const (
	DefaultAddress = ":8081"
	AddressEnvVar  = "AMIZONE_API_ADDRESS"
)

func main() {
	logger := klog.NewKlogr()
	_ = godotenv.Load(".env")

	config := struct{ address string }{}

	flagSet := flag.NewFlagSet("server config", flag.ExitOnError)
	flagSet.StringVar(&config.address, "address", EnvOrDefault(AddressEnvVar, DefaultAddress), "Address to listen on")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		logger.Error(err, "failed to parse flags")
		os.Exit(1)
	}

	s := server.ApiServer{
		Config: &server.Config{
			Logger:   logger,
			BindAddr: config.address,
		},
		Router: http.NewServeMux(),
	}

	// Start the server on a new go-thread
	go func() {
		logger.Info("Starting server", "address", config.address)
		if err := s.Run(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Main thread -- we use for interrupting OS signals
	osChannel := make(chan os.Signal, 1)
	signal.Notify(osChannel, os.Interrupt)
	signal.Notify(osChannel, os.Kill)

	// Block until a signal is received
	sig := <-osChannel

	// Log the signal
	logger.Info("os signal received", "signal", sig)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelFunc()

	if err := s.Stop(ctx); err != nil {
		logger.Error(err, "failed to gracefully shut down serer", err)
	}

	logger.Info("server gracefully shut down")
}

// EnvOrDefault is a generic implementation that returns either the environment variable accessed by `key`
// or the default value.
func EnvOrDefault[T string | int | bool](key string, def T) T {
	env, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	var ret T
	switch p := any(&ret).(type) {
	case *string:
		*p = env
	case *int:
		*p, _ = strconv.Atoi(env)
	case *bool:
		*p, _ = strconv.ParseBool(env)
	}
	return ret
}
