"""
Integration tests for datacat service

These tests verify:
1. Session creation and retrieval
2. State updates and nested state merging
3. Event and metric logging
4. Exception logging

Note: Data persistence across service restarts is tested in the Go unit tests
(TestPersistence and TestComprehensivePersistence in cmd/datacat-server/main_test.go)
to avoid race conditions with async saves and daemon batching.
"""

import json
import os
import subprocess
import sys
import time
import unittest

# Add python directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import DatacatClient, create_session


class TestDatacatIntegration(unittest.TestCase):
    """Integration tests for datacat service"""

    service_process: subprocess.Popen  # type: ignore
    base_url: str
    shared_client: DatacatClient  # type: ignore

    @classmethod
    def setUpClass(cls):
        """Start the datacat service before tests"""
        cls.service_process = None  # type: ignore
        cls.base_url = "http://localhost:9090"

        # Build the service and daemon
        repo_root = os.path.join(os.path.dirname(__file__), "..")

        # Build server
        build_result = subprocess.run(
            ["go", "build", "-o", "datacat", "./cmd/datacat-server"],
            cwd=repo_root,
            capture_output=True,
        )
        if build_result.returncode != 0:
            raise Exception(f"Failed to build service: {build_result.stderr.decode()}")

        # Build daemon (required by DatacatClient)
        daemon_build_result = subprocess.run(
            ["go", "build", "-o", "datacat-daemon", "./cmd/datacat-daemon"],
            cwd=repo_root,
            capture_output=True,
        )
        if daemon_build_result.returncode != 0:
            raise Exception(
                f"Failed to build daemon: {daemon_build_result.stderr.decode()}"
            )

        # Start the service
        cls.service_process = subprocess.Popen(
            ["./datacat"],
            cwd=os.path.join(os.path.dirname(__file__), ".."),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

        # Wait for service to start
        time.sleep(2)

        # Create shared client and daemon for all tests
        try:
            cls.shared_client = DatacatClient(cls.base_url)
            # Verify service is running
            test_session = cls.shared_client.register_session(
                "IntegrationTest", "1.0.0"
            )
            cls.shared_client.end_session(test_session)
        except Exception as e:
            cls.tearDownClass()
            raise Exception(f"Service failed to start: {e}")

    @classmethod
    def tearDownClass(cls):
        """Stop the datacat service after tests"""
        # Stop shared daemon
        if hasattr(cls, "shared_client") and cls.shared_client.daemon_manager:
            cls.shared_client.daemon_manager.stop()
        # Stop service
        if cls.service_process:
            cls.service_process.terminate()
            cls.service_process.wait(timeout=5)

    def setUp(self):
        """Create a fresh session for each test"""
        self.client = self.shared_client
        self.session_id = self.client.register_session("IntegrationTest", "1.0.0")

    def test_session_creation(self):
        """Test that sessions can be created and retrieved"""
        session = self.client.get_session(self.session_id)

        self.assertEqual(session["id"], self.session_id)
        self.assertTrue(session["active"])
        self.assertIn("created_at", session)
        self.assertIn("updated_at", session)

    def test_state_updates(self):
        """Test state updates"""
        # Update state
        state1 = {"status": "running", "progress": 0}
        self.client.update_state(self.session_id, state1)

        # Wait for daemon to flush
        time.sleep(6)

        # Retrieve and verify
        session = self.client.get_session(self.session_id)
        self.assertEqual(session["state"]["status"], "running")
        self.assertEqual(session["state"]["progress"], 0)

    def test_nested_state_merge(self):
        """Test that nested state updates merge correctly"""
        # Set initial nested state
        state1 = {
            "window_state": {"open": ["w1", "w2"], "active": "w1"},
            "memory": {"footprint_mb": 50},
        }
        self.client.update_state(self.session_id, state1)

        # Wait for daemon to flush
        time.sleep(6)

        # Update only part of window_state
        state2 = {"window_state": {"open": ["w1", "w2", "w3"]}}
        self.client.update_state(self.session_id, state2)

        # Wait for daemon to flush
        time.sleep(6)

        # Verify merge preserved active window
        session = self.client.get_session(self.session_id)
        self.assertEqual(session["state"]["window_state"]["open"], ["w1", "w2", "w3"])
        self.assertEqual(session["state"]["window_state"]["active"], "w1")
        self.assertEqual(session["state"]["memory"]["footprint_mb"], 50)

    def test_event_logging(self):
        """Test event logging"""
        # Log an event
        self.client.log_event(
            self.session_id, "test_event", data={"user": "alice", "action": "click"}
        )

        # Wait for daemon to flush
        time.sleep(6)

        # Retrieve and verify
        session = self.client.get_session(self.session_id)
        self.assertEqual(len(session["events"]), 1)
        self.assertEqual(session["events"][0]["name"], "test_event")
        self.assertEqual(session["events"][0]["data"]["user"], "alice")

    def test_metric_logging(self):
        """Test metric logging"""
        # Log a metric
        self.client.log_metric(
            self.session_id, "cpu_usage", 45.5, tags=["host:server1"]
        )

        # Wait for daemon to flush
        time.sleep(6)

        # Retrieve and verify
        session = self.client.get_session(self.session_id)
        self.assertEqual(len(session["metrics"]), 1)
        self.assertEqual(session["metrics"][0]["name"], "cpu_usage")
        self.assertEqual(session["metrics"][0]["value"], 45.5)
        self.assertIn("host:server1", session["metrics"][0]["tags"])

    def test_exception_logging(self):
        """Test exception logging"""
        # Create an exception
        try:
            raise ValueError("Test exception")
        except ValueError:
            exc_info = sys.exc_info()
            self.client.log_exception(
                self.session_id, exc_info, extra_data={"context": "test"}
            )

        # Wait for daemon to flush
        time.sleep(6)

        # Retrieve and verify
        session = self.client.get_session(self.session_id)
        self.assertEqual(len(session["events"]), 1)
        self.assertEqual(session["events"][0]["name"], "exception")
        self.assertEqual(session["events"][0]["exception_type"], "ValueError")
        self.assertIn("Test exception", session["events"][0]["exception_msg"])
        self.assertIsNotNone(session["events"][0]["stacktrace"])
        self.assertEqual(session["events"][0]["data"]["context"], "test")

    def test_session_end(self):
        """Test ending a session"""
        # End the session
        self.client.end_session(self.session_id)

        # Verify session is inactive
        session = self.client.get_session(self.session_id)
        self.assertFalse(session["active"])
        self.assertIn("ended_at", session)

    def test_convenience_session_class(self):
        """Test the convenience Session class"""
        # Use different port to avoid conflict with shared daemon
        session = create_session(
            self.base_url,
            daemon_port="8080",
            product="IntegrationTest",
            version="1.0.0",
        )

        try:
            # Test state update
            session.update_state({"test": "value"})

            # Test event logging
            session.log_event("test_event", data={"data": "test"})

            # Test metric logging
            session.log_metric("test_metric", 100.0)

            # Test exception logging
            try:
                raise RuntimeError("Test error")
            except RuntimeError:
                session.log_exception()

            # Wait for daemon to flush
            time.sleep(6)

            # Verify all data
            details = session.get_details()
            self.assertEqual(details["state"]["test"], "value")
            self.assertEqual(len(details["events"]), 2)  # test_event + exception
            self.assertEqual(len(details["metrics"]), 1)

            # End session
            session.end()
            time.sleep(1)
            details = session.get_details()
            self.assertFalse(details["active"])
        finally:
            # Clean up the daemon created by create_session
            if hasattr(session, "client") and hasattr(session.client, "daemon_manager"):
                session.client.daemon_manager.stop()

    def test_data_persistence_across_restart(self):
        """Test that session data persists across server restarts"""
        # Create a session with some data
        initial_state = {"app": "test", "version": "1.0", "count": 42}
        self.client.update_state(self.session_id, initial_state)

        # Log an event
        self.client.log_event(
            self.session_id,
            "test_persistence_event",
            data={"message": "before restart"},
        )

        # Log a metric
        self.client.log_metric(self.session_id, "test_metric", 100.5, tags=["test"])

        # Wait for daemon to flush
        time.sleep(6)

        # Store the session ID for later retrieval
        session_id = self.session_id

        # Stop the server
        self.service_process.terminate()
        self.service_process.wait(timeout=5)
        time.sleep(1)

        # Restart the server
        repo_root = os.path.join(os.path.dirname(__file__), "..")
        self.service_process = subprocess.Popen(
            ["./datacat"],
            cwd=repo_root,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

        # Wait for service to restart
        time.sleep(2)

        # Retrieve the session - it should still exist
        session = self.client.get_session(session_id)

        # Verify session data was persisted
        self.assertEqual(session["id"], session_id)
        self.assertEqual(session["state"]["app"], "test")
        self.assertEqual(session["state"]["version"], "1.0")
        self.assertEqual(session["state"]["count"], 42)

        # Verify events persisted
        self.assertEqual(len(session["events"]), 1)
        self.assertEqual(session["events"][0]["name"], "test_persistence_event")
        self.assertEqual(session["events"][0]["data"]["message"], "before restart")

        # Verify metrics persisted
        self.assertEqual(len(session["metrics"]), 1)
        self.assertEqual(session["metrics"][0]["name"], "test_metric")
        self.assertEqual(session["metrics"][0]["value"], 100.5)


if __name__ == "__main__":
    unittest.main()
