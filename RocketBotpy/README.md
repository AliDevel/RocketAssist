# RocketBotpy

This directory contains a Python implementation of the RocketAssist bot. It mirrors the Go functionality using Python classes for configuration, OpenAI interaction, message history and a simplified Rocket.Chat client.

## Running the Python version

1. Install the required dependencies:
   ```bash
   pip install -r ../requirements.txt
   ```
2. Copy `config.yaml.default` from the repository root to `config.yaml` and fill in your configuration values.
3. Run the bot using:
   ```bash
   python -m RocketBotpy.main
   ```

The example in `main.py` demonstrates using the concurrency handler to perform multiple OpenAI completions in parallel. The Rocket.Chat client is only a placeholder and does not include message polling.
