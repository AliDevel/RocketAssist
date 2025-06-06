import json
import time
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional
import requests

from .config import OpenAIConfig, ModelParams


@dataclass
class Message:
    role: str
    content: str


@dataclass
class CompletionResponse:
    choices: List[Dict[str, Any]]
    usage: Dict[str, Any] = field(default_factory=dict)


class OpenAI:
    def __init__(self, cfg: OpenAIConfig):
        self.cfg = cfg
        self.base_url = f"https://{cfg.host_name}"

    def _url(self, endpoint: str) -> str:
        return f"{self.base_url}/{endpoint}"

    def _headers(self) -> Dict[str, str]:
        return {
            "Authorization": f"Bearer {self.cfg.api_token}",
            "Content-Type": "application/json",
        }

    def completion(self, messages: List[Message]) -> CompletionResponse:
        url = self._url(self.cfg.completion_endpoint)
        data = {
            "model": self.cfg.model,
            "messages": [m.__dict__ for m in messages],
        }
        params: ModelParams = self.cfg.model_params
        if params.temperature is not None:
            data["temperature"] = params.temperature
        if params.top_p is not None:
            data["top_p"] = params.top_p
        if params.frequency_penalty is not None:
            data["frequency_penalty"] = params.frequency_penalty
        if params.presence_penalty is not None:
            data["presence_penalty"] = params.presence_penalty
        if params.max_tokens is not None:
            data["max_tokens"] = params.max_tokens
        r = requests.post(url, headers=self._headers(), json=data)
        r.raise_for_status()
        return CompletionResponse(**r.json())

    def moderation(self, text: str) -> Dict[str, Any]:
        url = self._url(self.cfg.moderation_endpoint)
        r = requests.post(url, headers=self._headers(), json={"input": text})
        r.raise_for_status()
        return r.json()

    # Assistant API helpers
    def create_thread(self) -> Dict[str, Any]:
        url = self._url("v1/threads")
        r = requests.post(url, headers=self._headers())
        r.raise_for_status()
        return r.json()

    def add_message_to_thread(self, thread_id: str, msg: Message) -> Dict[str, Any]:
        url = self._url(f"v1/threads/{thread_id}/messages")
        r = requests.post(url, headers=self._headers(), json=msg.__dict__)
        r.raise_for_status()
        return r.json()

    def create_run(self, thread_id: str) -> Dict[str, Any]:
        url = self._url(f"v1/threads/{thread_id}/runs")
        data = {"assistant_id": self.cfg.assistant_id}
        if self.cfg.pre_prompt:
            data["instructions"] = self.cfg.pre_prompt
        r = requests.post(url, headers=self._headers(), json=data)
        r.raise_for_status()
        return r.json()

    def retrieve_run(self, thread_id: str, run_id: str) -> Dict[str, Any]:
        url = self._url(f"v1/threads/{thread_id}/runs/{run_id}")
        r = requests.get(url, headers=self._headers())
        r.raise_for_status()
        return r.json()

    def wait_for_run_completion(self, thread_id: str, run_id: str, poll: float = 5.0, timeout: float = 60.0) -> Dict[str, Any]:
        start = time.time()
        while time.time() - start < timeout:
            status = self.retrieve_run(thread_id, run_id)
            if status.get("status") == "completed":
                return status
            if status.get("status") in {"failed", "cancelled", "expired"}:
                raise RuntimeError(f"run status {status.get('status')}")
            time.sleep(poll)
        raise TimeoutError("run did not complete")

    def get_messages(self, thread_id: str) -> Dict[str, Any]:
        url = self._url(f"v1/threads/{thread_id}/messages")
        r = requests.get(url, headers=self._headers())
        r.raise_for_status()
        return r.json()
