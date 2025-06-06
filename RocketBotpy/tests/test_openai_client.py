import sys
import types
from unittest import mock, TestCase, main

yaml_stub = types.SimpleNamespace(safe_load=lambda t: {})
requests_stub = types.SimpleNamespace(post=lambda *a, **kw: None, get=lambda *a, **kw: None)
websocket_stub = types.SimpleNamespace(create_connection=lambda *a, **kw: None)
sys.modules.setdefault('yaml', yaml_stub)
sys.modules.setdefault('requests', requests_stub)
sys.modules.setdefault('websocket', websocket_stub)

from RocketBotpy.openai_client import OpenAI, Message, CompletionResponse
from RocketBotpy.config import OpenAIConfig


def make_client():
    cfg = OpenAIConfig(api_token='t', host_name='api.test', model='gpt')
    return OpenAI(cfg)


class OpenAITest(TestCase):
    def test_completion_sends_request(self):
        client = make_client()
        messages = [Message(role='user', content='hi')]
        with mock.patch('requests.post') as post:
            post.return_value.json.return_value = {
                'choices': [{'message': {'content': 'ok'}}],
                'usage': {}
            }
            post.return_value.raise_for_status.return_value = None
            resp = client.completion(messages)
            self.assertIsInstance(resp, CompletionResponse)
            post.assert_called_once()
            args, kwargs = post.call_args
            self.assertEqual(args[0], 'https://api.test/v1/chat/completions')
            self.assertEqual(kwargs['json']['messages'][0]['content'], 'hi')


if __name__ == '__main__':
    main()
