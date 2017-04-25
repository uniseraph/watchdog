package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"github.com/omega/watchdog/backends"
	"github.com/omega/watchdog/watchdog"
)

const dockerAPIVersion = "1.23"

func init() {
	if err := os.Setenv("DOCKER_API_VERSION", dockerAPIVersion); err != nil {
		panic(err)
	}
}

var (
	Version   string
	GitCommit string
	BuildTime string
)

type watchdogOptions struct {
	version  bool
	loglevel string
	mode     string
}

func newWatchdogCommand() *cobra.Command {
	var opts watchdogOptions

	cmd := &cobra.Command{
		Use:           "watchdog [flags] address",
		Short:         "watch container status ",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
			if opts.version {
				showVersion()
				return
			}
			if err := setLogLevel(opts.loglevel); err != nil {
				logrus.Fatal(err)
			}
			if len(args) != 1 {
				logrus.Fatal("watchdog [FLAGS] ADDRESS")
			}
			runWatchdog(opts, args[0])
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.version, "version", "v", false, "Print version information and quit")
	flags.StringVar(&opts.loglevel, "log-level", "info", "Set log level (debug, info, error, fatal)")
	flags.StringVarP(&opts.mode, "mode", "m", "docker-compose", "Set service mode: name/docker-compose")

	return cmd
}

func setLogLevel(level string) error {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	return nil
}

func runWatchdog(opts watchdogOptions, address string) {
	dockerClient, err := getDockerClient()
	if err != nil {
		logrus.Fatal("connect to docker error: ", err)
	}

	backend, err := getServiceBackend(address, opts)
	if err != nil {
		logrus.Fatal("connect to service backend error: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	signalTrap(func(sig os.Signal) {
		logrus.Info("handle signal ", sig.String(), ", stop listening and exit")
		cancel()
	})

	if err := watchdog.NewManager(
		watchdog.NewWatchdog(
			backend,
			dockerClient,
			map[string]string{"registrator.service.getter": opts.mode},
		),
		dockerClient,
	).Run(ctx); err != nil {
		logrus.Fatal(err)
	}
}

func getServiceBackend(address string , opts watchdogOptions) (backends.ContainerBackend, error) {
	parts := strings.SplitN(address, "://", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid service backend address")
	}


	newOpts := make(map[string]string)
	newOpts["registrator.service.getter"]=opts.mode
	return backends.New(parts[0], parts[1], newOpts)
}

func getDockerClient() (client.APIClient, error) {
	return client.NewEnvClient()
}

func signalTrap(handle func(os.Signal)) {
	signalC := make(chan os.Signal, 1)

	signal.Notify(signalC, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for sig := range signalC {
			handle(sig)
		}
	}()
}

func showVersion() {
	if t, err := time.Parse(time.RFC3339Nano, BuildTime); err == nil {
		BuildTime = t.Format(time.ANSIC)
	}
	fmt.Printf("Watchdog version %s, build %s, timestamp %s\n", Version, GitCommit, BuildTime)
}

func main() {
	if err := newWatchdogCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}
