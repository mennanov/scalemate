package conf

import (
	"os"
	"path"

	"github.com/pkg/errors"
)

// AppConf is a configuration definition for Scheduler service.
type AppConf struct {
	NatsAddr        string
	NatsClusterName string
	TLSCertFile     string
	TLSKeyFile      string
	DBUrl           string
	JWTSecretKey    []byte
}

// SchdulerConf is a configuration for Accounts service.
var SchdulerConf *AppConf

func init() {
	conf, err := NewAppConfFromEnv()
	if err != nil {
		panic(err)
	}
	SchdulerConf = conf
}

// NewAppConfFromEnv creates a new instance of AppConf populating its values from environment variables.
func NewAppConfFromEnv() (*AppConf, error) {
	conf := &AppConf{
		NatsAddr:        os.Getenv("NATS_ADDR"),
		NatsClusterName: os.Getenv("NATS_CLUSTER"),
		TLSCertFile:     os.Getenv("SCHEDULER_TLS_CERT_FILE"),
		TLSKeyFile:      os.Getenv("SCHEDULER_TLS_KEY_FILE"),
		DBUrl:           os.Getenv("SCHEDULER_DB_URL"),
		JWTSecretKey:    []byte(os.Getenv("JWT_SECRET_KEY")),
	}

	if conf.NatsAddr == "" {
		return nil, errors.New("NATS_ADDR env variable is not set")
	}

	if conf.NatsClusterName == "" {
		return nil, errors.New("NATS_CLUSTER env variable is not set")
	}

	if conf.TLSCertFile == "" {
		return nil, errors.New("SCHEDULER_TLS_CERT_FILE env variable is not set")
	}

	if conf.TLSKeyFile == "" {
		return nil, errors.New("SCHEDULER_TLS_KEY_FILE env variable is not set")
	}

	// Bazel specific path to data files. See https://docs.bazel.build/versions/master/build-ref.html#data
	srcDir := os.Getenv("TEST_SRCDIR")

	if srcDir != "" {
		conf.TLSCertFile = path.Join(srcDir, conf.TLSCertFile)
		conf.TLSKeyFile = path.Join(srcDir, conf.TLSKeyFile)
	}

	if conf.DBUrl == "" {
		return nil, errors.New("SCHEDULER_DB_URL env variable is not set")
	}

	if len(conf.JWTSecretKey) == 0 {
		return nil, errors.New("JWT_SECRET_KEY env variable is not set")
	}

	return conf, nil
}
