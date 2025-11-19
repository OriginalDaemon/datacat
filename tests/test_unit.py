"""
Unit tests for datacat Python client

These tests focus on testing individual components in isolation,
including error handling, HeartbeatMonitor, and edge cases.
"""

import json
import os
import sys
import threading
import time
import unittest
from unittest.mock import Mock, patch, MagicMock

# Add python directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import DatacatClient, HeartbeatMonitor, Session, create_session

try:
    from urllib.error import HTTPError, URLError
except ImportError:
    from urllib2 import HTTPError, URLError


class TestDatacatClientErrorHandling(unittest.TestCase):
    """Test error handling in DatacatClient"""

    def setUp(self):
        """Set up test client"""
        self.client = DatacatClient("http://localhost:8080")

    @patch("datacat.urlopen")
    def test_http_error_handling(self, mock_urlopen):
        """Test handling of HTTP errors"""
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
            self.client.register_session()

        self.assertIn("HTTP Error 404", str(context.exception))
        self.assertIn("Not Found", str(context.exception))

    @patch("datacat.urlopen")
    def test_url_error_handling(self, mock_urlopen):
        """Test handling of URL errors (network issues)"""
        # Create a mock URLError
        mock_error = URLError("Connection refused")
        mock_urlopen.side_effect = mock_error

        # Verify exception is raised with proper message
        with self.assertRaises(Exception) as context:
            self.client.register_session()

        self.assertIn("URL Error", str(context.exception))
        self.assertIn("Connection refused", str(context.exception))

    @patch("datacat.urlopen")
    def test_general_exception_handling(self, mock_urlopen):
        """Test handling of general exceptions"""
        # Create a general exception
        mock_urlopen.side_effect = RuntimeError("Unexpected error")

        # Verify exception is raised with proper message
        with self.assertRaises(Exception) as context:
            self.client.register_session()

        self.assertIn("Request failed", str(context.exception))
        self.assertIn("Unexpected error", str(context.exception))

    def test_base_url_trailing_slash_removal(self):
        """Test that trailing slash is removed from base URL"""
        client = DatacatClient("http://localhost:8080/")
        self.assertEqual(client.base_url, "http://localhost:8080")

        client2 = DatacatClient("http://localhost:8080")
        self.assertEqual(client2.base_url, "http://localhost:8080")


class TestDatacatClientGetAllSessions(unittest.TestCase):
    """Test get_all_sessions method"""

    @patch("datacat.urlopen")
    def test_get_all_sessions(self, mock_urlopen):
        """Test get_all_sessions calls the correct endpoint"""
        # Mock response
        mock_response = Mock()
        mock_response.read.return_value = b'[{"id": "session1"}, {"id": "session2"}]'
        mock_urlopen.return_value = mock_response

        client = DatacatClient("http://localhost:8080")
        sessions = client.get_all_sessions()

        # Verify correct endpoint was called
        call_args = mock_urlopen.call_args
        request = call_args[0][0]
        self.assertEqual(
            request.get_full_url(), "http://localhost:8080/api/grafana/sessions"
        )

        # Verify response parsing
        self.assertEqual(len(sessions), 2)
        self.assertEqual(sessions[0]["id"], "session1")
        self.assertEqual(sessions[1]["id"], "session2")


class TestHeartbeatMonitor(unittest.TestCase):
    """Test HeartbeatMonitor functionality"""

    def setUp(self):
        """Set up test components"""
        self.mock_client = Mock(spec=DatacatClient)
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
        session = create_session("http://test.example.com:8080")

        # Verify client was created with correct URL
        mock_client_class.assert_called_once_with("http://test.example.com:8080")

        # Verify session was registered
        mock_client.register_session.assert_called_once()

        # Verify Session object has correct properties
        self.assertEqual(session.client, mock_client)
        self.assertEqual(session.session_id, "new-session-789")


class TestEdgeCases(unittest.TestCase):
    """Test edge cases and additional code paths"""

    def test_log_event_with_none_data(self):
        """Test log_event with None data parameter"""
        with patch("datacat.urlopen") as mock_urlopen:
            mock_response = Mock()
            mock_response.read.return_value = b'{"status": "ok"}'
            mock_urlopen.return_value = mock_response

            client = DatacatClient("http://localhost:8080")

            # Call log_event with None data (should use empty dict)
            result = client.log_event("session-123", "test_event", None)

            # Verify the request was made with empty data dict
            call_args = mock_urlopen.call_args
            request = call_args[0][0]
            sent_data = json.loads(request.data.decode("utf-8"))
            self.assertEqual(sent_data["data"], {})

    def test_log_event_without_data_parameter(self):
        """Test log_event called without data parameter"""
        with patch("datacat.urlopen") as mock_urlopen:
            mock_response = Mock()
            mock_response.read.return_value = b'{"status": "ok"}'
            mock_urlopen.return_value = mock_response

            client = DatacatClient("http://localhost:8080")

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


if __name__ == "__main__":
    unittest.main()
