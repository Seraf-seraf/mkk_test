package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	sqlmysql "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
	libredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Seraf-seraf/mkk_test/internal/config"
	httpserver "github.com/Seraf-seraf/mkk_test/internal/pkg/http"
)

type IntegrationSuite struct {
	suite.Suite
	ctx            context.Context
	mysqlC         *mysql.MySQLContainer
	redisC         *rediscontainer.RedisContainer
	DB             *sql.DB
	Redis          *libredis.Client
	Config         *config.Config
	Server         *http.Server
	ServerURL      string
	serverListener net.Listener
	oldWD          string
	repoRoot       string
}

func (s *IntegrationSuite) SetupSuite() {
	const methodCtx = "tests.IntegrationSuite.SetupSuite"

	testcontainers.SkipIfProviderIsNotHealthy(s.T())

	s.ctx = context.Background()

	root, err := findRepoRoot()
	s.Require().NoError(err, methodCtx)
	s.repoRoot = root

	wd, err := os.Getwd()
	s.Require().NoError(err, methodCtx)
	s.oldWD = wd
	s.Require().NoError(os.Chdir(root), methodCtx)

	s.mysqlC, err = mysql.Run(s.ctx, "mysql:8.4.8",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("test"),
		mysql.WithPassword("test"),
		testcontainers.WithWaitStrategy(wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(2*time.Minute)),
	)
	s.Require().NoError(err, methodCtx)

	dsn, err := s.mysqlC.ConnectionString(s.ctx, "parseTime=true", "multiStatements=true")
	s.Require().NoError(err, methodCtx)

	s.DB, err = sql.Open("mysql", dsn)
	s.Require().NoError(err, methodCtx)
	s.Require().NoError(s.DB.PingContext(s.ctx), methodCtx)

	s.runMigrations()

	s.redisC, err = rediscontainer.Run(s.ctx, "redis:7")
	s.Require().NoError(err, methodCtx)

	redisURL, err := s.redisC.ConnectionString(s.ctx)
	s.Require().NoError(err, methodCtx)

	redisOpts, err := libredis.ParseURL(redisURL)
	s.Require().NoError(err, methodCtx)

	s.Redis = libredis.NewClient(redisOpts)
	s.Require().NoError(s.Redis.Ping(s.ctx).Err(), methodCtx)

	s.Config = s.buildConfig(dsn, redisOpts.Addr)

}

func (s *IntegrationSuite) TearDownSuite() {
	const methodCtx = "tests.IntegrationSuite.TearDownSuite"

	if s.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = s.Server.Shutdown(ctx)
		cancel()
	}
	if s.serverListener != nil {
		_ = s.serverListener.Close()
	}
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.redisC != nil {
		_ = s.redisC.Terminate(context.Background())
	}
	if s.mysqlC != nil {
		_ = s.mysqlC.Terminate(context.Background())
	}
	if s.oldWD != "" {
		_ = os.Chdir(s.oldWD)
	}

	if s.T() != nil {
		s.T().Log(methodCtx)
	}
}

func (s *IntegrationSuite) runMigrations() {
	const methodCtx = "tests.IntegrationSuite.runMigrations"

	s.Require().NoError(goose.SetDialect("mysql"), methodCtx)

	goose.SetTableName("goose_db_version")
	s.Require().NoError(goose.Up(s.DB, filepath.Join(s.repoRoot, "internal", "migrations", "schema")), methodCtx)

	goose.SetTableName("goose_db_version_data")
	s.Require().NoError(goose.Up(s.DB, filepath.Join(s.repoRoot, "internal", "migrations", "data")), methodCtx)
}

func (s *IntegrationSuite) buildConfig(dsn string, redisAddr string) *config.Config {
	const methodCtx = "tests.IntegrationSuite.buildConfig"

	cfg, err := sqlmysql.ParseDSN(dsn)
	s.Require().NoError(err, methodCtx)

	host, portStr, err := net.SplitHostPort(cfg.Addr)
	s.Require().NoError(err, methodCtx)
	port, err := strconv.Atoi(portStr)
	s.Require().NoError(err, methodCtx)

	redisHost, redisPortStr, err := net.SplitHostPort(redisAddr)
	s.Require().NoError(err, methodCtx)
	redisPort, err := strconv.Atoi(redisPortStr)
	s.Require().NoError(err, methodCtx)

	return &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 0},
		MySQL: config.MySQLConfig{
			Host:     host,
			Port:     port,
			Database: cfg.DBName,
			User:     cfg.User,
			Password: cfg.Passwd,
		},
		Redis: config.RedisConfig{
			Host: redisHost,
			Port: redisPort,
		},
		Metrics:    config.MetricsConfig{Enabled: false},
		Migrations: config.MigrationsConfig{Auto: false},
		Auth: config.AuthConfig{JWT: config.JWTConfig{
			Secret:           "test-secret",
			AccessTTLMinutes: 30,
		}},
	}
}

func (s *IntegrationSuite) StartHTTPServer(opts httpserver.ServerOptions) {
	const methodCtx = "tests.IntegrationSuite.StartHTTPServer"

	server, err := httpserver.New(s.Config, httpserver.Deps{Redis: s.Redis}, opts)
	s.Require().NoError(err, methodCtx)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err, methodCtx)

	s.Server = server
	s.serverListener = listener
	s.ServerURL = fmt.Sprintf("http://%s", listener.Addr().String())

	go func() {
		_ = server.Serve(listener)
	}()
}

func (s *IntegrationSuite) TruncateTables(tables ...string) {
	const methodCtx = "tests.IntegrationSuite.TruncateTables"

	if len(tables) == 0 {
		return
	}

	_, err := s.DB.ExecContext(s.ctx, "SET FOREIGN_KEY_CHECKS=0")
	s.Require().NoError(err, methodCtx)

	for _, table := range tables {
		_, err = s.DB.ExecContext(s.ctx, fmt.Sprintf("TRUNCATE TABLE %s", table))
		s.Require().NoError(err, methodCtx)
	}

	_, err = s.DB.ExecContext(s.ctx, "SET FOREIGN_KEY_CHECKS=1")
	s.Require().NoError(err, methodCtx)
}

func findRepoRoot() (string, error) {
	const methodCtx = "tests.findRepoRoot"

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("%s: %w", methodCtx, err)
	}

	current := wd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("%s: не найден go.mod", methodCtx)
}
