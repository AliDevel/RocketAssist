package main

import (
	"fmt"
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
	// Step 1: Check if the AssistantID is available
	if oa.AssistantID != "" {
		return useAssistantAPI(rocketmsg, oa, hist) // Use Assistant API if AssistantID is present
	} else {
		return useCompletionAPI(rocketmsg, oa, hist) // Use Completion API if no AssistantID
	}
}

/*
func OpenAIResponse(rocketmsg rocket.Message, oa *openai.OpenAI, hist *History) error {
	//assistanceId := oa.AssistantID
	//assistant, err := oa.GetAssistantByID(assistanceId)
	//if err != nil {
	//	log.Fatalf("Error retrieving assistant: %v", err)
	//}
	place := rocketmsg.RoomName
	var threadID string
	//log.WithField("message", "Assistent model").Debug(assistant.Model)
	if existingThreadID, found := roomToThreadMap[place]; found {
		// If thread exists, retrieve the thread ID
		threadID = existingThreadID
		log.WithField("message", "Reusing existing thread").Debug(threadID)
	} else {
		// If no thread exists for this room, create a new one
		thread, err := oa.CreateThread()
		if err != nil {
			log.Fatalf("Error creating Thread: %v", err)
		}
		threadID = thread.ThreadID
		log.WithField("message", "New Thread ID").Debug(threadID)

		// Store the new thread ID in the map for future use
		roomToThreadMap[place] = threadID
		log.WithField("room", place).Debug("Storing new thread ID in map")
	}
	log.WithField("message", "Thread ID").Debug(threadID)

	msg := openai.Message{
		Role:    "user",
		Content: rocketmsg.GetNotAddressedText(),
	}

	threadmessage, err := oa.AddMessageToThread(threadID, msg)
	if err != nil {
		log.Fatalf("Error Adding Message to Thread: %v", err)
	}
	log.WithField("message", "Thread Message").Debug(threadmessage.ID)
	// Create the run in the thread using the PrePrompt and AssistantID from the struct
	runResp, err := oa.CreateRun(threadID)
	if err != nil {
		log.Fatalf("Error Creating RunID: %v", err)
	}
	log.WithField("message", "Create Run").Debug(runResp.RunID)

	runStatus, err := oa.WaitForRunCompletion(threadID, runResp.RunID)
	log.WithField("message", "Run status").Debug(runStatus.Status)
	messagesResp, err := oa.GetMessages(threadID)
	log.WithField("message", "Length of messages").Debug(messagesResp)
	rocketmsg.SetIsTyping(true)
	defer func() {
		rocketmsg.SetIsTyping(false)
	}()

	var response string
	response += messagesResp.Messages[0].Content[0].Text.Value
	log.WithField("Received", "Messages").Debug(messagesResp.Messages[0].Content[0].Text.Value)
	_, err = rocketmsg.Reply(fmt.Sprintf("@%s %s", rocketmsg.UserName, response))
	if err != nil {
		return fmt.Errorf("cannot send reply to rocketchat: %w", err)
	}

	/*
		if oa.InputModeration {
			// Send the input to the OpenAI moderation endpoint, and if it is flagged, return an error instead of sending anything to the completion endpoint.
			mresp, err := oa.Moderation(&openai.ModerationRequest{
				Input: rocketmsg.GetNotAddressedText(),
			})
			if err != nil {
				return fmt.Errorf("cannot perform perliminary request to the moderation endpoint: %w", err)
			}

			log.WithField("moderationResponse", mresp).Debug("Preliminary (input) moderation response.")

			if mresp.IsFlagged() {
				// @todo configurable message?
				_, err = rocketmsg.Reply(fmt.Sprintf("@%s :triangular_flag_on_post: Our bot uses OpenAI's moderation system, which flagged your message as inappropriate. Please try rephrasing your message to avoid any offensive or inappropriate content. REASON: %s :triangular_flag_on_post:",
					rocketmsg.UserName, mresp.FlaggedReason()))
				if err != nil {
					return fmt.Errorf("cannot send reply to rocketchat: %w", err)
				}
				return nil
			}
		}

		var systemMessage = openai.Message{
			Role:    "system",
			Content: oa.PrePrompt,
		}

		// Prepend the preprompt
		var messages []openai.Message
		if len(oa.PrePrompt) > 0 {
			messages = append(messages, systemMessage)
		}
		messages = append(messages, hist.AsOpenAIMessages(place)...)

		messages = append(messages, msg)

		OAUserid := "" // Userid to send OpenAI. If empty, the no UserId is sent.
		if oa.SendUserId {
			OAUserid = rocketmsg.UserId
		}
		cresp, err := oa.Completion(oa.NewCompletionRequest(messages, OAUserid))
		if err != nil {
			if errors.Is(err, &openai.ErrorContextLengthExceeded{}) {
				// If the reason for the error is context_length_exceeded, we clear history, so it does not happen on the next comment.
				hist.Clear(place)
				log.Debug("To prevent context_length_exceeded error to happen repeatedly, the history has been cleared.")
			}
			return fmt.Errorf("cannot perform completion request: %w", err)
		}

		if len(cresp.Choices) == 0 {
			return fmt.Errorf("no choices returned")
		}

		log.WithField("completionResponse", cresp).Trace("Completion response.")

		var mresp *openai.ModerationResponse
		if oa.OutputModeration {
			mresp, err = oa.Moderation(&openai.ModerationRequest{
				Input: cresp.Choices[0].Message.Content,
			})
			if err != nil {
				return fmt.Errorf("cannot perform follow-up request to the moderation endpoint (output check): %w", err)
			}

			if mresp.IsFlagged() {
				// @todo better explanation that it is the output that got flagged.
				response = fmt.Sprintf(":triangular_flag_on_post: (output flagged: %s) :triangular_flag_on_post:", mresp.FlaggedReason())
			}

			log.WithField("moderationResponse", mresp).Trace("Follow-up (output) moderation response.")

		}

		response += cresp.Choices[0].Message.Content

		// @todo further calls if finishReason indicates that the response is not completed.
		_, err = rocketmsg.Reply(fmt.Sprintf("@%s %s", rocketmsg.UserName, response))
		if err != nil {
			return fmt.Errorf("cannot send reply to rocketchat: %w", err)
		}

		if mresp == nil || !mresp.IsFlagged() {
			hist.Add(place, msg)
			hist.Add(place, openai.Message{
				Role:    "assistant",
				Content: cresp.Choices[0].Message.Content,
			})
		}

	return nil
}
*/
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

// Helper to send response back to the user
func sendResponseToUser(rocketmsg rocket.Message, responseText string) error {
	rocketmsg.SetIsTyping(true)
	defer rocketmsg.SetIsTyping(false)

	_, err := rocketmsg.Reply(fmt.Sprintf("@%s %s", rocketmsg.UserName, responseText))
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
