package models_test

import (
	"testing"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/stretchr/testify/assert"
)

func TestEnum_String(t *testing.T) {
	e := models.Enum(5)
	assert.Equal(t, "5", e.String())
}
