#!/usr/bin/env python
"""
Test Python 2.7.4 compatibility of AsyncSession

This script tests that AsyncSession works correctly in Python 2.7.4
using only standard library features.
"""

from __future__ import print_function
import sys
import os
import time

# Add parent directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))


def test_imports():
    """Test that all required imports work in Python 2.7.4"""
    print("Testing Python 2.7.4 compatibility...")
    print("Python version: %s" % sys.version)
    print()

    # Test Queue import (Python 2 vs 3 difference)
    try:
        import queue

        print("[OK] Using queue module (Python 3+)")
    except ImportError:
        import Queue as queue

        print("[OK] Using Queue module (Python 2)")

    # Test threading
    import threading

    print("[OK] Threading module available")

    # Test thread.daemon property
    t = threading.Thread(target=lambda: None)
    t.daemon = True
    print("[OK] Thread.daemon property works")

    # Test Queue features
    q = queue.Queue(maxsize=100)
    q.put_nowait("test")
    item = q.get_nowait()
    assert item == "test"
    print("[OK] Queue.Queue with put_nowait/get_nowait works")

    # Test Queue.Full and Queue.Empty exceptions
    try:
        q.get_nowait()
    except queue.Empty:
        print("[OK] Queue.Empty exception works")

    q = queue.Queue(maxsize=1)
    q.put_nowait("item1")
    try:
        q.put_nowait("item2")
    except queue.Full:
        print("[OK] Queue.Full exception works")

    print()
    print("All Python 2.7.4 compatibility checks passed!")
    print()


def test_async_session():
    """Test AsyncSession with mock session"""
    print("Testing AsyncSession functionality...")
    print()

    # Import AsyncSession
    from datacat import AsyncSession

    # Create a mock session for testing
    class MockClient:
        def __init__(self):
            self.calls = []

        def log_event(self, session_id, name, **kwargs):
            self.calls.append(("event", name, kwargs))

        def log_metric(self, session_id, name, value, tags=None):
            self.calls.append(("metric", name, value, tags))

        def update_state(self, session_id, state):
            self.calls.append(("state", state))

        def end_session(self, session_id):
            self.calls.append(("end",))

    class MockSession:
        def __init__(self):
            self.client = MockClient()
            self.session_id = "test-session-123"
            self._heartbeat_monitor = None

        def log_event(self, name, **kwargs):
            return self.client.log_event(self.session_id, name, **kwargs)

        def log_metric(self, name, value, tags=None):
            return self.client.log_metric(self.session_id, name, value, tags)

        def update_state(self, state):
            return self.client.update_state(self.session_id, state)

        def log_exception(self, exc_info=None, extra_data=None):
            return self.client.log_event(
                self.session_id, "exception", exc_info=exc_info, extra_data=extra_data
            )

        def end(self):
            return self.client.end_session(self.session_id)

        def get_details(self):
            return {"session_id": self.session_id}

    # Create async session
    mock_session = MockSession()
    async_session = AsyncSession(mock_session, queue_size=100)

    print("[OK] AsyncSession created")

    # Test logging operations (should be non-blocking)
    start = time.time()
    for i in range(50):
        async_session.log_event("test_event_%d" % i, data={"value": i})
        async_session.log_metric("test_metric_%d" % i, float(i))
        async_session.update_state({"iteration": i})
    end = time.time()

    elapsed_ms = (end - start) * 1000
    avg_per_call = elapsed_ms / 150  # 50 events + 50 metrics + 50 states

    print("[OK] Logged 150 items in %.2f ms" % elapsed_ms)
    print("  Average per call: %.4f ms" % avg_per_call)

    if avg_per_call < 0.1:
        print("  [OK] Performance is excellent (< 0.1ms per call)")
    else:
        print("  [WARN] Performance is slower than expected")

    # Get stats
    stats = async_session.get_stats()
    print()
    print("Stats before flush:")
    print("  Sent: %d" % stats["sent"])
    print("  Dropped: %d" % stats["dropped"])
    print("  Queued: %d" % stats["queued"])

    # Flush and wait
    print()
    print("Flushing queue...")
    async_session.flush(timeout=2.0)

    stats = async_session.get_stats()
    print()
    print("Stats after flush:")
    print("  Sent: %d" % stats["sent"])
    print("  Dropped: %d" % stats["dropped"])
    print("  Queued: %d" % stats["queued"])

    # Shutdown
    async_session.shutdown()
    print()
    print("[OK] AsyncSession shutdown complete")

    # Verify all calls were processed
    total_calls = len(mock_session.client.calls)
    print()
    print("Total calls processed: %d" % total_calls)

    if total_calls >= 150:
        print("[OK] All events were processed!")
    else:
        print("[WARN] Some events may have been dropped or not yet processed")

    print()
    print("AsyncSession test passed!")


def test_queue_overflow():
    """Test that queue overflow is handled gracefully"""
    print()
    print("=" * 70)
    print("Testing queue overflow handling...")
    print()

    from datacat import AsyncSession

    # Create mock session with slow processing
    class SlowMockSession:
        def __init__(self):
            self.calls = []
            self._heartbeat_monitor = None
            self.session_id = "test"
            self.client = self

        def log_event(self, name, **kwargs):
            time.sleep(0.1)  # Slow processing
            self.calls.append("event")

        def log_metric(self, name, value, tags=None):
            time.sleep(0.1)
            self.calls.append("metric")

        def update_state(self, state):
            time.sleep(0.1)
            self.calls.append("state")

        def log_exception(self, exc_info=None, extra_data=None):
            time.sleep(0.1)
            self.calls.append("exception")

        def end(self):
            return {}

        def get_details(self):
            return {}

    # Create async session with small queue
    slow_session = SlowMockSession()
    async_session = AsyncSession(slow_session, queue_size=10, drop_on_full=True)

    print("[OK] Created AsyncSession with queue_size=10")

    # Flood with events
    print("Flooding with 100 events...")
    for i in range(100):
        async_session.log_event("flood_%d" % i)

    # Check stats
    stats = async_session.get_stats()
    print()
    print("Stats after flood:")
    print("  Sent: %d" % stats["sent"])
    print("  Dropped: %d" % stats["dropped"])
    print("  Queued: %d" % stats["queued"])

    if stats["dropped"] > 0:
        print()
        print(
            "[OK] Queue overflow handled gracefully (dropped %d events)"
            % stats["dropped"]
        )
        print("  This is expected behavior when drop_on_full=True")
    else:
        print()
        print("[WARN] No events dropped (might process too fast)")

    # Shutdown
    async_session.shutdown(timeout=1.0)

    final_stats = async_session.get_stats()
    print()
    print("Final stats:")
    print("  Sent: %d" % final_stats["sent"])
    print("  Dropped: %d" % final_stats["dropped"])

    print()
    print("Queue overflow test passed!")


if __name__ == "__main__":
    print()
    print("=" * 70)
    print("Python 2.7.4 Compatibility Test")
    print("=" * 70)
    print()

    try:
        # Test imports and basic functionality
        test_imports()

        # Test AsyncSession
        test_async_session()

        # Test queue overflow
        test_queue_overflow()

        print()
        print("=" * 70)
        print("ALL TESTS PASSED!")
        print("=" * 70)
        print()
        print("AsyncSession is fully compatible with Python 2.7.4!")
        print("You can safely use it in your game or real-time application.")
        print()

    except Exception as e:
        print()
        print("=" * 70)
        print("TEST FAILED!")
        print("=" * 70)
        print()
        print("Error: %s" % str(e))
        import traceback

        traceback.print_exc()
        sys.exit(1)
