package app

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/Seraf-seraf/mkk_test/internal/tests"
)

type AppSuite struct {
	tests.IntegrationSuite
}

func TestAppSuite(t *testing.T) {
	const methodCtx = "app.TestAppSuite"

	t.Log(methodCtx)
	suite.Run(t, new(AppSuite))
}

func (s *AppSuite) SetupSuite() {
	s.IntegrationSuite.SetupSuite()
}

func (s *AppSuite) TestBuildAPIMiddlewares() {
	const methodCtx = "app.AppSuite.TestBuildAPIMiddlewares"

	publicMW, apiMW, optionalMW, err := buildAPIMiddlewares(s.Config)
	require.NoError(s.T(), err, methodCtx)
	require.NotEmpty(s.T(), publicMW, methodCtx)
	require.NotEmpty(s.T(), apiMW, methodCtx)
	require.NotEmpty(s.T(), optionalMW, methodCtx)
}

func (s *AppSuite) TestRunMigrations() {
	const methodCtx = "app.AppSuite.TestRunMigrations"

	require.NoError(s.T(), runMigrations(s.DB), methodCtx)
}

func (s *AppSuite) TestNewAndShutdown() {
	const methodCtx = "app.AppSuite.TestNewAndShutdown"

	cfg := *s.Config
	cfg.Migrations.Auto = false

	server, shutdown, err := New(&cfg)
	require.NoError(s.T(), err, methodCtx)
	require.NotNil(s.T(), server, methodCtx)
	require.NotNil(s.T(), shutdown, methodCtx)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.T(), err, methodCtx)

	go func() {
		_ = server.Serve(listener)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = shutdown(ctx)
	cancel()
	require.NoError(s.T(), err, methodCtx)
}
