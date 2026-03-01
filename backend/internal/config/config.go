package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig       `yaml:"server" json:"server"`
	Worker       WorkerConfig       `yaml:"worker" json:"worker"`
	Database     DatabaseConfig     `yaml:"database" json:"database"`
	Auth         AuthConfig         `yaml:"auth" json:"auth"`
	Security     SecurityConfig     `yaml:"security" json:"security"`
	Runbooks     RunbooksConfig     `yaml:"runbooks" json:"runbooks"`
	Fingerprint  FingerprintConfig  `yaml:"fingerprint" json:"fingerprint"`
	Limits       LimitsConfig       `yaml:"limits" json:"limits"`
	Sources      SourcesConfig      `yaml:"sources" json:"sources"`
	Destinations DestinationsConfig `yaml:"destinations" json:"destinations"`
	Routing      RoutingConfig      `yaml:"routing" json:"routing"`
}

type ServerConfig struct {
	Addr         string        `yaml:"addr" json:"addr"`
	ReadTimeout  time.Duration `yaml:"readTimeout" json:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout" json:"writeTimeout"`
}

type WorkerConfig struct {
	Interval       time.Duration `yaml:"interval" json:"interval"`
	BatchSize      int           `yaml:"batchSize" json:"batchSize"`
	MaxConcurrency int           `yaml:"maxConcurrency" json:"maxConcurrency"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver" json:"driver"`
	DSN    string `yaml:"dsn" json:"dsn"`
}

type AuthConfig struct {
	Mode      string `yaml:"mode" json:"mode"`
	BasicUser string `yaml:"basicUser" json:"basicUser"`
	BasicPass string `yaml:"basicPass" json:"basicPass"`
	JWTSecret string `yaml:"jwtSecret" json:"jwtSecret"`
}

type SecurityConfig struct {
	EgressAllowlist      []string      `yaml:"egressAllowlist" json:"egressAllowlist"`
	RequestTimeout       time.Duration `yaml:"requestTimeout" json:"requestTimeout"`
	RetryCount           int           `yaml:"retryCount" json:"retryCount"`
	CircuitBreakerFails  int           `yaml:"circuitBreakerFails" json:"circuitBreakerFails"`
	CircuitBreakerWindow time.Duration `yaml:"circuitBreakerWindow" json:"circuitBreakerWindow"`
	SettingsCryptoKeyB64 string        `yaml:"settingsCryptoKeyB64" json:"settingsCryptoKeyB64"`
}

type RunbooksConfig struct {
	Mode string `yaml:"mode" json:"mode"`
	Path string `yaml:"path" json:"path"`
}

type FingerprintConfig struct {
	Fields []string `yaml:"fields" json:"fields"`
}

type LimitsConfig struct {
	StepTimeout   time.Duration `yaml:"stepTimeout" json:"stepTimeout"`
	DedupCooldown time.Duration `yaml:"dedupCooldown" json:"dedupCooldown"`
	MaxSteps      int           `yaml:"maxSteps" json:"maxSteps"`
}

type SourcesConfig struct {
	Alertmanager []SourceConfig `yaml:"alertmanager" json:"alertmanager"`
}

type SourceConfig struct {
	Name     string `yaml:"name" json:"name"`
	Receiver string `yaml:"receiver" json:"receiver"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
}

type DestinationsConfig struct {
	Telegram   []TelegramDestination   `yaml:"telegram" json:"telegram"`
	Mattermost []MattermostDestination `yaml:"mattermost" json:"mattermost"`
}

type TelegramDestination struct {
	ID       string `yaml:"id" json:"id"`
	Name     string `yaml:"name" json:"name"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	BotToken string `yaml:"botToken" json:"botToken"`
	ChatID   string `yaml:"chatId" json:"chatId"`
	TopicID  int64  `yaml:"topicId" json:"topicId"`
}

type MattermostDestination struct {
	ID         string `yaml:"id" json:"id"`
	Name       string `yaml:"name" json:"name"`
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	WebhookURL string `yaml:"webhookUrl" json:"webhookUrl"`
	Channel    string `yaml:"channel" json:"channel"`
}

type RoutingConfig struct {
	Rules []RoutingRule `yaml:"rules" json:"rules"`
}

type RoutingRule struct {
	Name         string            `yaml:"name" json:"name"`
	RouteKey     string            `yaml:"routeKey" json:"routeKey"`
	Enabled      bool              `yaml:"enabled" json:"enabled"`
	Priority     int               `yaml:"priority" json:"priority"`
	MatchLabels  map[string]string `yaml:"matchLabels" json:"matchLabels"`
	Destinations []string          `yaml:"destinations" json:"destinations"`
}

func BuiltInDefaults() Config {
	return Config{
		Server: ServerConfig{
			Addr:         ":8080",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Worker: WorkerConfig{
			Interval:       30 * time.Second,
			BatchSize:      20,
			MaxConcurrency: 4,
		},
		Database: DatabaseConfig{
			Driver: "postgres",
			DSN:    "postgres://postgres:postgres@localhost:5432/runbook_hunter?sslmode=disable",
		},
		Auth: AuthConfig{
			Mode: "basic",
		},
		Security: SecurityConfig{
			EgressAllowlist:      []string{"localhost", "127.0.0.1", "*.svc.cluster.local"},
			RequestTimeout:       5 * time.Second,
			RetryCount:           2,
			CircuitBreakerFails:  3,
			CircuitBreakerWindow: 1 * time.Minute,
		},
		Runbooks: RunbooksConfig{
			Mode: "file",
			Path: "/runbooks",
		},
		Fingerprint: FingerprintConfig{
			Fields: []string{"alertname", "service", "env", "instance|job"},
		},
		Limits: LimitsConfig{
			StepTimeout:   5 * time.Second,
			DedupCooldown: 5 * time.Minute,
			MaxSteps:      3,
		},
		Sources: SourcesConfig{
			Alertmanager: []SourceConfig{{Name: "default", Receiver: "default", Enabled: true}},
		},
		Destinations: DestinationsConfig{},
		Routing:      RoutingConfig{Rules: []RoutingRule{}},
	}
}

func Load(path string) (Config, error) {
	cfg := BuiltInDefaults()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
	}
	applyEnvOverrides(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	overrides := map[string]*string{
		"RH_HTTP_ADDR":           &cfg.Server.Addr,
		"RH_DATABASE_DRIVER":     &cfg.Database.Driver,
		"RH_DATABASE_DSN":        &cfg.Database.DSN,
		"RH_AUTH_MODE":           &cfg.Auth.Mode,
		"RH_AUTH_BASIC_USER":     &cfg.Auth.BasicUser,
		"RH_AUTH_BASIC_PASS":     &cfg.Auth.BasicPass,
		"RH_AUTH_JWT_SECRET":     &cfg.Auth.JWTSecret,
		"RH_RUNBOOK_MODE":        &cfg.Runbooks.Mode,
		"RH_RUNBOOK_PATH":        &cfg.Runbooks.Path,
		"RH_SETTINGS_CRYPTO_KEY": &cfg.Security.SettingsCryptoKeyB64,
	}
	for env, ptr := range overrides {
		if value := os.Getenv(env); value != "" {
			*ptr = value
		}
	}
}

func (c Config) Validate() error {
	if c.Database.Driver != "postgres" && c.Database.Driver != "sqlite" {
		return errors.New("database.driver must be postgres or sqlite")
	}
	if c.Auth.Mode == "basic" && (strings.TrimSpace(c.Auth.BasicUser) == "" || strings.TrimSpace(c.Auth.BasicPass) == "") {
		return errors.New("auth.basic user/password required in basic mode")
	}
	if c.Auth.Mode == "jwt" && strings.TrimSpace(c.Auth.JWTSecret) == "" {
		return errors.New("auth.jwt secret required in jwt mode")
	}
	if c.Runbooks.Mode != "file" && c.Runbooks.Mode != "db" {
		return errors.New("runbooks.mode must be file or db")
	}
	if c.Limits.MaxSteps <= 0 {
		return errors.New("limits.maxSteps must be > 0")
	}
	return nil
}
