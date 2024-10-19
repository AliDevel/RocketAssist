package openai

import (
	"RocketAssist/config"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//var ErrorContextLengthExceeded = errors.New("context length exceeded")

type ErrorContextLengthExceeded struct {
	message string
}

func (e *ErrorContextLengthExceeded) Error() string {
	return e.message
}

func NewErrorContextLengthExceeded(msg string) error {
	return &ErrorContextLengthExceeded{msg}
}

func (e *ErrorContextLengthExceeded) Is(tgt error) bool {
	_, ok := tgt.(*ErrorContextLengthExceeded)
	if !ok {
		return false
	}
	return true
}

type ThreadRequest struct {
	AssistantID  string `json:"assistant_id"` // ID of the assistant
	Instructions string `json:"instructions"` // Instructions for the thread
}

type ThreadResponse struct {
	ThreadID string `json:"id"` // ID of the created thread
	Status   string `json:"status"`
	Error    struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Struct to capture the assistant retrieval response
type AssistantDetails struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Model  string `json:"model"`
}
type OpenAI struct {
	HostName           string
	CompletionEndpoint string
	ModerationEndpoint string
	AssistanceEndpoint string
	ApiToken           string
	PrePrompt          string
	Model              string
	InputModeration    bool
	OutputModeration   bool
	SendUserId         bool
	ModelParams        config.ModelParams
	AssistantID        string
}

type HTTPError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

func NewFromConfig(config *config.Config) *OpenAI {
	oa := OpenAI{
		HostName:           config.OpenAI.HostName,
		ApiToken:           config.OpenAI.ApiToken,
		PrePrompt:          strings.TrimSpace(config.OpenAI.PrePrompt),
		Model:              config.OpenAI.Model,
		ModerationEndpoint: config.OpenAI.ModerationEndpoint,
		CompletionEndpoint: config.OpenAI.CompletionEndpoint,
		AssistanceEndpoint: config.OpenAI.AssistanceEndpoint,
		InputModeration:    config.OpenAI.InputModeration,
		OutputModeration:   config.OpenAI.OutputModeration,
		SendUserId:         config.OpenAI.SendUserId,

		ModelParams: config.OpenAI.ModelParams,
		AssistantID: config.OpenAI.AssistantID,
	}
	return &oa
}

func (o *OpenAI) CompletionURL() (string, error) {
	url, err := url.JoinPath("https://", o.HostName, o.CompletionEndpoint)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (o *OpenAI) ModerationURL() (string, error) {
	url, err := url.JoinPath("https://", o.HostName, o.ModerationEndpoint)
	if err != nil {
		return "", err
	}
	return url, nil
}
func (o *OpenAI) ThreadURL() (string, error) {
	url, err := url.JoinPath("https://", o.HostName, "/v1/threads")
	if err != nil {
		return "", fmt.Errorf("failed to build Thread URL: %w", err)
	}
	return url, nil
}
func (o *OpenAI) Completion(cReq *CompletionRequest) (*CompletionResponse, error) {
	var cResp CompletionResponse
	url, err := o.CompletionURL()
	if err != nil {
		return nil, fmt.Errorf("cannot assemble endpoint url: %w", err)
	}
	err = o.request(url, cReq, &cResp)
	if cResp.Error.Code == "context_length_exceeded" {
		return nil, NewErrorContextLengthExceeded(cResp.Error.Message)
	} else if cResp.Error.Message != "" {
		return nil, fmt.Errorf("%w: %s ", err, cResp.Error.Message)
	}
	// All other errors
	if err != nil {
		return &cResp, fmt.Errorf("an error occured while performing the request: %w", err)
	}

	return &cResp, nil
}

func (o *OpenAI) Moderation(mReq *ModerationRequest) (*ModerationResponse, error) {
	var mResp ModerationResponse
	url, err := o.ModerationURL()
	if err != nil {
		return nil, fmt.Errorf("cannot assemble endpoint url: %w", err)
	}
	err = o.request(url, mReq, &mResp)
	if mResp.Error.Message != "" {
		return nil, fmt.Errorf("%w: %s ", err, mResp.Error.Message)
	}
	if err != nil {
		return &mResp, fmt.Errorf("an error occured during performing the request: %w", err)
	}

	if len(mResp.ID) == 0 {
		// It is dangerous to proceed if the moderation response does not look like to be legit.
		return nil, fmt.Errorf("empty moderation response")
	}

	return &mResp, nil
}

func (o *OpenAI) request(url string, request interface{}, oaResponse interface{}) error {
	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("cannot marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("cannot create new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.ApiToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot perform request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return parseError(resp, oaResponse)
	}

	err = json.NewDecoder(resp.Body).Decode(oaResponse)
	if err != nil {
		return fmt.Errorf("cannot parse response body: %w", err)
	}

	return nil
}

// Reusable request function for POST and GET
func (o *OpenAI) requestAPI(method, url string, request interface{}, oaResponse interface{}) error {
	var req *http.Request
	var err error

	if request != nil {
		data, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("cannot marshal request body: %w", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(data))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return fmt.Errorf("cannot create new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.ApiToken))
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(oaResponse)
	if err != nil {
		return fmt.Errorf("cannot parse response body: %w", err)
	}

	return nil
}

func (o *OpenAI) NewCompletionRequest(messages []Message, user string) *CompletionRequest {
	r := &CompletionRequest{
		Model:            o.Model,
		Messages:         messages,
		Temperature:      o.ModelParams.Temperature,
		TopP:             o.ModelParams.TopP,
		MaxTokens:        o.ModelParams.MaxTokens,
		PresencePenalty:  o.ModelParams.PresencePenalty,
		FrequencyPenalty: o.ModelParams.FrequencyPenalty,
	}

	if len(user) > 0 {
		r.User = &user
	}

	return r
}

func parseError(resp *http.Response, oaResponse interface{}) error {
	if resp.Body == nil {
		return fmt.Errorf("HTTP Error: %d", resp.StatusCode)
	}

	err := json.NewDecoder(resp.Body).Decode(oaResponse)
	if err != nil {
		return fmt.Errorf("HTTP Error: %d but response body is not available: %w", resp.StatusCode, err)
	}

	return fmt.Errorf("Non-OK status code: %d", resp.StatusCode)
}

func (o *OpenAI) AssistanceURL() (string, error) {
	if o.AssistantID != "" {
		// URL for accessing a specific assistant by its ID
		url, err := url.JoinPath("https://", o.HostName, o.AssistanceEndpoint, o.AssistantID)
		if err != nil {
			return "", fmt.Errorf("failed to build Assistance URL with ID: %w", err)
		}
		return url, nil
	}

	// URL for creating a new assistant (no assistant ID needed)
	url, err := url.JoinPath("https://", o.HostName, o.AssistanceEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to build Assistance URL: %w", err)
	}
	return url, nil
}

// Function to retrieve an assistant by ID
func (o *OpenAI) GetAssistantByID(assistantID string) (*AssistantDetails, error) {
	url, err := o.AssistanceURL()
	if err != nil {
		return nil, fmt.Errorf("cannot assemble endpoint url: %w", err)
	}

	// Initialize the response object
	var response AssistantDetails

	// Make the GET request
	err = o.requestAPI("GET", url, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("error retrieving assistant: %w", err)
	}

	return &response, nil
}
func (o *OpenAI) CreateThread() (*ThreadResponse, error) {
	// Define the URL for the threads endpoint
	url := "https://api.openai.com/v1/threads"

	// Initialize the response
	var tResp ThreadResponse

	// Send the POST request with an empty body
	err := o.requestAPI("POST", url, nil, &tResp) // `nil` is passed for an empty body
	if err != nil {
		return nil, fmt.Errorf("an error occurred while creating the thread: %w", err)
	}

	// Check for any API-specific errors
	if tResp.Error.Message != "" {
		return nil, fmt.Errorf("%s: %s", tResp.Error.Code, tResp.Error.Message)
	}

	return &tResp, nil
}
func (o *OpenAI) AddMessageToThread(threadID string, msg Message) (*MessageResponse, error) {
	// Define the URL for adding a message to the thread
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadID)

	// Initialize the response
	var mResp MessageResponse

	// Send the POST request with the prepared message
	err := o.requestAPI("POST", url, msg, &mResp)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while adding the message: %w", err)
	}

	// Check for any API-specific errors
	if mResp.Error.Message != "" {
		return nil, fmt.Errorf("%s: %s", mResp.Error.Code, mResp.Error.Message)
	}

	return &mResp, nil
}

func (o *OpenAI) CreateRun(threadID string) (*RunResponse, error) {
	// Define the URL for creating a run within the thread
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs", threadID)

	// Use o.AssistantID and o.PrePrompt in the request
	rReq := &RunRequest{
		AssistantID: o.AssistantID, // Use o.AssistantID here
		//Instructions: o.PrePrompt,   // Use o.PrePrompt here
	}

	// Initialize the response
	var rResp RunResponse

	// Send the POST request with the run details
	err := o.requestAPI("POST", url, rReq, &rResp)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while creating the run: %w", err)
	}

	// Check for any API-specific errors
	if rResp.Error.Message != "" {
		return nil, fmt.Errorf("%s: %s", rResp.Error.Code, rResp.Error.Message)
	}

	return &rResp, nil
}
func (o *OpenAI) GetMessages(threadID string) (*MessageListResponse, error) {
	// Define the URL for fetching messages from the thread
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadID)

	// Initialize the response
	var mResp MessageListResponse

	// Send the GET request (note that request body is nil for GET)
	err := o.requestAPI("GET", url, nil, &mResp)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while fetching the messages: %w", err)
	}

	return &mResp, nil
}
func (o *OpenAI) RetrieveRun(threadID, runID string) (*RunResponse, error) {
	// Define the URL for retrieving the run status
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs/%s", threadID, runID)

	// Initialize the response
	var rResp RunResponse

	// Send the GET request to retrieve the run status
	err := o.requestAPI("GET", url, nil, &rResp)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while retrieving the run: %w", err)
	}

	// Check for any API-specific errors
	if rResp.Error.Message != "" {
		return nil, fmt.Errorf("%s: %s", rResp.Error.Code, rResp.Error.Message)
	}

	return &rResp, nil
}
func (o *OpenAI) WaitForRunCompletion(threadID, runID string) (*RunResponse, error) {
	done := false

	for !done {
		// Retrieve the run status
		resp, err := o.RetrieveRun(threadID, runID)
		if err != nil {
			return nil, fmt.Errorf("error getting run: %v", err)
		}

		// Handle different statuses
		switch resp.Status {
		case "in_progress", "queued":
			// Wait for a few seconds before checking again
			time.Sleep(5 * time.Second)

		case "completed":
			done = true
			return resp, nil // Return the completed run with messages

		case "failed":
			return nil, fmt.Errorf("run failed: %v", resp.Error.Message)

		case "cancelled", "cancelling", "expired":
			return nil, fmt.Errorf("run was cancelled or expired")

		case "requires_action":
			return nil, fmt.Errorf("run requires action")

		default:
			return nil, fmt.Errorf("unexpected run status: %s", resp.Status)
		}
	}

	return nil, fmt.Errorf("run did not complete")
}
