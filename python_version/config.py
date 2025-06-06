import yaml
from dataclasses import dataclass, field
from typing import Optional
from pathlib import Path

@dataclass
class ModelParams:
    temperature: Optional[float] = None
    top_p: Optional[float] = None
    frequency_penalty: Optional[float] = None
    presence_penalty: Optional[float] = None
    max_tokens: Optional[int] = None

@dataclass
class RocketChatConfig:
    user_id: str = ""
    user: str = ""
    password: str = ""
    auth_token: str = ""
    host_name: str = ""
    ssl: bool = True
    port: int = 80

@dataclass
class OpenAIConfig:
    host_name: str = "api.openai.com"
    api_token: str = ""
    completion_endpoint: str = "v1/chat/completions"
    moderation_endpoint: str = "v1/moderations"
    assistance_endpoint: str = "v1/assistants"
    model: str = "gpt-3.5-turbo"
    history_size: int = 6
    history_max_length: int = 2048
    message_retention: Optional[str] = None
    pre_prompt: str = ""
    input_moderation: bool = False
    output_moderation: bool = False
    send_user_id: bool = False
    model_params: ModelParams = field(default_factory=ModelParams)
    assistant_id: str = ""

@dataclass
class Config:
    log_level: str = "info"
    rocketchat: RocketChatConfig = field(default_factory=RocketChatConfig)
    openai: OpenAIConfig = field(default_factory=OpenAIConfig)


def load_config(path: Path) -> Config:
    data = yaml.safe_load(path.read_text())
    openai_cfg = data.get('OpenAI', {})
    rocketchat_cfg = data.get('RocketChat', {})
    cfg = Config(
        log_level=data.get('LogLevel', 'info'),
        rocketchat=RocketChatConfig(**{k.lower(): v for k, v in rocketchat_cfg.items()}),
        openai=OpenAIConfig(**{k.lower(): v for k, v in openai_cfg.items()})
    )
    return cfg
