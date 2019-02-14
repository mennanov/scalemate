package server

import (
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/utils"
)

// AMQPEnvConf maps to the name of the env variable with the AMQP address to connect to.
var AMQPEnvConf = utils.AMQPEnvConf{
	Addr: "SHARED_AMQP_ADDR",
}

// TLSEnvConf maps TLS env variables.
var TLSEnvConf = utils.TLSEnvConf{
	CertFile: "ACCOUNTS_TLS_CERT_FILE",
	KeyFile:  "ACCOUNTS_TLS_KEY_FILE",
}

// DBEnvConf maps to the name of the env variable with the Postgres address to connect to.
var DBEnvConf = utils.DBEnvConf{
	Host:     "ACCOUNTS_DB_HOST",
	Port:     "ACCOUNTS_DB_PORT",
	User:     "ACCOUNTS_DB_USER",
	Name:     "ACCOUNTS_DB_NAME",
	Password: "ACCOUNTS_DB_PASSWORD",
}

// AppEnvConf represents other application settings env variable mapping.
type AppEnvConf struct {
	BCryptCost      string
	AccessTokenTTL  string
	RefreshTokenTTL string
	JWTSecretKey    string
	TLSCert         string
	TLSKey          string
}

// AccountsEnvConf maps env variables for the Accounts service.
var AccountsEnvConf = AppEnvConf{
	BCryptCost:      "ACCOUNTS_BCRYPT_COST",
	AccessTokenTTL:  "ACCOUNTS_ACCESS_TOKEN_TTL",
	RefreshTokenTTL: "ACCOUNTS_REFRESH_TOKEN_TTL",
	JWTSecretKey:    "ACCOUNTS_JWT_SECRET_KEY",
}

// BCryptCostFromEnv gets the BCrypt cost value from environment variables.
func BCryptCostFromEnv(conf AppEnvConf) (int, error) {
	value, err := strconv.Atoi(os.Getenv(conf.BCryptCost))
	if err != nil {
		return 0, errors.Wrap(err, "strconv.Atoi failed")
	}
	return value, nil
}

// AccessTokenFromEnv gets the access token duration value from environment variables.
func AccessTokenFromEnv(conf AppEnvConf) (time.Duration, error) {
	value, err := time.ParseDuration(os.Getenv(conf.AccessTokenTTL))
	if err != nil {
		return 0, errors.Wrapf(err, "time.ParseDuration failed for '%s'", os.Getenv(conf.AccessTokenTTL))
	}
	return value, nil
}

// RefreshTokenFromEnv gets the refresh token duration value from environment variables.
func RefreshTokenFromEnv(conf AppEnvConf) (time.Duration, error) {
	value, err := time.ParseDuration(os.Getenv(conf.RefreshTokenTTL))
	if err != nil {
		return 0, errors.Wrapf(err, "time.ParseDuration failed for '%s'", os.Getenv(conf.RefreshTokenTTL))
	}
	return value, nil
}

// JWTSecretKeyFromEnv gets the JWT secret key value from environment variables.
func JWTSecretKeyFromEnv(conf AppEnvConf) ([]byte, error) {
	value := os.Getenv(conf.JWTSecretKey)
	if value == "" {
		return nil, errors.New("JWT secret key is empty")
	}
	return []byte(value), nil
}

