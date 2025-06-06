"""Simplified Rocket.Chat client.

This implementation provides the minimum needed features for the Python
version of RocketAssist. It connects to the websocket endpoint, performs a
token based login and allows subscribing to rooms and iterating over incoming
messages."""

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
        self._next_id = 0
        self.channels: Dict[str, str] = {}

    def connect(self):
        ws_url = self.host.replace('http', 'ws') + "/websocket"
        self.ws = websocket.create_connection(ws_url)
        self.ws.send(json.dumps({"msg": "connect", "version": "1", "support": ["1"]}))
        self.ws.recv()

    def login(self):
        if not self.ws:
            return
        self._next_id += 1
        payload = {
            "msg": "method",
            "method": "login",
            "id": str(self._next_id),
            "params": [{"resume": self.token}]
        }
        self.ws.send(json.dumps(payload))
        self.ws.recv()

    def subscribe_room(self, rid: str, name: str = ""):
        if not self.ws:
            return
        self.channels[rid] = name
        self._next_id += 1
        payload = {
            "msg": "sub",
            "id": str(self._next_id),
            "name": "stream-room-messages",
            "params": [rid, False],
        }
        self.ws.send(json.dumps(payload))

    def send_message(self, rid: str, text: str):
        url = f"{self.host}/api/v1/chat.postMessage"
        headers = {"X-Auth-Token": self.token, "X-User-Id": self.cfg.user_id}
        requests.post(url, headers=headers, json={"roomId": rid, "text": text})

    def iter_messages(self):
        """Yield incoming messages from subscribed rooms."""
        if not self.ws:
            return
        while True:
            raw = self.ws.recv()
            data = json.loads(raw)
            if data.get("msg") == "ping":
                self.ws.send(json.dumps({"msg": "pong"}))
                continue
            if data.get("collection") != "stream-room-messages":
                continue
            fields = data.get("fields", {})
            args = fields.get("args", [])
            if not args:
                continue
            obj = args[0]
            rid = obj.get("rid", "")
            yield Message(
                room_id=rid,
                room_name=self.channels.get(rid, ""),
                user=obj.get("u", {}).get("username", ""),
                user_id=obj.get("u", {}).get("_id", ""),
                text=obj.get("msg", ""),
                is_direct=rid.startswith(self.cfg.user_id),
                am_i_pinged=f"@{self.cfg.user}" in obj.get("msg", ""),
            )
