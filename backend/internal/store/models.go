package store

import (
	"time"

	"gorm.io/gorm"
)

type Signal struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	IncidentID   uint   `gorm:"index"`
	Fingerprint  string `gorm:"index"`
	AlertName    string `gorm:"index"`
	Status       string
	Labels       []byte `gorm:"type:json"`
	Annotations  []byte `gorm:"type:json"`
	StartsAt     time.Time
	EndsAt       *time.Time
	GeneratorURL string
}

type Incident struct {
	ID                 uint `gorm:"primaryKey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Fingerprint        string `gorm:"uniqueIndex"`
	RouteKey           string `gorm:"index"`
	AlertName          string `gorm:"index"`
	Service            string `gorm:"index"`
	Env                string `gorm:"index"`
	Severity           string `gorm:"index"`
	Status             string `gorm:"index"`
	RunbookName        string
	Brief              string
	OpenedAt           time.Time
	ClosedAt           *time.Time
	LastSignalAt       time.Time
	Labels             []byte `gorm:"type:json"`
	ClosureState       string `gorm:"index"`
	ClosureReason      string
	ClosureCriteria    []byte `gorm:"type:json"`
	ResolvedAt         *time.Time
	ClosureReadyStreak int
}

type StepRun struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	IncidentID uint      `gorm:"index"`
	StepName   string    `gorm:"index"`
	Tool       string    `gorm:"index"`
	Status     string    `gorm:"index"`
	Output     string    `gorm:"type:text"`
	Error      string    `gorm:"type:text"`
	StartedAt  time.Time `gorm:"index"`
	FinishedAt time.Time
}

type Runbook struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"uniqueIndex"`
	Enabled   bool   `gorm:"index"`
	Spec      []byte `gorm:"type:json"`
}

type RunbookExecution struct {
	ID                 uint `gorm:"primaryKey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	IncidentID         uint   `gorm:"index"`
	RunbookName        string `gorm:"index"`
	RunbookVersionHash string
	Status             string `gorm:"index"`
	StartedAt          time.Time
	FinishedAt         *time.Time
	Trigger            string
	Active             bool `gorm:"index"`
}

type RunbookExecutionStep struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExecutionID    uint `gorm:"index:idx_exec_step_order,unique"`
	StepName       string
	StepOrder      int    `gorm:"index:idx_exec_step_order,unique"`
	Kind           string `gorm:"index"`
	Status         string `gorm:"index"`
	Output         string `gorm:"type:text"`
	Error          string `gorm:"type:text"`
	Recommendation string `gorm:"type:text"`
	Required       bool
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

type SourceConfig struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"uniqueIndex"`
	Type      string
	Enabled   bool
	Config    []byte `gorm:"type:json"`
}

type DestinationConfig struct {
	ID              string `gorm:"primaryKey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Name            string
	DestinationType string `gorm:"index"`
	Enabled         bool   `gorm:"index"`
	Config          []byte `gorm:"type:json"`
}

type RoutingRule struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Name         string `gorm:"uniqueIndex"`
	RouteKey     string `gorm:"index"`
	Enabled      bool   `gorm:"index"`
	Priority     int
	MatchLabels  []byte `gorm:"type:json"`
	Destinations []byte `gorm:"type:json"`
}

type SettingsOverride struct {
	Key       string `gorm:"primaryKey"`
	Value     string `gorm:"type:text"`
	Encrypted bool
	UpdatedAt time.Time
}

type ApprovalRequest struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IncidentID  uint `gorm:"index"`
	ExecutionID uint `gorm:"index"`
	StepOrder   int
	Action      string
	Status      string
	Reason      string
}

type IncidentEvent struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	IncidentID uint   `gorm:"index"`
	EventType  string `gorm:"index"`
	Message    string
	Payload    []byte `gorm:"type:json"`
}

type GitOpsChange struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ChangeType  string `gorm:"index"`
	Title       string
	Status      string `gorm:"index"`
	Branch      string
	PRURL       string
	PRNumber    int64
	Desired     []byte `gorm:"type:json"`
	Applied     []byte `gorm:"type:json"`
	DriftStatus string `gorm:"index"`
	Error       string `gorm:"type:text"`
}

type DeliveryState struct {
	ID                uint `gorm:"primaryKey"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	IncidentID        uint   `gorm:"index:idx_incident_destination,unique"`
	DestinationID     string `gorm:"index:idx_incident_destination,unique"`
	LastHash          string `gorm:"index"`
	LastExecutionHash string
	LastSentAt        time.Time `gorm:"index"`
}

func (Incident) TableName() string             { return "incidents" }
func (Signal) TableName() string               { return "signals" }
func (StepRun) TableName() string              { return "step_runs" }
func (Runbook) TableName() string              { return "runbooks" }
func (RunbookExecution) TableName() string     { return "runbook_executions" }
func (RunbookExecutionStep) TableName() string { return "runbook_execution_steps" }
func (SourceConfig) TableName() string         { return "source_configs" }
func (DestinationConfig) TableName() string    { return "destination_configs" }
func (RoutingRule) TableName() string          { return "routing_rules" }
func (SettingsOverride) TableName() string     { return "settings_overrides" }
func (ApprovalRequest) TableName() string      { return "approval_requests" }
func (IncidentEvent) TableName() string        { return "incident_events" }
func (GitOpsChange) TableName() string         { return "gitops_changes" }
func (DeliveryState) TableName() string        { return "delivery_states" }

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Incident{},
		&Signal{},
		&StepRun{},
		&Runbook{},
		&RunbookExecution{},
		&RunbookExecutionStep{},
		&SourceConfig{},
		&DestinationConfig{},
		&RoutingRule{},
		&SettingsOverride{},
		&ApprovalRequest{},
		&IncidentEvent{},
		&GitOpsChange{},
		&DeliveryState{},
	)
}
