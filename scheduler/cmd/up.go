package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/scheduler/event_listeners"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/utils"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Run: func(cmd *cobra.Command, args []string) {
		schedulerServer, connections, err := server.NewSchedulerServerFromEnv(server.SchedulerEnvConf)
		defer utils.Close(schedulerServer)

		if err != nil {
			logrus.WithError(err).Error("failed to create gRPC server from env")
			return
		}

		preparedListeners, err := event_listeners.SetUpAMQPEventListeners(event_listeners.RegisteredEventListeners, connections.AMQP)
		if err != nil {
			logrus.WithError(err).Error("failed to SetUpAMQPEventListeners")
			return
		}

		eventListenersCtx, eventListenersCtxCancel := context.WithCancel(context.Background())
		eventListenersWaitGroup := &sync.WaitGroup{}
		event_listeners.RunEventListeners(eventListenersCtx, eventListenersWaitGroup, preparedListeners, schedulerServer)

		serverCtx, serverCtxCancel := context.WithCancel(context.Background())

		serverStopped := make(chan struct{})
		go func() {
			server.Serve(serverCtx, grpcAddr, schedulerServer)
			serverStopped <- struct{}{}
		}()

		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		select {
		case <-shutdown:
			serverCtxCancel()
			// Wait for the server to gracefully stop.
			<-serverStopped
			eventListenersCtxCancel()
			// Wait for event listeners to safely stop.
			eventListenersWaitGroup.Wait()
		}

	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}
