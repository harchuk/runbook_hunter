package config

import (
	"errors"
	"os"
	"strconv"
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
	Closure      ClosureConfig      `yaml:"closure" json:"closure"`
	GitOps       GitOpsConfig       `yaml:"gitops" json:"gitops"`
	AWX          AWXConfig          `yaml:"awx" json:"awx"`
	Access       AccessConfig       `yaml:"access" json:"access"`
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
	Mode      string     `yaml:"mode" json:"mode"`
	BasicUser string     `yaml:"basicUser" json:"basicUser"`
	BasicPass string     `yaml:"basicPass" json:"basicPass"`
	JWTSecret string     `yaml:"jwtSecret" json:"jwtSecret"`
	OIDC      OIDCConfig `yaml:"oidc" json:"oidc"`
}

type OIDCConfig struct {
	IssuerURL          string   `yaml:"issuerUrl" json:"issuerUrl"`
	ClientID           string   `yaml:"clientId" json:"clientId"`
	GroupsClaim        string   `yaml:"groupsClaim" json:"groupsClaim"`
	UsernameClaim      string   `yaml:"usernameClaim" json:"usernameClaim"`
	AdminGroups        []string `yaml:"adminGroups" json:"adminGroups"`
	AllowBasicFallback bool     `yaml:"allowBasicFallback" json:"allowBasicFallback"`
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

type ClosureConfig struct {
	Policy               string `yaml:"policy" json:"policy"`
	AutoCloseConsecutive int    `yaml:"autoCloseConsecutive" json:"autoCloseConsecutive"`
}

type GitOpsConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Mode           string `yaml:"mode" json:"mode"`
	RepoURL        string `yaml:"repoUrl" json:"repoUrl"`
	BaseBranch     string `yaml:"baseBranch" json:"baseBranch"`
	BasePath       string `yaml:"basePath" json:"basePath"`
	CommitSignoff  bool   `yaml:"commitSignoff" json:"commitSignoff"`
	Provider       string `yaml:"provider" json:"provider"`
	TokenSecretRef string `yaml:"tokenSecretRef" json:"tokenSecretRef"`
	Token          string `yaml:"token" json:"-"`
}

type AWXConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	URL                string        `yaml:"url" json:"url"`
	Token              string        `yaml:"token" json:"-"`
	RequestTimeout     time.Duration `yaml:"requestTimeout" json:"requestTimeout"`
	DefaultInventoryID int64         `yaml:"defaultInventoryId" json:"defaultInventoryId"`
}

type AccessConfig struct {
	GroupRules []GroupRule `yaml:"groupRules" json:"groupRules"`
}

type GroupRule struct {
	Group      string   `yaml:"group" json:"group"`
	Services   []string `yaml:"services" json:"services"`
	Envs       []string `yaml:"envs" json:"envs"`
	Severities []string `yaml:"severities" json:"severities"`
	AlertNames []string `yaml:"alertNames" json:"alertNames"`
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
			Mode:      "basic",
			BasicUser: "admin",
			BasicPass: "change-me",
			OIDC: OIDCConfig{
				GroupsClaim:        "groups",
				UsernameClaim:      "preferred_username",
				AdminGroups:        []string{"runbook-admins"},
				AllowBasicFallback: true,
			},
		},
		Security: SecurityConfig{
			EgressAllowlist:      []string{"localhost", "127.0.0.1", "api", "postgres", "*.svc.cluster.local"},
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
		Closure: ClosureConfig{
			Policy:               "auto+manual",
			AutoCloseConsecutive: 2,
		},
		GitOps: GitOpsConfig{
			Enabled:    true,
			Mode:       "strict",
			BaseBranch: "main",
			BasePath:   ".",
			Provider:   "github",
		},
		AWX: AWXConfig{
			Enabled:        false,
			RequestTimeout: 20 * time.Second,
		},
		Access: AccessConfig{
			GroupRules: []GroupRule{},
		},
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
		"RH_AUTH_OIDC_ISSUER":    &cfg.Auth.OIDC.IssuerURL,
		"RH_AUTH_OIDC_CLIENT_ID": &cfg.Auth.OIDC.ClientID,
		"RH_AUTH_OIDC_GROUPS":    &cfg.Auth.OIDC.GroupsClaim,
		"RH_AUTH_OIDC_USERNAME":  &cfg.Auth.OIDC.UsernameClaim,
		"RH_RUNBOOK_MODE":        &cfg.Runbooks.Mode,
		"RH_RUNBOOK_PATH":        &cfg.Runbooks.Path,
		"RH_SETTINGS_CRYPTO_KEY": &cfg.Security.SettingsCryptoKeyB64,
		"RH_GITOPS_TOKEN":        &cfg.GitOps.Token,
		"RH_AWX_TOKEN":           &cfg.AWX.Token,
		"RH_AWX_URL":             &cfg.AWX.URL,
	}
	for env, ptr := range overrides {
		if value := os.Getenv(env); value != "" {
			*ptr = value
		}
	}
	if value := strings.TrimSpace(os.Getenv("RH_AUTH_OIDC_ADMIN_GROUPS")); value != "" {
		cfg.Auth.OIDC.AdminGroups = splitCSV(value)
	}
	if value := strings.TrimSpace(os.Getenv("RH_AUTH_OIDC_ALLOW_BASIC_FALLBACK")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			cfg.Auth.OIDC.AllowBasicFallback = parsed
		}
	}
}

func (c Config) Validate() error {
	if c.Database.Driver != "postgres" && c.Database.Driver != "sqlite" {
		return errors.New("database.driver must be postgres or sqlite")
	}
	mode := strings.ToLower(strings.TrimSpace(c.Auth.Mode))
	if mode != "basic" && mode != "jwt" && mode != "oidc" {
		return errors.New("auth.mode must be basic, jwt or oidc")
	}
	if mode == "basic" && (strings.TrimSpace(c.Auth.BasicUser) == "" || strings.TrimSpace(c.Auth.BasicPass) == "") {
		return errors.New("auth.basic user/password required in basic mode")
	}
	if mode == "jwt" && strings.TrimSpace(c.Auth.JWTSecret) == "" {
		return errors.New("auth.jwt secret required in jwt mode")
	}
	if mode == "oidc" {
		if strings.TrimSpace(c.Auth.OIDC.IssuerURL) == "" {
			return errors.New("auth.oidc.issuerUrl required in oidc mode")
		}
		if strings.TrimSpace(c.Auth.OIDC.ClientID) == "" {
			return errors.New("auth.oidc.clientId required in oidc mode")
		}
		if strings.TrimSpace(c.Auth.OIDC.GroupsClaim) == "" {
			return errors.New("auth.oidc.groupsClaim required in oidc mode")
		}
	}
	if c.Runbooks.Mode != "file" && c.Runbooks.Mode != "db" {
		return errors.New("runbooks.mode must be file or db")
	}
	if c.Limits.MaxSteps <= 0 {
		return errors.New("limits.maxSteps must be > 0")
	}
	gitopsMode := strings.ToLower(strings.TrimSpace(c.GitOps.Mode))
	if gitopsMode == "" {
		gitopsMode = "strict"
	}
	if gitopsMode != "strict" && gitopsMode != "hybrid" && gitopsMode != "disabled" {
		return errors.New("gitops.mode must be strict, hybrid or disabled")
	}
	provider := strings.ToLower(strings.TrimSpace(c.GitOps.Provider))
	if provider == "" {
		provider = "github"
	}
	if provider != "github" {
		return errors.New("gitops.provider must be github in MVP")
	}
	policy := strings.ToLower(strings.TrimSpace(c.Closure.Policy))
	if policy == "" {
		policy = "auto+manual"
	}
	if policy != "auto+manual" && policy != "manual" {
		return errors.New("closure.policy must be auto+manual or manual")
	}
	if c.Closure.AutoCloseConsecutive <= 0 {
		return errors.New("closure.autoCloseConsecutive must be > 0")
	}
	if c.AWX.RequestTimeout <= 0 {
		return errors.New("awx.requestTimeout must be > 0")
	}
	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
