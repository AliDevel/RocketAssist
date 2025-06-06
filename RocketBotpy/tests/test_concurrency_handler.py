import sys
import types
import time
import unittest

yaml_stub = types.SimpleNamespace(safe_load=lambda t: {})
requests_stub = types.SimpleNamespace(post=lambda *a, **kw: None, get=lambda *a, **kw: None)
websocket_stub = types.SimpleNamespace(create_connection=lambda *a, **kw: None)
sys.modules.setdefault('yaml', yaml_stub)
sys.modules.setdefault('requests', requests_stub)
sys.modules.setdefault('websocket', websocket_stub)

from RocketBotpy.concurrency_handler import ConcurrencyHandler


class ConcurrencyHandlerTest(unittest.TestCase):
    def test_runs_tasks(self):
        handler = ConcurrencyHandler(workers=2)
        results = []

        def task(x):
            time.sleep(0.1)
            results.append(x)

        for i in range(3):
            handler.submit(task, i)
        handler.shutdown()

        self.assertEqual(sorted(results), [0, 1, 2])


if __name__ == '__main__':
    unittest.main()
