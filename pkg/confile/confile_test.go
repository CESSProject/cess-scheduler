package confile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	confile := "./conf_test.toml"
	err := NewConfigfile().Parse(confile)
	assert.NoError(t, err)
}
