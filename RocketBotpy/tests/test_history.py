import sys
import types
from datetime import timedelta
import unittest

yaml_stub = types.SimpleNamespace(safe_load=lambda t: {})
requests_stub = types.SimpleNamespace(post=lambda *a, **kw: None, get=lambda *a, **kw: None)
websocket_stub = types.SimpleNamespace(create_connection=lambda *a, **kw: None)
sys.modules.setdefault('yaml', yaml_stub)
sys.modules.setdefault('requests', requests_stub)
sys.modules.setdefault('websocket', websocket_stub)

from RocketBotpy.history import History
from RocketBotpy.openai_client import Message


class HistoryTest(unittest.TestCase):
    def test_add_and_get(self):
        hist = History(size=3, expiration=timedelta(seconds=5))
        m1 = Message(role='user', content='hi')
        m2 = Message(role='assistant', content='hello')
        hist.add('room', m1)
        hist.add('room', m2)
        self.assertEqual(hist.get_as_string('room'), 'hi\nhello')

    def test_size_limit(self):
        hist = History(size=2, expiration=timedelta(seconds=5))
        for i in range(3):
            hist.add('r', Message(role='user', content=str(i)))
        self.assertEqual(hist.get_as_string('r'), '1\n2')

    def test_clear(self):
        hist = History()
        hist.add('x', Message(role='user', content='msg'))
        hist.clear('x')
        self.assertEqual(hist.get_as_string('x'), '')


if __name__ == '__main__':
    unittest.main()
