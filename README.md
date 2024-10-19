# RocketAssist

RocketAssist is an intelligent chatbot designed for seamless integration with Rocket.Chat, leveraging the power of OpenAI's APIs to deliver dynamic and meaningful conversations. Written in Go, this chatbot can be customized to fit a wide range of use cases by offering flexibility in prompts and responses. Whether you need a conversational assistant or an automated responder, RocketAssist is equipped to meet your needs.

Additionally, RocketAssist offers advanced functionality when using OpenAI’s Assistance API, allowing it to read and process data from files. This feature provides users with enhanced flexibility, enabling the bot to access stored knowledge and respond based on information in external files. This ensures that the bot answers precisely according to your business's needs and requirements.
## Features
Rocket.Chat Integration: Seamlessly integrates with Rocket.Chat to respond to user inputs in real-time.
OpenAI-powered: Utilizes OpenAI’s advanced language models for natural, context-aware conversations.
Customizable: Easily adapt the bot's behavior with custom prompts and configurations.
Flexible API Usage: Supports both OpenAI’s Completion API and Assistance API, allowing for rich conversational contexts.
## Open AI 
1. Go to open AI and Create API key: https://platform.openai.com/api-keys
2. If you want to fine tune models then please go and create fine tuned model then provide in config.yaml model id: https://platform.openai.com/finetune
3. if you want to use knowledge based solution(Reading from files), Please go and create to assistants. Then add System instructions and add your files there: https://platform.openai.com/assistants
## Install

1. Start by creating a user with the bot role in your Rocket.Chat server. Make sure to invite the bot user to the channels where you want it to be active.
2. Collect the user ID for the bot user and set a secure password for it.
3. Rename the [config.yaml.default](config.yaml.default) file to config.yaml.
4. Open the config.yaml file in a text editor and fill it out with the appropriate values. You will need to provide URLs, API tokens, the bot ID, and the bot password for your Rocket.Chat instance. There are comments in the default configuration file to help you understand what each setting does. 
5. If you want to use a different location for the config.yaml file, set the BARTENDER_CONFIG environmental variable to the full path of the file (e.g. BARTENDER_CONFIG=/etc/bartender/config.yaml).
6. Go to the RocketAssist folder and build binary file:
```
go build 

```
7. Run RocketAssist by: 
```
./RocketAssist

```
8. If everything is set up correctly, the bot's status in Rocket.Chat should change to "available" and it will be ready to respond to user input in the specified channels. 


#### Known issues
 - The bot cannot guarantee that the history will not grow bigger than 4k/8k/32k tokens which will trigger an error. To prevent this, do not send very long messages to the bot and do not set the history size too big.
#### Thanks

Thanks to  [Denes Lados](https://github.com/mimrock) for the [Bartender](https://github.com/mimrock/Bartender).

#### License

This project is licensed under the MIT License. See the `LICENSE` file for more details.
