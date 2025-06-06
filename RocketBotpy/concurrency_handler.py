from concurrent.futures import ThreadPoolExecutor
from typing import Callable, Iterable


class ConcurrencyHandler:
    """Simple wrapper around ThreadPoolExecutor for running tasks concurrently."""

    def __init__(self, workers: int = 4):
        self._executor = ThreadPoolExecutor(max_workers=workers)

    def submit(self, fn: Callable, *args, **kwargs):
        """Submit a task to be executed concurrently."""
        return self._executor.submit(fn, *args, **kwargs)

    def map(self, fn: Callable, iterable: Iterable):
        """Convenience method to map a function over an iterable concurrently."""
        return self._executor.map(fn, iterable)

    def shutdown(self, wait: bool = True):
        self._executor.shutdown(wait=wait)
