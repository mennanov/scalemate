package server

import (
	"os"

	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

// AMQPEnvConf maps to the name of the env variable with the AMQP address to connect to.
var AMQPEnvConf = utils.AMQPEnvConf{
	Addr: "SHARED_AMQP_ADDR",
}

// TLSEnvConf maps TLS env variables.
var TLSEnvConf = utils.TLSEnvConf{
	CertFile: "SCHEDULER_TLS_CERT_FILE",
	KeyFile:  "SCHEDULER_TLS_KEY_FILE",
}

// DBEnvConf maps to the name of the env variable with the Postgres address to connect to.
var DBEnvConf = utils.DBEnvConf{
	Host:     "SCHEDULER_DB_HOST",
	Port:     "SCHEDULER_DB_PORT",
	User:     "SCHEDULER_DB_USER",
	Name:     "SCHEDULER_DB_NAME",
	Password: "SCHEDULER_DB_PASSWORD",
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

// SchedulerEnvConf maps env variables for the Scheduler service.
var SchedulerEnvConf = AppEnvConf{
	BCryptCost:      "SCHEDULER_BCRYPT_COST",
	AccessTokenTTL:  "SCHEDULER_ACCESS_TOKEN_TTL",
	RefreshTokenTTL: "SCHEDULER_REFRESH_TOKEN_TTL",
	JWTSecretKey:    "SCHEDULER_JWT_SECRET_KEY",
}

// JWTClaimsInjectorFromEnv creates a new auth.JWTClaimsInjector instance from env variables.
func JWTClaimsInjectorFromEnv(conf AppEnvConf) (*auth.JWTClaimsInjector, error) {
	value := os.Getenv(conf.JWTSecretKey)
	if value == "" {
		return nil, errors.New("JWT secret key is empty")
	}
	return auth.NewJWTClaimsInjector([]byte(value)), nil
}
