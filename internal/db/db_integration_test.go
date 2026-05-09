//go:build integration

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabase_ValidURL(t *testing.T) {
	assert.NotNil(t, testDB)
	assert.NotNil(t, testDB.pool)
}

func TestNewDatabase_InvalidConnString(t *testing.T) {
	_, err := NewDatabase("not-a-valid-url")
	require.Error(t, err)
}

func TestNewDatabase_UnreachableHost(t *testing.T) {
	_, err := NewDatabase("postgres://user:pass@127.0.0.1:19999/db?connect_timeout=1")
	require.Error(t, err)
}
