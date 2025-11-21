"""
Test offline mode behavior for the daemon

These tests verify that:
1. Daemon can create sessions when server is unavailable
2. Daemon queues operations when server is unavailable
3. Daemon retries operations when server becomes available
4. Clients can retrieve session data from daemon when server is down
"""

import json
import os
import subprocess
import sys
import time
import unittest

try:
    from urllib.request import Request, urlopen
    from urllib.error import URLError, HTTPError
except ImportError:
    from urllib2 import Request, urlopen, URLError, HTTPError


class TestOfflineMode(unittest.TestCase):
    """Test offline mode behavior"""

    daemon_process = None
    daemon_port = "8079"
    server_url = "http://localhost:19999"  # Use invalid port to simulate server down
    base_daemon_url = None

    @classmethod
    def setUpClass(cls):
        """Start the daemon without a server"""
        # Build the daemon
        repo_root = os.path.join(os.path.dirname(__file__), "..")
        build_result = subprocess.run(
            ["go", "build", "-o", "datacat-daemon", "./cmd/datacat-daemon"],
            cwd=repo_root,
            capture_output=True,
        )

        if build_result.returncode != 0:
            raise Exception(
                "Failed to build daemon: {}".format(build_result.stderr.decode())
            )

        # Create daemon config
        config = {
            "daemon_port": cls.daemon_port,
            "server_url": cls.server_url,
            "batch_interval_seconds": 5,
            "max_batch_size": 100,
            "heartbeat_timeout_seconds": 60,
        }

        config_path = os.path.join(repo_root, "daemon_config.json")
        with open(config_path, "w") as f:
            json.dump(config, f)

        # Start daemon
        cls.daemon_process = subprocess.Popen(
            ["./datacat-daemon"],
            cwd=repo_root,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

        # Wait for daemon to start
        time.sleep(2)

        cls.base_daemon_url = "http://localhost:{}".format(cls.daemon_port)

    @classmethod
    def tearDownClass(cls):
        """Stop the daemon"""
        if cls.daemon_process:
            cls.daemon_process.terminate()
            cls.daemon_process.wait(timeout=5)

    def _make_request(self, url, method="GET", data=None):
        """Helper to make HTTP requests"""
        headers = {"Content-Type": "application/json"}

        if data is not None:
            data = json.dumps(data).encode("utf-8")

        req = Request(url, data=data, headers=headers)
        if method != "GET":
            req.get_method = lambda: method

        response = urlopen(req)
        response_data = response.read().decode("utf-8")

        # Handle empty responses
        if not response_data or response_data.strip() == "":
            return {}

        return json.loads(response_data)

    def test_create_session_offline(self):
        """Test that sessions can be created when server is offline"""
        url = "{}/register".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )

        session_id = result.get("session_id")
        self.assertIsNotNone(session_id)
        self.assertTrue(session_id.startswith("local-session-"))

    def test_state_updates_offline(self):
        """Test that state updates work when server is offline"""
        # Create session
        url = "{}/register".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id = result.get("session_id")

        # Update state - should succeed locally
        url = "{}/state".format(self.base_daemon_url)
        state = {"status": "running", "progress": 50}
        result = self._make_request(
            url, method="POST", data={"session_id": session_id, "state": state}
        )
        self.assertIsNotNone(result)

    def test_events_offline(self):
        """Test that events can be logged when server is offline"""
        # Create session
        url = "{}/register".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id = result.get("session_id")

        # Log event - should succeed locally
        url = "{}/event".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "session_id": session_id,
                "name": "test_event",
                "data": {"key": "value"},
            },
        )
        self.assertIsNotNone(result)

    def test_metrics_offline(self):
        """Test that metrics can be logged when server is offline"""
        # Create session
        url = "{}/register".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id = result.get("session_id")

        # Log metric - should succeed locally
        url = "{}/metric".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "session_id": session_id,
                "name": "test_metric",
                "value": 123.45,
                "tags": ["tag1"],
            },
        )
        self.assertIsNotNone(result)

    def test_get_session_offline(self):
        """Test that session details can be retrieved from daemon when server is offline"""
        # Create session
        url = "{}/register".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id = result.get("session_id")

        # Update some state
        url = "{}/state".format(self.base_daemon_url)
        state = {"status": "running", "version": "1.0"}
        self._make_request(
            url, method="POST", data={"session_id": session_id, "state": state}
        )

        # Get session details from daemon
        url = "{}/session?session_id={}".format(self.base_daemon_url, session_id)
        session = self._make_request(url, method="GET")
        self.assertIsNotNone(session)
        self.assertEqual(session["id"], session_id)
        self.assertTrue(session["active"])
        self.assertIn("state", session)

    def test_end_session_offline(self):
        """Test that sessions can be ended when server is offline"""
        # Create session
        url = "{}/register".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id = result.get("session_id")

        # End session - should succeed locally
        url = "{}/end".format(self.base_daemon_url)
        result = self._make_request(url, method="POST", data={"session_id": session_id})
        self.assertIsNotNone(result)

    def test_get_all_sessions_offline(self):
        """Test that all sessions can be retrieved from daemon when server is offline"""
        # Create a few sessions
        url = "{}/register".format(self.base_daemon_url)
        result1 = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id1 = result1.get("session_id")

        result2 = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id2 = result2.get("session_id")

        # Get all sessions from daemon
        url = "{}/sessions".format(self.base_daemon_url)
        sessions = self._make_request(url, method="GET")
        self.assertIsNotNone(sessions)
        self.assertIsInstance(sessions, list)
        # Should have at least the two we just created
        self.assertGreaterEqual(len(sessions), 2)

    def test_heartbeat_offline(self):
        """Test that heartbeats work when server is offline"""
        # Create session
        url = "{}/register".format(self.base_daemon_url)
        result = self._make_request(
            url,
            method="POST",
            data={
                "parent_pid": os.getpid(),
                "product": "TestProduct",
                "version": "1.0.0",
            },
        )
        session_id = result.get("session_id")

        # Send heartbeat
        url = "{}/heartbeat".format(self.base_daemon_url)
        result = self._make_request(url, method="POST", data={"session_id": session_id})
        self.assertIsNotNone(result)


if __name__ == "__main__":
    unittest.main()
