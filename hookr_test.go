package hookr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHookr(t *testing.T) {
	hookr := New()
	assert.NotNil(t, hookr, "Expected hookr to be not nil")
}
