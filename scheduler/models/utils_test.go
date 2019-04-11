package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mennanov/scalemate/shared/utils"
)

func TestEnum_String(t *testing.T) {
	e := utils.Enum(5)
	assert.Equal(t, "5", e.String())
}
