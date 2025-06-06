from dataclasses import dataclass, field
from typing import Dict, List
from datetime import datetime, timedelta

from .openai_client import Message

@dataclass
class TimedMessage:
    message: Message
    timestamp: datetime


@dataclass
class History:
    size: int = 6
    max_length: int = 2048
    expiration: timedelta = timedelta(days=365*100)
    messages: Dict[str, List[TimedMessage]] = field(default_factory=dict)

    def _clear_expired(self, place: str, now: datetime) -> List[TimedMessage]:
        msgs = self.messages.get(place, [])
        return [m for m in msgs if now - m.timestamp <= self.expiration]

    def add(self, place: str, message: Message):
        now = datetime.utcnow()
        self.messages[place] = self._clear_expired(place, now)
        entry = TimedMessage(message=message, timestamp=now)
        msgs = self.messages.setdefault(place, [])
        msgs.append(entry)
        if len(msgs) > self.size:
            self.messages[place] = msgs[-self.size:]

    def clear(self, place: str):
        self.messages[place] = []

    def as_openai_messages(self, place: str) -> List[Message]:
        now = datetime.utcnow()
        self.messages[place] = self._clear_expired(place, now)
        return [m.message for m in self.messages.get(place, [])]

    def get_as_string(self, place: str) -> str:
        msgs = self.as_openai_messages(place)
        return "\n".join(m.content for m in msgs)
