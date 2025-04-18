package types

// --- Request Parameter Structs ---

// SamplingContent represents the content of a message (text, image, or audio).
type SamplingContent struct {
	Type     string `json:"type"` // "text", "image", "audio"
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`     // Base64 encoded data for image/audio
	MimeType string `json:"mimeType,omitempty"` // e.g., "image/jpeg", "audio/wav"
}

// SamplingMessage represents a single message in the sampling prompt sequence.
type SamplingMessage struct {
	Role    string          `json:"role"` // "user", "assistant" typically
	Content SamplingContent `json:"content"`
}

// ModelHint provides advisory model names or families.
type ModelHint struct {
	Name string `json:"name"`
}

// ModelPreferences guides client-side model selection.
type ModelPreferences struct {
	Hints                []ModelHint `json:"hints,omitempty"`
	CostPriority         float64     `json:"costPriority"`         // 0-1
	SpeedPriority        float64     `json:"speedPriority"`        // 0-1
	IntelligencePriority float64     `json:"intelligencePriority"` // 0-1
}

// SamplingCreateMessageParams defines the parameters for the sampling/createMessage request.
type SamplingCreateMessageParams struct {
	Messages         []SamplingMessage `json:"messages"`
	ModelPreferences ModelPreferences  `json:"modelPreferences,omitempty"`
	SystemPrompt     string            `json:"systemPrompt,omitempty"`
	MaxTokens        *int              `json:"maxTokens,omitempty"` // Use pointer for optionality
}

// --- Response Result Structs ---

// SamplingCreateMessageResult defines the successful result structure.
type SamplingCreateMessageResult struct {
	Role       string          `json:"role"` // Typically "assistant"
	Content    SamplingContent `json:"content"`
	Model      string          `json:"model"`      // Actual model used by client
	StopReason string          `json:"stopReason"` // e.g., "endTurn", "maxTokens"
}
