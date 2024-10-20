package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel   string `yaml:"LogLevel"`
	RocketChat struct {
		UserId    string
		User      string `yaml:"User"`
		Password  string `yaml:"Password"`
		AuthToken string `yaml:"Authtoken"`
		HostName  string `yaml:"HostName"`
		SSL       bool   `yaml:"SSL"`
		Port      uint16 `yaml:"Port"`
	} `yaml:"RocketChat"`
	OpenAI struct {
		HostName           string         `yaml:"HostName"`
		ApiToken           string         `yaml:"ApiToken"`
		CompletionEndpoint string         `yaml:"CompletionEndpoint"`
		ModerationEndpoint string         `yaml:"ModerationEndpoint"`
		AssistanceEndpoint string         `yaml:"AssistanceEndpoint"`
		Model              string         `yaml:"Model"`
		HistorySize        int            `yaml:"HistorySize"`
		HistoryMaxLength   int            `yaml:"HistoryMaxLength"`
		MessageRetention   *time.Duration `yaml:"MessageRetention,omitempty"`
		PrePrompt          string         `yaml:"PrePrompt"`
		InputModeration    bool           `yaml:"InputModeration"`
		OutputModeration   bool           `yaml:"OutputModeration"`
		SendUserId         bool           `yaml:"SendUserId"`
		ModelParams        ModelParams    `yaml:"ModelParams,omitempty"`
		AssistantID        string         `yaml:"AssistantID"`
	} `yaml:"OpenAI"`
}

type ModelParams struct {
	Temperature      *float64 `yaml:"Temperature,omitempty"`
	TopP             *float64 `yaml:"TopP,omitempty"`
	FrequencyPenalty *float64 `yaml:"FrequencyPenalty,omitempty"`
	PresencePenalty  *float64 `yaml:"PresencePenalty,omitempty"`
	MaxTokens        *int     `yaml:"MaxTokens,omitempty"`
}

func NewConfig(path string) (*Config, error) {
	//log.WithField("message", "Method").Debug("Config")
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read configfile: %w", err)
	}

	// Parse YAML file
	var config Config

	// Default values
	config.RocketChat.SSL = true

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("cannot parse configfile %w", err)
	}

	return &config, nil
}
