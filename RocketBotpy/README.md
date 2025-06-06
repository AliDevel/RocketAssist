# RocketBotpy

This directory contains a Python implementation of the RocketAssist bot. It mirrors the Go functionality using Python classes for configuration, OpenAI interaction and message history.  A lightweight Rocket.Chat client provides basic websocket communication.

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

`main.py` now runs a minimal bot that connects to your Rocket.Chat instance, listens for messages and replies using OpenAI.  Incoming messages are processed concurrently using the `ConcurrencyHandler`.
