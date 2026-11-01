package council

// ReviewResult represents the output from a single critic
type ReviewResult struct {
	ReviewerType string      `json:"reviewer_type"`
	Model        string      `json:"model"`
	RawOutput    string      `json:"raw_output"`
	Issues       []Issue     `json:"issues"`
	Error        error       `json:"-"`
	Warning      string      `json:"warning,omitempty"`
}

// Issue represents a specific problem identified by a critic
type Issue struct {
	Severity    string `json:"severity"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
	Fix         string `json:"fix,omitempty"`
}

// SenateReport represents the final synthesized report
type SenateReport struct {
	CriticalIssues []Issue `json:"critical_issues"`
	Suggestions    []Issue `json:"suggestions"`
	Nitpicks       []Issue `json:"nitpicks"`
	Summary        string  `json:"summary"`
}

// ReviewConfig holds configuration for the review process
type ReviewConfig struct {
	Reviewers []ReviewerConfig
	Chairman  ReviewerConfig
	Code      string
	Context   string // Additional context (e.g., PR description)
}

// ReviewerConfig holds configuration for a specific reviewer
type ReviewerConfig struct {
	Name    string
	Role    string // security, logic, style, chairman
	Model   string
	APIKey  string
	BaseURL string // Optional: for Ollama or other OpenAI-compatible APIs
}
