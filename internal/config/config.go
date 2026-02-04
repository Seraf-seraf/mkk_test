package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "configs/config.yml"

// Config описывает конфигурацию приложения.
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	Redis      RedisConfig      `yaml:"redis"`
	Metrics    MetricsConfig    `yaml:"metrics"`
	Slogger    SloggerConfig    `yaml:"slogger"`
	Migrations MigrationsConfig `yaml:"migrations"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type MySQLConfig struct {
	Host     string          `yaml:"host"`
	Port     int             `yaml:"port"`
	Database string          `yaml:"database"`
	User     string          `yaml:"user"`
	Password string          `yaml:"password"`
	Pool     MySQLPoolConfig `yaml:"pool"`
}

type MySQLPoolConfig struct {
	MaxOpenConns           int `yaml:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `yaml:"conn_max_lifetime_seconds"`
	ConnMaxIdleTimeSeconds int `yaml:"conn_max_idle_time_seconds"`
}

type RedisConfig struct {
	Host string          `yaml:"host"`
	Port int             `yaml:"port"`
	Pool RedisPoolConfig `yaml:"pool"`
}

type RedisPoolConfig struct {
	Size           int `yaml:"size"`
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type SloggerConfig struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	AddSource bool   `yaml:"add_source"`
	Output    string `yaml:"output"`
}

type MigrationsConfig struct {
	Auto bool `yaml:"auto"`
}

// Load читает и парсит YAML конфигурацию. Если путь пустой, используется DefaultPath.
func Load(path string) (*Config, error) {
	const methodCtx = "config.Load"

	if path == "" {
		path = DefaultPath
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s: файл конфигурации не найден: %s", methodCtx, path)
		}
		return nil, fmt.Errorf("%s: ошибка проверки файла конфигурации: %w", methodCtx, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: ошибка чтения файла конфигурации: %w", methodCtx, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: ошибка разбора конфигурации: %w", methodCtx, err)
	}

	return &cfg, nil
}

// LoadDefault читает конфигурацию из DefaultPath.
func LoadDefault() (*Config, error) {
	const methodCtx = "config.LoadDefault"

	cfg, err := Load(DefaultPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	return cfg, nil
}
