package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/utils"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
		if err != nil {
			logrus.WithError(err).Error("utils.ConnectDBFromEnv failed")
			return
		}
		defer utils.Close(db)

		amqpConnection, err := utils.ConnectAMQPFromEnv(server.AMQPEnvConf)
		if err != nil {
			logrus.WithError(err).Error("utils.ConnectAMQPFromEnv failed")
			return
		}
		defer utils.Close(amqpConnection)

		jwtSecretKey, err := server.JWTSecretKeyFromEnv(server.AccountsEnvConf)
		if err != nil {
			fmt.Printf("server.JWTSecretKeyFromEnv failed: %s", err)
			return
		}

		bCryptCost, err := server.BCryptCostFromEnv(server.AccountsEnvConf)
		if err != nil {
			fmt.Printf("server.BCryptCostFromEnv failed: %s", err)
			return
		}

		accessTokenTTL, err := server.AccessTokenFromEnv(server.AccountsEnvConf)
		if err != nil {
			fmt.Printf("server.AccessTokenFromEnv failed: %s", err)
			return
		}

		refreshTokenTTL, err := server.RefreshTokenFromEnv(server.AccountsEnvConf)
		if err != nil {
			fmt.Printf("server.RefreshTokenFromEnv failed: %s", err)
			return
		}

		logger := logrus.StandardLogger()
		utils.SetLogrusLevelFromEnv(logger)

		accountsServer, err := server.NewAccountsServer(
			server.WithLogger(logger),
			server.WithDBConnection(db),
			server.WithAMQPConsumers(amqpConnection),
			server.WithAMQPProducer(amqpConnection),
			server.WithJWTSecretKey(jwtSecretKey),
			server.WithClaimsInjector(jwtSecretKey),
			server.WithAccessTokenTTL(accessTokenTTL),
			server.WithRefreshTokenTTL(refreshTokenTTL),
			server.WithBCryptCost(bCryptCost),
		)
		if err != nil {
			fmt.Printf("Failed to start gRPC server: %+v\n", err)
			return
		}
		defer utils.Close(accountsServer)

		shutdown := make(chan os.Signal)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		ctx, ctxCancel := context.WithCancel(context.Background())

		go func() {
			<-shutdown
			ctxCancel()
		}()

		accountsServer.Serve(ctx, grpcAddr)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}
