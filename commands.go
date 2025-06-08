package main

import (
	"fmt"
	"regexp"
	"time"

	"RocketAssist/openai"
	"RocketAssist/rocket"

	log "github.com/sirupsen/logrus"
)

func DemoResponse(msg rocket.Message) {
	reply, err := msg.Reply(fmt.Sprintf("@%s %s %d", msg.UserName, msg.GetNotAddressedText(), 0))

	// If no error replying, take the reply and edit it to count to 10 asynchronously
	if err == nil {
		go func() {
			msg.SetIsTyping(true)
			for i := 1; i <= 10; i++ {
				time.Sleep(time.Second)
				reply.EditText(fmt.Sprintf("@%s %s %d", msg.UserName, msg.GetNotAddressedText(), i))
			}
			msg.SetIsTyping(false)
		}()
	}

	// React to the initial message
	msg.React(":grinning:")
}

var roomToThreadMap = make(map[string]string)

func OpenAIResponse(rocketmsg rocket.Message, oa *openai.OpenAI, hist *History) error {
	if oa.AssistantID != "" {
		return useAssistantAPI(rocketmsg, oa, hist)
	}
	return useCompletionAPI(rocketmsg, oa, hist)
}

// Function for handling Assistant API
func useAssistantAPI(rocketmsg rocket.Message, oa *openai.OpenAI, hist *History) error {
	place := rocketmsg.RoomName
	threadID := getOrCreateThread(place, oa) // Get or create thread based on room name

	msg := openai.Message{
		Role:    "user",
		Content: rocketmsg.GetNotAddressedText(),
	}

	_, err := oa.AddMessageToThread(threadID, msg)
	if err != nil {
		return fmt.Errorf("error adding message to thread: %w", err)
	}

	// Create the run using the assistant's thread
	runResp, err := oa.CreateRun(threadID)
	if err != nil {
		return fmt.Errorf("error creating run: %w", err)
	}

	runStatus, err := oa.WaitForRunCompletion(threadID, runResp.RunID)
	log.WithField("message", "Length of messages").Debug(runStatus.RunID)
	if err != nil {
		return fmt.Errorf("error waiting for run completion: %w", err)
	}

	// Get the response message and reply
	messagesResp, err := oa.GetMessages(threadID)
	if err != nil {
		return fmt.Errorf("error getting messages: %w", err)
	}
	return sendResponseToUser(rocketmsg, messagesResp.Messages[0].Content[0].Text.Value)
}

// Function for handling Completion API
func useCompletionAPI(rocketmsg rocket.Message, oa *openai.OpenAI, hist *History) error {
	// Perform any necessary input moderation before proceeding (if required)
	msg := openai.Message{
		Role:    "user",
		Content: rocketmsg.GetNotAddressedText(),
	}

	// Prepare messages, including any system messages or history if needed
	var messages []openai.Message
	if len(oa.PrePrompt) > 0 {
		systemMessage := openai.Message{
			Role:    "system",
			Content: oa.PrePrompt,
		}
		messages = append(messages, systemMessage)
	}
	messages = append(messages, hist.AsOpenAIMessages(rocketmsg.RoomName)...)
	messages = append(messages, msg)

	// Call the Completion API
	OAUserid := ""
	if oa.SendUserId {
		OAUserid = rocketmsg.UserId
	}
	cresp, err := oa.Completion(oa.NewCompletionRequest(messages, OAUserid))
	if err != nil {
		return fmt.Errorf("error completing request: %w", err)
	}

	if len(cresp.Choices) == 0 {
		return fmt.Errorf("no choices returned")
	}

	// Handle moderation and response sending
	return sendResponseToUser(rocketmsg, cresp.Choices[0].Message.Content)
}

// Function to remove references like
func removeReferences(input string) string {
	// Define a regex pattern to match 【...】
	re := regexp.MustCompile(`【.*?†.*?】`)
	// Replace all occurrences of the pattern with an empty string
	return re.ReplaceAllString(input, "")
}

// Helper to send response back to the user
func sendResponseToUser(rocketmsg rocket.Message, responseText string) error {
	rocketmsg.SetIsTyping(true)
	defer rocketmsg.SetIsTyping(false)
	cleanedText := removeReferences(responseText)
	_, err := rocketmsg.Reply(fmt.Sprintf("@%s %s", rocketmsg.UserName, cleanedText))
	if err != nil {
		return fmt.Errorf("cannot send reply to rocketchat: %w", err)
	}
	return nil
}

// Helper function to get or create a thread
func getOrCreateThread(place string, oa *openai.OpenAI) string {
	if existingThreadID, found := roomToThreadMap[place]; found {
		log.WithField("message", "Reusing existing thread").Debug(existingThreadID)
		return existingThreadID
	}

	thread, err := oa.CreateThread()
	if err != nil {
		log.Fatalf("error creating thread: %v", err)
	}

	roomToThreadMap[place] = thread.ThreadID
	log.WithField("room", place).Debug("Storing new thread ID in map")
	return thread.ThreadID
}
