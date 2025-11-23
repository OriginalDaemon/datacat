"""
Unit tests for datacat Python client

These tests focus on testing individual components in isolation,
including error handling, HeartbeatMonitor, and edge cases.
"""

import json
import os
import subprocess
import sys
import threading
import time
import unittest
from unittest.mock import Mock, patch, MagicMock

# Add python directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import (
    DatacatClient,
    DaemonManager,
    HeartbeatMonitor,
    Session,
    create_session,
)

try:
    from urllib.error import HTTPError, URLError
except ImportError:
    from urllib2 import HTTPError, URLError


class TestDatacatClientErrorHandling(unittest.TestCase):
    """Test error handling in DatacatClient"""

    @patch("datacat.DaemonManager")
    def setUp(self, mock_daemon_class):
        """Set up test client"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon
        self.client = DatacatClient("http://localhost:9090")

    def tearDown(self):
        """Clean up test client"""
        if hasattr(self, "client") and hasattr(self.client, "daemon_manager"):
            self.client.daemon_manager.stop()

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_http_error_handling(self, mock_daemon_class, mock_urlopen):
        """Test handling of HTTP errors"""
        # Set up daemon mock
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        # Create a mock HTTPError
        mock_error = HTTPError(
            url="http://test.com",
            code=404,
            msg="Not Found",
            hdrs={},
            fp=None,
        )
        mock_urlopen.side_effect = mock_error

        # Verify exception is raised with proper message
        with self.assertRaises(Exception) as context:
            self.client.register_session("TestProduct", "1.0.0")

        self.assertIn("HTTP Error 404", str(context.exception))
        self.assertIn("Not Found", str(context.exception))

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_url_error_handling(self, mock_daemon_class, mock_urlopen):
        """Test handling of URL errors (network issues)"""
        # Set up daemon mock
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        # Create a mock URLError
        mock_error = URLError("Connection refused")
        mock_urlopen.side_effect = mock_error

        # Verify exception is raised with proper message
        with self.assertRaises(Exception) as context:
            self.client.register_session("TestProduct", "1.0.0")

        self.assertIn("URL Error", str(context.exception))
        self.assertIn("Connection refused", str(context.exception))

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_general_exception_handling(self, mock_daemon_class, mock_urlopen):
        """Test handling of general exceptions"""
        # Set up daemon mock
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        # Create a general exception
        mock_urlopen.side_effect = RuntimeError("Unexpected error")

        # Verify exception is raised with proper message
        with self.assertRaises(Exception) as context:
            self.client.register_session("TestProduct", "1.0.0")

        self.assertIn("Request failed", str(context.exception))
        self.assertIn("Unexpected error", str(context.exception))

    @patch("datacat.DaemonManager")
    def test_base_url_trailing_slash_removal(self, mock_daemon_class):
        """Test that trailing slash is removed from base URL"""
        # Set up daemon mock
        mock_daemon = Mock()
        mock_daemon.daemon_port = "8079"  # Set the daemon port on the mock
        mock_daemon_class.return_value = mock_daemon

        client = DatacatClient("http://localhost:9090/")
        # Client always uses daemon URL, not the server URL directly
        self.assertEqual(client.base_url, "http://localhost:8079")

        client2 = DatacatClient("http://localhost:9090")
        self.assertEqual(client2.base_url, "http://localhost:8079")


class TestDatacatClientGetAllSessions(unittest.TestCase):
    """Test get_all_sessions method"""

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_get_all_sessions(self, mock_daemon_class, mock_urlopen):
        """Test get_all_sessions calls the correct endpoint"""
        # Set up daemon mock
        mock_daemon = Mock()
        mock_daemon.daemon_port = "8079"  # Set the daemon port on the mock
        mock_daemon_class.return_value = mock_daemon

        # Mock response
        mock_response = Mock()
        mock_response.read.return_value = b'[{"id": "session1"}, {"id": "session2"}]'
        mock_urlopen.return_value = mock_response

        client = DatacatClient("http://localhost:9090")
        sessions = client.get_all_sessions()

        # Verify correct endpoint was called (daemon endpoint, not server)
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertEqual(request.get_full_url(), "http://localhost:8079/sessions")

        # Verify response parsing
        self.assertEqual(len(sessions), 2)
        self.assertEqual(sessions[0]["id"], "session1")
        self.assertEqual(sessions[1]["id"], "session2")


class TestHeartbeatMonitor(unittest.TestCase):
    """Test HeartbeatMonitor functionality"""

    def setUp(self):
        """Set up test components"""
        self.mock_client = Mock(spec=DatacatClient)
        self.mock_client.use_daemon = False
        self.session_id = "test-session-123"

    def test_monitor_initialization(self):
        """Test HeartbeatMonitor initialization"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=30,
            check_interval=2,
        )

        self.assertEqual(monitor.client, self.mock_client)
        self.assertEqual(monitor.session_id, self.session_id)
        self.assertEqual(monitor.timeout, 30)
        self.assertEqual(monitor.check_interval, 2)
        self.assertFalse(monitor._running)
        self.assertFalse(monitor._hung_logged)

    def test_monitor_start(self):
        """Test starting the heartbeat monitor"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=30,
            check_interval=1,
        )

        result = monitor.start()

        # Verify monitor is running
        self.assertTrue(monitor._running)
        self.assertEqual(result, monitor)  # Verify chaining
        self.assertIsNotNone(monitor._thread)
        self.assertTrue(monitor._thread.is_alive())

        # Clean up
        monitor.stop()

    def test_monitor_is_running(self):
        """Test is_running method"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=30,
            check_interval=1,
        )

        # Not running initially
        self.assertFalse(monitor.is_running())

        # Running after start
        monitor.start()
        time.sleep(0.1)
        self.assertTrue(monitor.is_running())

        # Not running after stop
        monitor.stop()
        time.sleep(0.1)
        self.assertFalse(monitor.is_running())

    def test_monitor_heartbeat(self):
        """Test sending heartbeat updates"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=30,
            check_interval=1,
        )

        initial_time = monitor._last_heartbeat
        time.sleep(0.1)
        monitor.heartbeat()

        # Verify heartbeat time was updated
        self.assertGreater(monitor._last_heartbeat, initial_time)

    def test_monitor_hung_detection(self):
        """Test that monitor detects hung application"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=2,  # Short timeout for testing
            check_interval=1,
        )

        monitor.start()

        # Wait for timeout + check_interval
        time.sleep(3.5)

        # Verify hung event was logged
        self.mock_client.log_event.assert_called_once()
        call_args = self.mock_client.log_event.call_args
        self.assertEqual(call_args[0][0], self.session_id)
        self.assertEqual(call_args[0][1], "application_appears_hung")
        self.assertIn("seconds_since_heartbeat", call_args[0][2])
        self.assertIn("timeout", call_args[0][2])

        monitor.stop()

    def test_monitor_prevents_hung_with_regular_heartbeat(self):
        """Test that regular heartbeats prevent hung detection"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=2,
            check_interval=0.5,
        )

        monitor.start()

        # Send heartbeats regularly
        for _ in range(5):
            time.sleep(0.5)
            monitor.heartbeat()

        # Verify no hung event was logged
        self.mock_client.log_event.assert_not_called()

        monitor.stop()

    def test_monitor_recovery_after_hung(self):
        """Test that monitor detects recovery after hung state"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=1,
            check_interval=0.5,
        )

        monitor.start()

        # Wait for hung detection
        time.sleep(2)

        # Verify hung was logged
        self.assertTrue(self.mock_client.log_event.called)
        hung_call = self.mock_client.log_event.call_args
        self.assertEqual(hung_call[0][1], "application_appears_hung")

        # Send heartbeat to trigger recovery
        self.mock_client.reset_mock()
        monitor.heartbeat()
        time.sleep(0.2)

        # Verify recovery was logged
        recovery_calls = [
            call
            for call in self.mock_client.log_event.call_args_list
            if len(call[0]) > 1 and call[0][1] == "application_recovered"
        ]
        self.assertEqual(len(recovery_calls), 1)

        monitor.stop()

    def test_monitor_stop(self):
        """Test stopping the monitor"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=30,
            check_interval=1,
        )

        monitor.start()
        self.assertTrue(monitor._running)

        monitor.stop()
        time.sleep(0.1)

        # Verify monitor stopped
        self.assertFalse(monitor._running)
        if monitor._thread:
            self.assertFalse(monitor._thread.is_alive())

    def test_monitor_logging_failure_handling(self):
        """Test that monitor continues on logging failures"""
        # Make log_event raise an exception
        self.mock_client.log_event.side_effect = Exception("Network error")

        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=1,
            check_interval=0.5,
        )

        monitor.start()

        # Wait for hung detection
        time.sleep(2)

        # Monitor should still be running despite logging failure
        self.assertTrue(monitor.is_running())

        monitor.stop()

    def test_monitor_start_idempotent(self):
        """Test that calling start multiple times is safe"""
        monitor = HeartbeatMonitor(
            self.mock_client,
            self.session_id,
            timeout=30,
            check_interval=1,
        )

        result1 = monitor.start()
        result2 = monitor.start()

        # Both should return the monitor
        self.assertEqual(result1, monitor)
        self.assertEqual(result2, monitor)

        # Should still be running
        self.assertTrue(monitor.is_running())

        monitor.stop()


class TestSessionHeartbeatIntegration(unittest.TestCase):
    """Test Session class heartbeat integration"""

    def setUp(self):
        """Set up test components"""
        self.mock_client = Mock(spec=DatacatClient)
        self.mock_client.use_daemon = False
        self.session_id = "test-session-456"
        self.session = Session(self.mock_client, self.session_id)

    def test_start_heartbeat_monitor(self):
        """Test starting heartbeat monitor on session"""
        monitor = self.session.start_heartbeat_monitor(timeout=60, check_interval=5)

        # Verify monitor was created and started
        self.assertIsNotNone(self.session._heartbeat_monitor)
        self.assertEqual(monitor, self.session._heartbeat_monitor)
        self.assertTrue(monitor.is_running())

        # Clean up
        monitor.stop()

    def test_heartbeat_without_monitor(self):
        """Test that heartbeat raises error if monitor not started"""
        with self.assertRaises(Exception) as context:
            self.session.heartbeat()

        self.assertIn("not started", str(context.exception))
        self.assertIn("start_heartbeat_monitor", str(context.exception))

    def test_heartbeat_with_monitor(self):
        """Test sending heartbeat after monitor is started"""
        monitor = self.session.start_heartbeat_monitor(timeout=60, check_interval=5)

        # Should not raise
        self.session.heartbeat()

        # Clean up
        monitor.stop()

    def test_stop_heartbeat_monitor(self):
        """Test stopping heartbeat monitor"""
        monitor = self.session.start_heartbeat_monitor(timeout=60, check_interval=5)
        self.assertTrue(monitor.is_running())

        self.session.stop_heartbeat_monitor()

        # Monitor should be stopped and cleared
        self.assertIsNone(self.session._heartbeat_monitor)
        self.assertFalse(monitor.is_running())

    def test_session_end_stops_heartbeat_monitor(self):
        """Test that ending session stops heartbeat monitor"""
        self.session.start_heartbeat_monitor(timeout=60, check_interval=5)
        monitor = self.session._heartbeat_monitor

        self.assertTrue(monitor.is_running())

        # End session
        self.session.end()

        # Monitor should be stopped
        self.assertFalse(monitor.is_running())

    def test_session_end_without_monitor(self):
        """Test ending session without heartbeat monitor"""
        # Should not raise
        self.session.end()

        # Verify end_session was called
        self.mock_client.end_session.assert_called_once_with(self.session_id)

    def test_start_heartbeat_monitor_reuses_existing(self):
        """Test that starting monitor twice reuses the same monitor"""
        monitor1 = self.session.start_heartbeat_monitor(timeout=60, check_interval=5)
        monitor2 = self.session.start_heartbeat_monitor(timeout=30, check_interval=3)

        # Should be the same monitor instance
        self.assertEqual(monitor1, monitor2)

        # Clean up
        monitor1.stop()


class TestCreateSessionFactory(unittest.TestCase):
    """Test create_session factory function"""

    @patch("datacat.DatacatClient")
    def test_create_session(self, mock_client_class):
        """Test create_session creates session correctly"""
        # Set up mock
        mock_client = Mock()
        mock_client.register_session.return_value = "new-session-789"
        mock_client_class.return_value = mock_client

        # Create session
        session = create_session(
            "http://test.example.com:8080", product="TestProduct", version="1.0.0"
        )

        # Verify client was created with correct URL and default parameters
        mock_client_class.assert_called_once_with(
            "http://test.example.com:8080", daemon_port="auto"
        )

        # Verify session was registered with product and version
        mock_client.register_session.assert_called_once_with("TestProduct", "1.0.0")

        # Verify Session object has correct properties
        self.assertEqual(session.client, mock_client)
        self.assertEqual(session.session_id, "new-session-789")


class TestEdgeCases(unittest.TestCase):
    """Test edge cases and additional code paths"""

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_log_event_with_none_data(self, mock_daemon_class, mock_urlopen):
        """Test log_event with None data parameter"""
        # Set up daemon mock
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient("http://localhost:9090")

        # Call log_event with None data (should use empty dict)
        result = client.log_event("session-123", "test_event", data=None)

        # Verify the request was made with empty data dict
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["data"], {})

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_log_event_without_data_parameter(self, mock_daemon_class, mock_urlopen):
        """Test log_event called without data parameter"""
        # Set up daemon mock
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient("http://localhost:9090")

        # Call log_event without data parameter
        result = client.log_event("session-123", "test_event")

        # Verify the request was made with empty data dict
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["data"], {})

    def test_monitor_recovery_logging_exception_handling(self):
        """Test that recovery logging exception is silently handled"""
        mock_client = Mock(spec=DatacatClient)
        mock_client.use_daemon = False

        # Make log_event fail only during recovery logging
        call_count = [0]

        def log_event_side_effect(*args, **kwargs):
            call_count[0] += 1
            if call_count[0] > 1 and args[1] == "application_recovered":
                raise Exception("Network error during recovery logging")
            return {"status": "ok"}

        mock_client.log_event.side_effect = log_event_side_effect

        monitor = HeartbeatMonitor(
            mock_client,
            "test-session",
            timeout=1,
            check_interval=0.5,
        )

        monitor.start()

        # Wait for hung detection
        time.sleep(2)

        # Manually set hung flag to test recovery path
        with monitor._lock:
            monitor._hung_logged = True

        # Send heartbeat - should trigger recovery logging which will fail
        # This should not raise an exception
        monitor.heartbeat()

        # Wait a bit to ensure no exception was raised
        time.sleep(0.5)

        # Monitor should still be running
        self.assertTrue(monitor.is_running())

        monitor.stop()


class TestDaemonManager(unittest.TestCase):
    """Test DaemonManager functionality"""

    def test_daemon_manager_initialization(self):
        """Test DaemonManager initialization with defaults"""
        manager = DaemonManager()
        self.assertEqual(manager.daemon_port, "auto")  # Default changed to "auto"
        self.assertEqual(manager.server_url, "http://localhost:9090")
        self.assertIsNotNone(manager.daemon_binary)
        self.assertIsNone(manager.process)
        self.assertFalse(manager._started)

    def test_daemon_manager_initialization_with_params(self):
        """Test DaemonManager initialization with custom parameters"""
        manager = DaemonManager(
            daemon_port="9999",
            server_url="http://example.com:8080",
            daemon_binary="/custom/path/daemon",
        )
        self.assertEqual(manager.daemon_port, "9999")
        self.assertEqual(manager.server_url, "http://example.com:8080")
        self.assertEqual(manager.daemon_binary, "/custom/path/daemon")

    def test_find_daemon_binary_returns_default(self):
        """Test that _find_daemon_binary returns a default value"""
        manager = DaemonManager()
        binary = manager._find_daemon_binary()
        # Should return some path (could be default or found path)
        self.assertIsInstance(binary, str)
        self.assertTrue(len(binary) > 0)

    @patch("datacat.subprocess.call")
    def test_is_in_path_unix(self, mock_call):
        """Test _is_in_path on Unix systems"""
        manager = DaemonManager()

        # Mock successful result (binary in PATH)
        mock_call.return_value = 0
        with patch("sys.platform", "linux"):
            result = manager._is_in_path("test-binary")
        self.assertTrue(result)

        # Verify 'which' was called on Unix
        mock_call.assert_called_with(
            ["which", "test-binary"], stdout=subprocess.PIPE, stderr=subprocess.PIPE
        )

    @patch("datacat.subprocess.call")
    def test_is_in_path_windows(self, mock_call):
        """Test _is_in_path on Windows systems"""
        manager = DaemonManager()

        # Mock successful result (binary in PATH)
        mock_call.return_value = 0
        with patch("sys.platform", "win32"):
            result = manager._is_in_path("test-binary.exe")
        self.assertTrue(result)

        # Verify 'where' was called on Windows
        mock_call.assert_called_with(
            ["where", "test-binary.exe"], stdout=subprocess.PIPE, stderr=subprocess.PIPE
        )

    @patch("datacat.subprocess.call")
    def test_is_in_path_not_found(self, mock_call):
        """Test _is_in_path when binary is not in PATH"""
        manager = DaemonManager()

        # Mock unsuccessful result (binary not in PATH)
        mock_call.return_value = 1
        result = manager._is_in_path("nonexistent-binary")
        self.assertFalse(result)

    @patch("datacat.subprocess.call")
    def test_is_in_path_exception_handling(self, mock_call):
        """Test _is_in_path handles exceptions gracefully"""
        manager = DaemonManager()

        # Mock exception
        mock_call.side_effect = Exception("Command failed")
        result = manager._is_in_path("test-binary")
        self.assertFalse(result)

    @patch("datacat.os.path.exists")
    def test_find_daemon_binary_checks_paths(self, mock_exists):
        """Test that _find_daemon_binary checks various paths"""
        manager = DaemonManager()

        # Mock that binary exists in second path
        def exists_side_effect(path):
            return path == "./datacat-daemon"

        mock_exists.side_effect = exists_side_effect

        binary = manager._find_daemon_binary()
        self.assertEqual(binary, "./datacat-daemon")

    @patch("datacat.os.path.exists")
    @patch("datacat.subprocess.call")
    def test_find_daemon_binary_returns_default_when_not_found(
        self, mock_call, mock_exists
    ):
        """Test that _find_daemon_binary returns default when not found anywhere"""
        manager = DaemonManager()

        # Mock that binary doesn't exist anywhere
        mock_exists.return_value = False
        mock_call.return_value = 1  # Not in PATH

        binary = manager._find_daemon_binary()
        self.assertEqual(binary, "datacat-daemon")

    def test_is_running_when_not_started(self):
        """Test is_running returns False when not started"""
        manager = DaemonManager()
        self.assertFalse(manager.is_running())

    @patch("datacat.subprocess.Popen")
    @patch("datacat.subprocess.call")
    @patch("datacat.time.sleep")
    @patch("datacat.atexit.register")
    @patch("builtins.open", new_callable=MagicMock)
    def test_daemon_start_success(
        self, mock_open, mock_atexit, mock_sleep, mock_call, mock_popen
    ):
        """Test successful daemon start"""
        # Mock that binary is not in PATH (so it uses default)
        mock_call.return_value = 1

        # Mock process that stays alive
        mock_process = Mock()
        mock_process.poll.return_value = None  # Process is running
        mock_popen.return_value = mock_process

        manager = DaemonManager()
        manager.start()

        # Verify daemon was started
        self.assertTrue(manager._started)
        self.assertEqual(manager.process, mock_process)
        # Popen should be called for starting the daemon
        self.assertTrue(mock_popen.called)
        mock_atexit.assert_called_once()

    @patch("datacat.subprocess.Popen")
    @patch("datacat.subprocess.call")
    @patch("datacat.time.sleep")
    @patch("builtins.open", new_callable=MagicMock)
    def test_daemon_start_failure(self, mock_open, mock_sleep, mock_call, mock_popen):
        """Test daemon start when process fails immediately"""
        # Mock that binary is not in PATH
        mock_call.return_value = 1

        # Mock process that exits immediately
        mock_process = Mock()
        mock_process.poll.return_value = 1  # Process exited
        mock_popen.return_value = mock_process

        manager = DaemonManager()

        with self.assertRaises(Exception) as context:
            manager.start()

        self.assertIn("failed to start", str(context.exception).lower())

    @patch("datacat.subprocess.Popen")
    @patch("builtins.open", new_callable=MagicMock)
    def test_daemon_start_binary_not_found(self, mock_open, mock_popen):
        """Test daemon start when binary is not found"""
        # Mock OSError (binary not found)
        mock_popen.side_effect = OSError("No such file or directory")

        manager = DaemonManager(daemon_binary="/nonexistent/daemon")

        with self.assertRaises(Exception) as context:
            manager.start()

        self.assertIn("Failed to start daemon binary", str(context.exception))
        self.assertIn("/nonexistent/daemon", str(context.exception))

    @patch("datacat.subprocess.Popen")
    @patch("datacat.subprocess.call")
    @patch("datacat.time.sleep")
    @patch("datacat.atexit.register")
    @patch("builtins.open", new_callable=MagicMock)
    def test_daemon_start_idempotent(
        self, mock_open, mock_atexit, mock_sleep, mock_call, mock_popen
    ):
        """Test that calling start multiple times doesn't start multiple daemons"""
        # Mock that binary is not in PATH
        mock_call.return_value = 1

        # Mock process that stays alive
        mock_process = Mock()
        mock_process.poll.return_value = None  # Process is running
        mock_popen.return_value = mock_process

        manager = DaemonManager()
        initial_popen_call_count = mock_popen.call_count
        manager.start()
        after_first_start = mock_popen.call_count

        # Call start again
        manager.start()
        after_second_start = mock_popen.call_count

        # Should not call Popen again for the second start
        self.assertEqual(after_first_start, after_second_start)

    @patch("datacat.subprocess.Popen")
    @patch("datacat.subprocess.call")
    @patch("datacat.time.sleep")
    @patch("datacat.atexit.register")
    @patch("builtins.open", new_callable=MagicMock)
    def test_daemon_stop(
        self, mock_open, mock_atexit, mock_sleep, mock_call, mock_popen
    ):
        """Test stopping the daemon"""
        # Mock that binary is not in PATH
        mock_call.return_value = 1

        # Mock process that stays alive
        mock_process = Mock()
        mock_process.poll.return_value = None  # Process is running
        mock_process.wait.return_value = None
        mock_popen.return_value = mock_process

        manager = DaemonManager()
        manager.start()
        manager.stop()

        # Verify daemon was stopped
        self.assertFalse(manager._started)
        mock_process.terminate.assert_called_once()
        mock_process.wait.assert_called_once()

    @patch("datacat.subprocess.Popen")
    @patch("datacat.subprocess.call")
    @patch("datacat.time.sleep")
    @patch("datacat.atexit.register")
    @patch("builtins.open", new_callable=MagicMock)
    def test_daemon_stop_with_kill(
        self, mock_open, mock_atexit, mock_sleep, mock_call, mock_popen
    ):
        """Test stopping daemon when terminate times out"""
        # Mock that binary is not in PATH
        mock_call.return_value = 1

        # Mock process that stays alive
        mock_process = Mock()
        mock_process.poll.return_value = None  # Process is running
        mock_process.wait.side_effect = Exception("Timeout")  # Simulate timeout
        mock_popen.return_value = mock_process

        manager = DaemonManager()
        manager.start()
        manager.stop()

        # Verify daemon was terminated and killed
        mock_process.terminate.assert_called_once()
        mock_process.kill.assert_called_once()
        self.assertFalse(manager._started)

    def test_daemon_stop_when_not_running(self):
        """Test stopping daemon when it's not running"""
        manager = DaemonManager()
        # Should not raise
        manager.stop()
        self.assertFalse(manager._started)


class TestDatacatClientWithDaemon(unittest.TestCase):
    """Test DatacatClient with daemon enabled"""

    @patch("datacat.DaemonManager")
    def test_client_with_daemon_initialization(self, mock_daemon_class):
        """Test client initialization with daemon enabled"""
        mock_daemon = Mock()
        mock_daemon.daemon_port = "9999"  # Set the daemon port on the mock
        mock_daemon_class.return_value = mock_daemon

        client = DatacatClient(base_url="http://example.com:8080", daemon_port="9999")

        # Verify daemon was created and started
        mock_daemon_class.assert_called_once_with(
            daemon_port="9999", server_url="http://example.com:8080"
        )
        mock_daemon.start.assert_called_once()

        # Verify base_url points to daemon
        self.assertEqual(client.base_url, "http://localhost:9999")
        self.assertTrue(client.use_daemon)

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_register_session_with_daemon(self, mock_daemon_class, mock_urlopen):
        """Test register_session with daemon sends parent PID"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"session_id": "test-123"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()
        session_id = client.register_session("TestProduct", "1.0.0")

        # Verify correct endpoint was called
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertTrue(request.get_full_url().endswith("/register"))

        # Verify parent PID, product, and version were sent
        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertIn("parent_pid", sent_data)
        self.assertEqual(sent_data["product"], "TestProduct")
        self.assertEqual(sent_data["version"], "1.0.0")
        self.assertEqual(sent_data["parent_pid"], os.getpid())

        self.assertEqual(session_id, "test-123")

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_update_state_with_daemon(self, mock_daemon_class, mock_urlopen):
        """Test update_state with daemon uses correct format"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()
        client.update_state("session-123", {"key": "value"})

        # Verify correct endpoint and format
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertTrue(request.get_full_url().endswith("/state"))

        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["session_id"], "session-123")
        self.assertEqual(sent_data["state"], {"key": "value"})

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_log_event_with_daemon(self, mock_daemon_class, mock_urlopen):
        """Test log_event with daemon uses correct format"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()
        client.log_event("session-123", "test_event", data={"data": "value"})

        # Verify correct endpoint and format
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertTrue(request.get_full_url().endswith("/event"))

        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["session_id"], "session-123")
        self.assertEqual(sent_data["name"], "test_event")
        self.assertEqual(sent_data["data"], {"data": "value"})

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_log_metric_with_daemon(self, mock_daemon_class, mock_urlopen):
        """Test log_metric with daemon uses correct format"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()
        client.log_metric("session-123", "cpu_usage", 45.5, ["tag1", "tag2"])

        # Verify correct endpoint and format
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertTrue(request.get_full_url().endswith("/metric"))

        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["session_id"], "session-123")
        self.assertEqual(sent_data["name"], "cpu_usage")
        self.assertEqual(sent_data["value"], 45.5)
        self.assertEqual(sent_data["tags"], ["tag1", "tag2"])

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_end_session_with_daemon(self, mock_daemon_class, mock_urlopen):
        """Test end_session with daemon uses correct format"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()
        client.end_session("session-123")

        # Verify correct endpoint and format
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertTrue(request.get_full_url().endswith("/end"))

        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["session_id"], "session-123")


class TestHeartbeatMonitorWithDaemon(unittest.TestCase):
    """Test HeartbeatMonitor with daemon enabled"""

    @patch("datacat.DaemonManager")
    def test_monitor_loop_with_daemon(self, mock_daemon_class):
        """Test that monitor loop skips hung detection when using daemon"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_client = DatacatClient()
        mock_client.log_event = Mock()

        monitor = HeartbeatMonitor(
            mock_client,
            "test-session",
            timeout=1,
            check_interval=0.5,
        )

        monitor.start()

        # Wait for timeout to pass
        time.sleep(2)

        # Verify no hung event was logged (daemon handles it)
        mock_client.log_event.assert_not_called()

        monitor.stop()

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_heartbeat_with_daemon(self, mock_daemon_class, mock_urlopen):
        """Test that heartbeat sends to daemon when using daemon"""
        mock_daemon = Mock()
        mock_daemon.daemon_port = "9999"  # Set the daemon port on the mock
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient(daemon_port="9999")
        monitor = HeartbeatMonitor(client, "test-session")

        monitor.heartbeat()

        # Verify heartbeat was sent to daemon
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertTrue(request.get_full_url().endswith("/heartbeat"))
        self.assertIn("localhost:9999", request.get_full_url())

        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["session_id"], "test-session")

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_heartbeat_with_daemon_handles_failure(
        self, mock_daemon_class, mock_urlopen
    ):
        """Test that heartbeat failure with daemon is handled gracefully"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        # Mock network failure
        mock_urlopen.side_effect = Exception("Network error")

        client = DatacatClient()
        monitor = HeartbeatMonitor(client, "test-session")

        # Should not raise exception
        monitor.heartbeat()


class TestSessionConvenienceMethods(unittest.TestCase):
    """Test Session convenience methods"""

    def test_session_update_state(self):
        """Test Session.update_state wrapper"""
        mock_client = Mock()
        mock_client.update_state.return_value = {"status": "ok"}

        session = Session(mock_client, "session-123")
        result = session.update_state({"key": "value"})

        mock_client.update_state.assert_called_once_with(
            "session-123", {"key": "value"}
        )
        self.assertEqual(result, {"status": "ok"})

    def test_session_log_event(self):
        """Test Session.log_event wrapper"""
        mock_client = Mock()
        mock_client.log_event.return_value = {"status": "ok"}

        session = Session(mock_client, "session-123")
        result = session.log_event("test_event", data={"data": "value"})

        # Session.log_event now passes category, group, labels, stacktrace
        mock_client.log_event.assert_called_once_with(
            "session-123",
            "test_event",
            category=None,
            group=None,
            labels=[],  # Session normalizes None to []
            message=None,
            data={"data": "value"},
            stacktrace=None,
        )
        self.assertEqual(result, {"status": "ok"})

    def test_session_log_metric(self):
        """Test Session.log_metric wrapper"""
        mock_client = Mock()
        mock_client.log_metric.return_value = {"status": "ok"}

        session = Session(mock_client, "session-123")
        result = session.log_metric("cpu_usage", 45.5, ["tag1"])

        # Session.log_metric now passes additional parameters with defaults
        mock_client.log_metric.assert_called_once_with(
            "session-123", "cpu_usage", 45.5, ["tag1"], "gauge", None, None, None, None
        )
        self.assertEqual(result, {"status": "ok"})

    def test_session_get_details(self):
        """Test Session.get_details wrapper"""
        mock_client = Mock()
        mock_client.get_session.return_value = {"id": "session-123", "state": {}}

        session = Session(mock_client, "session-123")
        result = session.get_details()

        mock_client.get_session.assert_called_once_with("session-123")
        self.assertEqual(result, {"id": "session-123", "state": {}})


class TestProductVersionValidation(unittest.TestCase):
    """Test that product and version are required for session creation"""

    @patch("datacat.DaemonManager")
    def test_register_session_requires_product(self, mock_daemon):
        """Test that register_session requires product parameter"""
        mock_daemon.return_value.start = Mock()
        client = DatacatClient("http://localhost:9090")
        with self.assertRaises(Exception) as context:
            client.register_session(None, "1.0.0")
        self.assertIn("Product and version are required", str(context.exception))

    @patch("datacat.DaemonManager")
    def test_register_session_requires_version(self, mock_daemon):
        """Test that register_session requires version parameter"""
        mock_daemon.return_value.start = Mock()
        client = DatacatClient("http://localhost:9090")
        with self.assertRaises(Exception) as context:
            client.register_session("TestProduct", None)
        self.assertIn("Product and version are required", str(context.exception))

    @patch("datacat.DaemonManager")
    def test_register_session_rejects_empty_product(self, mock_daemon):
        """Test that register_session rejects empty product string"""
        mock_daemon.return_value.start = Mock()
        client = DatacatClient("http://localhost:9090")
        with self.assertRaises(Exception) as context:
            client.register_session("", "1.0.0")
        self.assertIn("Product and version are required", str(context.exception))

    @patch("datacat.DaemonManager")
    def test_register_session_rejects_empty_version(self, mock_daemon):
        """Test that register_session rejects empty version string"""
        mock_daemon.return_value.start = Mock()
        client = DatacatClient("http://localhost:9090")
        with self.assertRaises(Exception) as context:
            client.register_session("TestProduct", "")
        self.assertIn("Product and version are required", str(context.exception))

    def test_create_session_requires_product(self):
        """Test that create_session factory requires product parameter"""
        with self.assertRaises(Exception) as context:
            create_session("http://localhost:9090", product=None, version="1.0.0")
        self.assertIn("Product and version are required", str(context.exception))

    def test_create_session_requires_version(self):
        """Test that create_session factory requires version parameter"""
        with self.assertRaises(Exception) as context:
            create_session("http://localhost:9090", product="TestProduct", version=None)
        self.assertIn("Product and version are required", str(context.exception))

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_log_exception_with_all_fields(self, mock_daemon_class, mock_urlopen):
        """Test log_exception sends all exception-specific fields"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()

        # Create a fake exception
        try:
            raise ValueError("test error")
        except ValueError:
            exc_info = sys.exc_info()
            client.log_exception("session-123", exc_info=exc_info)

        # Verify correct endpoint and format
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertTrue(request.get_full_url().endswith("/event"))

        sent_data = json.loads(request.data.decode("utf-8"))
        self.assertEqual(sent_data["session_id"], "session-123")
        self.assertEqual(sent_data["name"], "exception")
        self.assertEqual(sent_data["category"], "error")  # Changed from level to category
        self.assertIn("exception", sent_data["labels"])
        self.assertIn("ValueError", sent_data["labels"])
        self.assertEqual(sent_data["exception_type"], "ValueError")
        self.assertEqual(sent_data["exception_msg"], "test error")
        self.assertIsInstance(sent_data["stacktrace"], list)
        self.assertTrue(len(sent_data["stacktrace"]) > 0)
        self.assertIsNotNone(sent_data["source_file"])
        self.assertIsNotNone(sent_data["source_line"])
        self.assertIsNotNone(sent_data["source_function"])

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_log_event_with_all_optional_fields(self, mock_daemon_class, mock_urlopen):
        """Test log_event with category, group, labels, and message"""
        mock_daemon = Mock()
        mock_daemon.daemon_port = "8079"  # Set daemon port for mock
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()
        client.log_event(
            "session-123",
            "custom_event",
            category="warning",  # Changed from level to category
            group="my.component",  # Changed from category to group
            labels=["tag1", "tag2"],
            message="This is a warning",
            data={"key": "value"},
        )

        # Verify all fields were sent
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        sent_data = json.loads(request.data.decode("utf-8"))

        self.assertEqual(sent_data["name"], "custom_event")
        self.assertEqual(sent_data["category"], "warning")  # Changed from level
        self.assertEqual(sent_data["group"], "my.component")  # Now group instead of category
        self.assertEqual(sent_data["labels"], ["tag1", "tag2"])
        self.assertEqual(sent_data["message"], "This is a warning")
        self.assertEqual(sent_data["data"], {"key": "value"})

    @patch("datacat.urlopen")
    @patch("datacat.DaemonManager")
    def test_update_state_with_null_deletes_keys(self, mock_daemon_class, mock_urlopen):
        """Test that passing None as a value deletes state keys"""
        mock_daemon = Mock()
        mock_daemon_class.return_value = mock_daemon

        mock_response = Mock()
        mock_response.read.return_value = b'{"status": "ok"}'
        mock_urlopen.return_value = mock_response

        client = DatacatClient()

        # Update state with None value to delete key
        client.update_state("session-123", {"user": None, "count": 10})

        # Verify None was sent (will be serialized as null in JSON)
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        sent_data = json.loads(request.data.decode("utf-8"))

        self.assertIsNone(sent_data["state"]["user"])
        self.assertEqual(sent_data["state"]["count"], 10)


if __name__ == "__main__":
    unittest.main()
