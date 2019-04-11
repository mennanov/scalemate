package conf

import (
	"os"
	"path"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// AppConf is a configuration definition for Accounts service.
type AppConf struct {
	NatsAddr        string
	NatsClusterName string
	TLSCertFile     string
	TLSKeyFile      string
	DBUrl           string
	BCryptCost      int
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	JWTSecretKey    []byte
}

// AccountsConf is a configuration for Accounts service.
var AccountsConf *AppConf

func init() {
	conf, err := NewAppConfFromEnv()
	if err != nil {
		panic(err)
	}
	AccountsConf = conf
}

// NewAppConfFromEnv creates a new instance of AppConf populating its values from environment variables.
func NewAppConfFromEnv() (*AppConf, error) {
	conf := &AppConf{
		NatsAddr:        os.Getenv("NATS_ADDR"),
		NatsClusterName: os.Getenv("NATS_CLUSTER"),
		TLSCertFile:     os.Getenv("ACCOUNTS_TLS_CERT_FILE"),
		TLSKeyFile:      os.Getenv("ACCOUNTS_TLS_KEY_FILE"),
		DBUrl:           os.Getenv("ACCOUNTS_DB_URL"),
		JWTSecretKey:    []byte(os.Getenv("JWT_SECRET_KEY")),
	}

	if conf.NatsAddr == "" {
		return nil, errors.New("NATS_ADDR env variable is not set")
	}

	if conf.NatsClusterName == "" {
		return nil, errors.New("NATS_CLUSTER env variable is not set")
	}

	if conf.TLSCertFile == "" {
		return nil, errors.New("ACCOUNTS_TLS_CERT_FILE env variable is not set")
	}

	if conf.TLSKeyFile == "" {
		return nil, errors.New("ACCOUNTS_TLS_KEY_FILE env variable is not set")
	}

	// Bazel specific path to data files. See https://docs.bazel.build/versions/master/build-ref.html#data
	srcDir := os.Getenv("TEST_SRCDIR")

	if srcDir != "" {
		conf.TLSCertFile = path.Join(srcDir, conf.TLSCertFile)
		conf.TLSKeyFile = path.Join(srcDir, conf.TLSKeyFile)
	}

	if conf.DBUrl == "" {
		return nil, errors.New("ACCOUNTS_DB_URL env variable is not set")
	}

	if len(conf.JWTSecretKey) == 0 {
		return nil, errors.New("JWT_SECRET_KEY env variable is not set")
	}

	var err error

	conf.BCryptCost, err = strconv.Atoi(os.Getenv("ACCOUNTS_BCRYPT_COST"))
	if err != nil {
		return nil, errors.Wrap(err, "strconv.Atoi failed for ACCOUNTS_BCRYPT_COST env variable")
	}

	conf.AccessTokenTTL, err = time.ParseDuration(os.Getenv("ACCOUNTS_ACCESS_TOKEN_TTL"))
	if err != nil {
		return nil, errors.Wrapf(err, "time.ParseDuration failed for ACCOUNTS_ACCESS_TOKEN_TTL env variable")
	}

	conf.RefreshTokenTTL, err = time.ParseDuration(os.Getenv("ACCOUNTS_REFRESH_TOKEN_TTL"))
	if err != nil {
		return nil, errors.Wrapf(err, "time.ParseDuration failed for ACCOUNTS_REFRESH_TOKEN_TTL env variable")
	}

	return conf, nil
}
