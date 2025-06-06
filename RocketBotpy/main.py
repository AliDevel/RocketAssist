"""Entry point for the Python version of RocketAssist.

This module puts together configuration loading, the OpenAI client and a small
Rocket.Chat connection to form a minimal bot.  It mirrors the Go example but is
implemented purely in Python."""

import logging
from pathlib import Path
from typing import Dict

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
    rocket.connect()
    rocket.login()
    rocket.subscribe_room(cfg.rocketchat.user_id)

    handler = ConcurrencyHandler(workers=2)
    thread_map: Dict[str, str] = {}

    def process(msg):
        if msg.user_id == cfg.rocketchat.user_id:
            return

        history.add(msg.room_id, Message(role="user", content=msg.text))

        if cfg.openai.assistant_id:
            thread_id = thread_map.get(msg.room_id)
            if not thread_id:
                thread = oa.create_thread()
                thread_id = thread.get("id") or thread.get("thread_id")
                thread_map[msg.room_id] = thread_id
            oa.add_message_to_thread(thread_id, Message(role="user", content=msg.text))
            run = oa.create_run(thread_id)
            oa.wait_for_run_completion(thread_id, run.get("id") or run.get("run_id"))
            messages = oa.get_messages(thread_id)
            reply_text = messages["messages"][0]["content"][0]["text"]["value"]
        else:
            msgs = []
            if cfg.openai.pre_prompt:
                msgs.append(Message(role="system", content=cfg.openai.pre_prompt))
            msgs.extend(history.as_openai_messages(msg.room_id))
            msgs.append(Message(role="user", content=msg.text))
            comp = oa.completion(msgs)
            reply_text = comp.choices[0]["message"]["content"]

        rocket.send_message(msg.room_id, reply_text)
        history.add(msg.room_id, Message(role="assistant", content=reply_text))

    for incoming in rocket.iter_messages():
        handler.submit(process, incoming)


if __name__ == '__main__':
    main()
