package logger

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Seraf-seraf/mkk_test/internal/config"
)

func TestNewDefault(t *testing.T) {
	const methodCtx = "logger.TestNewDefault"

	log, err := New(config.SloggerConfig{})
	require.NoError(t, err, methodCtx)
	require.NotNil(t, log, methodCtx)
}

func TestNewInvalidFormat(t *testing.T) {
	const methodCtx = "logger.TestNewInvalidFormat"

	_, err := New(config.SloggerConfig{Format: "xml"})
	require.Error(t, err, methodCtx)
}

func TestParseLevel(t *testing.T) {
	const methodCtx = "logger.TestParseLevel"

	_, err := parseLevel("debug")
	require.NoError(t, err, methodCtx)

	_, err = parseLevel("unknown")
	require.Error(t, err, methodCtx)
}

func TestOutputWriter(t *testing.T) {
	const methodCtx = "logger.TestOutputWriter"

	_, err := outputWriter("stdout")
	require.NoError(t, err, methodCtx)

	_, err = outputWriter("file")
	require.Error(t, err, methodCtx)
}
