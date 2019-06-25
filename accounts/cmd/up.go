package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/credentials"

	"github.com/mennanov/scalemate/accounts/conf"
	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.StandardLogger()
		logger.SetLevel(logrus.Level(verbosity))

		creds, err := credentials.NewServerTLSFromFile(conf.AccountsConf.TLSCertFile, conf.AccountsConf.TLSKeyFile)
		if err != nil {
			logger.WithError(err).Fatal("NewServerTLSFromFile failed")
		}

		db, err := utils.ConnectDBFromEnv(conf.AccountsConf.DBUrl)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to DB")
		}
		defer utils.Close(db, logger)

		sc, err := stan.Connect(
			conf.AccountsConf.NatsClusterName,
			fmt.Sprintf("accounts-service-%s", uuid.New().String()),
			stan.NatsURL(conf.AccountsConf.NatsAddr))
		if err != nil {
			logger.WithError(err).Fatal("stan.Connect failed")
		}
		defer utils.Close(sc, logger)

		producer := events.NewNatsProducer(sc, events.AccountsSubjectName, 5)

		accountsServer, err := server.NewAccountsServer(
			server.WithLogger(logger),
			server.WithDBConnection(db),
			server.WithProducer(producer),
			server.WithJWTSecretKey(conf.AccountsConf.JWTSecretKey),
			server.WithClaimsInjector(auth.NewJWTClaimsInjector(conf.AccountsConf.JWTSecretKey)),
			server.WithAccessTokenTTL(conf.AccountsConf.AccessTokenTTL),
			server.WithRefreshTokenTTL(conf.AccountsConf.RefreshTokenTTL),
			server.WithBCryptCost(conf.AccountsConf.BCryptCost),
		)
		if err != nil {
			logger.WithError(err).Fatal("Failed to start gRPC server")
		}

		shutdown := make(chan os.Signal)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		ctx, ctxCancel := context.WithCancel(context.Background())

		go func() {
			<-shutdown
			ctxCancel()
		}()

		accountsServer.Serve(ctx, grpcAddr, creds)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}
