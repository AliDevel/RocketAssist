import sys
import types
import tempfile
from pathlib import Path
import unittest
import importlib

# Expected data returned from yaml.safe_load
expected = {
    'LogLevel': 'debug',
    'RocketChat': {
        'user_id': 'uid123',
        'user': 'bot',
        'password': 'secret',
        'auth_token': 'token',
        'host_name': 'example.com',
        'ssl': True,
        'port': 443,
    },
    'OpenAI': {
        'host_name': 'api.example.com',
        'api_token': 'key',
        'completion_endpoint': 'v1/chat/completions',
        'model': 'gpt-test',
        'history_size': 10,
        'history_max_length': 500,
    }
}

yaml_stub = types.SimpleNamespace(safe_load=lambda text: expected)
requests_stub = types.SimpleNamespace(post=lambda *a, **kw: None, get=lambda *a, **kw: None)
websocket_stub = types.SimpleNamespace(create_connection=lambda *a, **kw: None)
sys.modules['yaml'] = yaml_stub
sys.modules.setdefault('requests', requests_stub)
sys.modules.setdefault('websocket', websocket_stub)

import RocketBotpy.config as config
importlib.reload(config)

load_config = config.load_config
Config = config.Config


class ConfigTest(unittest.TestCase):
    def test_load_config(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            cfg_file = Path(tmpdir) / 'config.yaml'
            cfg_file.write_text('dummy')
            cfg = load_config(cfg_file)
            self.assertIsInstance(cfg, Config)
            self.assertEqual(cfg.log_level, 'debug')
            self.assertEqual(cfg.rocketchat.user_id, 'uid123')
            self.assertEqual(cfg.rocketchat.port, 443)
            self.assertEqual(cfg.openai.host_name, 'api.example.com')
            self.assertEqual(cfg.openai.model, 'gpt-test')
            self.assertEqual(cfg.openai.history_size, 10)
            self.assertEqual(cfg.openai.history_max_length, 500)


if __name__ == '__main__':
    unittest.main()
