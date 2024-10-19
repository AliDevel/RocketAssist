package openai

// https://openai.com/blog/introducing-chatgpt-and-whisper-apis

type CompletionResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int       `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   HTTPError `json:"error"`
}

type CompletionRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	Temperature      *float64  `json:"temperature,omitempty"`
	MaxTokens        *int      `json:"max_tokens,omitempty"`
	TopP             *float64  `json:"top_p,omitempty"`
	PresencePenalty  *float64  `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64  `json:"frequency_penalty,omitempty"`
	User             *string   `json:"user,omitempty"`
}
type MessageResponse struct {
	ID     string `json:"id"`     // ID of the created message
	Status string `json:"status"` // Status of the operation (e.g., "success")
	Error  struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type MessageContent struct {
	Type string `json:"type"`
	Text struct {
		Value       string     `json:"value"`
		Annotations []struct{} `json:"annotations"` // Assuming annotations are present but can be ignored for now
	} `json:"text"`
}
type MessageRun struct {
	ID          string           `json:"id"`
	Role        string           `json:"role"`
	AssistantID *string          `json:"assistant_id,omitempty"` // AssistantID can be null
	Content     []MessageContent `json:"content"`                // Content is an array
	CreatedAt   int64            `json:"created_at"`
	ThreadID    string           `json:"thread_id"`
	RunID       *string          `json:"run_id,omitempty"` // RunID can be null
}
type MessageListResponse struct {
	Messages []MessageRun `json:"data"`

	Object  string  `json:"object"`
	FirstID *string `json:"first_id"`
	LastID  *string `json:"last_id"`
	HasMore bool    `json:"has_more"`
}
type Choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
type RunRequest struct {
	AssistantID  string `json:"assistant_id"` // ID of the assistant
	Instructions string `json:"instructions"` // Instructions for the run
}
type RunResponse struct {
	RunID  string `json:"id"`     // ID of the created run
	Status string `json:"status"` // Status of the operation (e.g., "success")
	Error  struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}
