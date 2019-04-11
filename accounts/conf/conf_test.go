package conf_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mennanov/scalemate/accounts/conf"
)

func TestNewAppConfFromEnv(t *testing.T) {
	appConf, err := conf.NewAppConfFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, conf.AccountsConf, appConf)
}
