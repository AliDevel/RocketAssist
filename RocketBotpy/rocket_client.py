"""Simplified Rocket.Chat client used for demonstration purposes.
This does not fully replicate the Go implementation but outlines a similar API."""

import json
import threading
from dataclasses import dataclass
from typing import Dict, Optional
import websocket  # type: ignore
import requests

from .config import RocketChatConfig


@dataclass
class Message:
    room_id: str
    room_name: str
    user: str
    user_id: str
    text: str
    is_direct: bool = False
    am_i_pinged: bool = False


class RocketConnection:
    def __init__(self, cfg: RocketChatConfig):
        self.cfg = cfg
        self.ws: Optional[websocket.WebSocket] = None
        self.host = f"{'https' if cfg.ssl else 'http'}://{cfg.host_name}:{cfg.port}"
        self.token = cfg.auth_token

    def connect(self):
        ws_url = self.host.replace('http', 'ws') + "/websocket"
        self.ws = websocket.create_connection(ws_url)

    def login(self):
        if not self.ws:
            return
        payload = {
            "msg": "method",
            "method": "login",
            "id": "1",
            "params": [{"resume": self.token}]
        }
        self.ws.send(json.dumps(payload))
        self.ws.recv()

    def send_message(self, rid: str, text: str):
        url = f"{self.host}/api/v1/chat.postMessage"
        headers = {"X-Auth-Token": self.token, "X-User-Id": self.cfg.user_id}
        requests.post(url, headers=headers, json={"roomId": rid, "text": text})

    # This is a greatly simplified polling mechanism.
    def get_new_message(self) -> Message:
        raise NotImplementedError("Polling messages is not implemented in this example")
