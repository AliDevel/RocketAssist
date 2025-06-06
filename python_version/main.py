"""Entry point for the Python version of RocketAssist.
This script demonstrates how the Go logic could be mapped to Python.
It loads configuration and sets up the OpenAI client. Rocket.Chat interaction
is only sketched out for brevity."""

import logging
from pathlib import Path

from .config import load_config
from .openai_client import OpenAI, Message
from .history import History
from .rocket_client import RocketConnection
from .concurrency_handler import ConcurrencyHandler


def main():
    cfg = load_config(Path('config.yaml'))
    logging.basicConfig(level=getattr(logging, cfg.log_level.upper(), logging.INFO))

    oa = OpenAI(cfg.openai)
    history = History(size=cfg.openai.history_size,
                      max_length=cfg.openai.history_max_length)

    rocket = RocketConnection(cfg.rocketchat)
    # In a full implementation rocket.connect() and rocket.login() would be called
    # followed by a loop receiving messages and responding using OpenAI.
    # Here we demonstrate running multiple OpenAI completions concurrently.

    handler = ConcurrencyHandler(workers=2)

    def ask(text: str):
        messages = [Message(role="user", content=text)]
        resp = oa.completion(messages)
        print(resp.choices[0]["message"]["content"])

    for text in ["Hello", "How are you?"]:
        handler.submit(ask, text)

    handler.shutdown()


if __name__ == '__main__':
    main()
