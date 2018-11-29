package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mennanov/scalemate/scheduler/models"
)

func TestEnum_String(t *testing.T) {
	e := models.Enum(5)
	assert.Equal(t, "5", e.String())
}
