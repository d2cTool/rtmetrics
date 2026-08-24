package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssign_NilKeepsDest(t *testing.T) {
	dest := "keep"
	Assign(&dest, (*string)(nil))
	assert.Equal(t, "keep", dest)
}

func TestAssign_SetZeroValue(t *testing.T) {
	dest := true
	src := false
	Assign(&dest, &src)
	assert.False(t, dest)
}

func TestAssignFunc_NilKeepsDest(t *testing.T) {
	dest := 300
	require.NoError(t, AssignFunc(&dest, (*string)(nil), ParseIntervalSeconds))
	assert.Equal(t, 300, dest)
}

func TestAssignFunc_Convert(t *testing.T) {
	dest := 300
	src := "2m"
	require.NoError(t, AssignFunc(&dest, &src, ParseIntervalSeconds))
	assert.Equal(t, 120, dest)
}

func TestAssignFunc_Error(t *testing.T) {
	dest := time.Second
	src := "nope"
	err := AssignFunc(&dest, &src, time.ParseDuration)
	require.Error(t, err)
	assert.Equal(t, time.Second, dest)
}
