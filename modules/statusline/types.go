package statusline

// StatusJSON is the input from Claude Code via stdin.
type StatusJSON struct {
	TranscriptPath string         `json:"transcript_path,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	Model          *ModelInfo     `json:"model,omitempty"`
	Workspace      *WorkspaceInfo `json:"workspace,omitempty"`
	ContextWindow  *ContextWindow `json:"context_window,omitempty"`
	Cost           *CostInfo      `json:"cost,omitempty"`
}

type ModelInfo struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type WorkspaceInfo struct {
	CurrentDir string `json:"current_dir,omitempty"`
	ProjectDir string `json:"project_dir,omitempty"`
}

type ContextWindow struct {
	ContextWindowSize    int           `json:"context_window_size,omitempty"`
	RemainingPercentage  *float64      `json:"remaining_percentage,omitempty"`
	CurrentUsage         *CurrentUsage `json:"current_usage,omitempty"`
}

type CurrentUsage struct {
	InputTokens              int `json:"input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

type CostInfo struct {
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
}

// Config for statusline module.
type Config struct {
	Format  string        `yaml:"format" mapstructure:"format"`
	Context ContextConfig `yaml:"context" mapstructure:"context"`
	Model   ModelConfig   `yaml:"model" mapstructure:"model"`
	Cost    CostConfig    `yaml:"cost" mapstructure:"cost"`
	Custom  CustomConfig  `yaml:"custom" mapstructure:"custom"`
}

type ContextConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type ModelConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type CostConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type CustomConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Command string `yaml:"command" mapstructure:"command"`
	Timeout int    `yaml:"timeout" mapstructure:"timeout"` // ms
}

// DefaultConfig returns config with all disabled (BASIS).
func DefaultConfig() Config {
	return Config{
		Format:  "$context │ $model │ $cost │ $custom",
		Context: ContextConfig{Enabled: false},
		Model:   ModelConfig{Enabled: false},
		Cost:    CostConfig{Enabled: false},
		Custom:  CustomConfig{Enabled: false, Timeout: 500},
	}
}

// MaxContextTokens default context window.
const MaxContextTokens = 200000
