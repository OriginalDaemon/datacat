"""
datacat - Python client for the datacat REST API

This module provides a simple interface to interact with the datacat service
for logging application data, events, and metrics.

When using the client with a daemon (recommended), it automatically starts
a local daemon subprocess that batches and optimizes network traffic.

Compatible with Python 2.7+ and Python 3.x
"""

from __future__ import print_function
import atexit
import json
import os
import subprocess
import sys
import threading
import time
import traceback

try:
    # Python 3
    from urllib.request import urlopen, Request  # type: ignore
    from urllib.error import URLError, HTTPError  # type: ignore
except ImportError:
    # Python 2
    from urllib2 import urlopen, Request, URLError, HTTPError  # type: ignore


class DaemonManager(object):
    """Manages the local datacat daemon subprocess"""

    def __init__(
        self, daemon_port="8079", server_url="http://localhost:9090", daemon_binary=None
    ):
        """
        Initialize the daemon manager

        Args:
            daemon_port (str): Port for the daemon to listen on
            server_url (str): URL of the datacat server
            daemon_binary (str): Path to the daemon binary (auto-detected if None)
        """
        self.daemon_port = daemon_port
        self.server_url = server_url
        self.daemon_binary = daemon_binary or self._find_daemon_binary()
        self.process = None
        self._started = False

    def _find_daemon_binary(self):
        """Find the daemon binary in common locations"""
        # Determine binary name based on platform
        binary_name = (
            "datacat-daemon.exe" if sys.platform == "win32" else "datacat-daemon"
        )

        # Check common locations
        possible_paths = [
            binary_name,  # In PATH
            "./" + binary_name,  # Current directory
            "./cmd/datacat-daemon/" + binary_name,  # Development
            os.path.join(
                os.path.dirname(__file__),
                "..",
                "cmd",
                "datacat-daemon",
                binary_name,
            ),
            "./bin/" + binary_name,  # Built binaries
            os.path.join(
                os.path.dirname(__file__),
                "..",
                "bin",
                binary_name,
            ),
        ]

        for path in possible_paths:
            if os.path.exists(path) or self._is_in_path(path):
                return path

        # Return default and let it fail if not found
        return binary_name

    def _is_in_path(self, binary):
        """Check if binary exists in PATH"""
        try:
            # Use 'which' on Unix or 'where' on Windows
            cmd = "where" if sys.platform == "win32" else "which"
            result = subprocess.call(
                [cmd, binary], stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )
            return result == 0
        except Exception:
            return False

    def start(self):
        """Start the daemon subprocess"""
        if self._started and self.process and self.process.poll() is None:
            return  # Already running

        # Create config for daemon
        config = {
            "daemon_port": self.daemon_port,
            "server_url": self.server_url,
            "batch_interval_seconds": 5,
            "max_batch_size": 100,
            "heartbeat_timeout_seconds": 60,
        }

        # Write config to temporary file
        config_path = "daemon_config.json"
        with open(config_path, "w") as f:
            json.dump(config, f, indent=2)

        # Start daemon process
        try:
            self.process = subprocess.Popen(
                [self.daemon_binary],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                stdin=subprocess.PIPE,
            )
            self._started = True

            # Wait a bit for daemon to start
            time.sleep(1)

            # Check if daemon is running
            if self.process.poll() is not None:
                raise Exception("Daemon failed to start")

            # Register cleanup on exit
            atexit.register(self.stop)

        except OSError as e:
            raise Exception(
                "Failed to start daemon binary '{}': {}".format(
                    self.daemon_binary, str(e)
                )
            )

    def stop(self):
        """Stop the daemon subprocess"""
        if self.process and self.process.poll() is None:
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except Exception:
                self.process.kill()
        self._started = False

    def is_running(self):
        """Check if daemon is running"""
        return self._started and self.process and self.process.poll() is None


class DatacatClient(object):
    """Client for interacting with the datacat daemon (daemon mode only)"""

    def __init__(
        self, base_url="http://localhost:9090", daemon_port="8079"
    ):
        """
        Initialize the datacat client

        Args:
            base_url (str): Base URL of the datacat server (used as daemon's upstream)
            daemon_port (str): Port for the local daemon
        """
        self.use_daemon = True  # Always use daemon
        self.daemon_manager = DaemonManager(
            daemon_port=daemon_port, server_url=base_url
        )
        self.daemon_manager.start()
        # Point to daemon instead of server
        self.base_url = "http://localhost:{}".format(daemon_port)

    def _make_request(self, url, method="GET", data=None):
        """
        Make an HTTP request to the datacat API

        Args:
            url (str): Full URL to request
            method (str): HTTP method (GET or POST)
            data (dict): Optional data to send as JSON

        Returns:
            dict: Response data as dictionary

        Raises:
            Exception: If the request fails
        """
        headers = {"Content-Type": "application/json"}

        if data is not None:
            data = json.dumps(data).encode("utf-8")

        request = Request(url, data=data, headers=headers)
        if method != "GET":
            request.get_method = lambda: method  # type: ignore[method-assign]

        try:
            response = urlopen(request)
            response_data = response.read().decode("utf-8")
            return json.loads(response_data)
        except HTTPError as e:
            error_msg = "HTTP Error {0}: {1}".format(e.code, e.reason)
            raise Exception(error_msg)
        except URLError as e:
            error_msg = "URL Error: {0}".format(e.reason)
            raise Exception(error_msg)
        except Exception as e:
            error_msg = "Request failed: {0}".format(str(e))
            raise Exception(error_msg)

    def register_session(self, product, version):
        """
        Register a new session with the datacat service or daemon

        Args:
            product (str): Product name (required)
            version (str): Product version (required)

        Returns:
            str: Session ID

        Raises:
            Exception: If registration fails or if product/version are empty
        """
        if not product or not version:
            raise Exception("Product and version are required to create a session")

        url = "{0}/register".format(self.base_url)
        # Send parent PID so daemon can monitor for crashes
        data = {"parent_pid": os.getpid(), "product": product, "version": version}
        response = self._make_request(url, method="POST", data=data)
        return response.get("session_id")

    def get_session(self, session_id):
        """
        Get details of a specific session

        Args:
            session_id (str): Session ID

        Returns:
            dict: Session details

        Raises:
            Exception: If the request fails
        """
        url = "{0}/session?session_id={1}".format(self.base_url, session_id)
        return self._make_request(url, method="GET")

    def update_state(self, session_id, state):
        """
        Update the state of a session

        Args:
            session_id (str): Session ID
            state (dict): State data to update

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails
        """
        url = "{0}/state".format(self.base_url)
        data = {"session_id": session_id, "state": state}
        return self._make_request(url, method="POST", data=data)

    def log_event(self, session_id, name, data=None):
        """
        Log an event to a session

        Args:
            session_id (str): Session ID
            name (str): Event name
            data (dict): Optional event data

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails
        """
        if data is None:
            data = {}

        url = "{0}/event".format(self.base_url)
        request_data = {"session_id": session_id, "name": name, "data": data}
        return self._make_request(url, method="POST", data=request_data)

    def log_metric(self, session_id, name, value, tags=None):
        """
        Log a metric to a session

        Args:
            session_id (str): Session ID
            name (str): Metric name
            value (float): Metric value
            tags (list): Optional list of tags

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails
        """
        url = "{0}/metric".format(self.base_url)
        metric_data = {
            "session_id": session_id,
            "name": name,
            "value": float(value),
            "tags": tags or [],
        }
        return self._make_request(url, method="POST", data=metric_data)

    def end_session(self, session_id):
        """
        End a session

        Args:
            session_id (str): Session ID

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails
        """
        url = "{0}/end".format(self.base_url)
        return self._make_request(
            url, method="POST", data={"session_id": session_id}
        )

    def log_exception(self, session_id, exc_info=None, extra_data=None):
        """
        Log an exception to a session

        Args:
            session_id (str): Session ID
            exc_info (tuple): Exception info from sys.exc_info(), if None uses current exception
            extra_data (dict): Optional additional data to include

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails
        """
        if exc_info is None:
            exc_info = sys.exc_info()

        exc_type, exc_value, exc_traceback = exc_info

        exception_data = {
            "type": exc_type.__name__ if exc_type else "Unknown",
            "message": str(exc_value) if exc_value else "",
            "traceback": traceback.format_exception(exc_type, exc_value, exc_traceback),
        }

        if extra_data:
            exception_data.update(extra_data)

        return self.log_event(session_id, "exception", exception_data)

    def get_all_sessions(self):
        """
        Get all sessions

        Returns:
            list: List of all sessions

        Raises:
            Exception: If the request fails
        """
        url = "{0}/sessions".format(self.base_url)
        return self._make_request(url, method="GET")

    def pause_heartbeat_monitoring(self, session_id):
        """
        Tell the daemon to pause heartbeat monitoring for this session

        This is useful when the application will be intentionally unresponsive
        (e.g., long-running computation, blocking I/O, etc.)

        Args:
            session_id (str): Session ID

        Returns:
            dict: Response from the daemon

        Raises:
            Exception: If the request fails
        """
        url = "{0}/pause_heartbeat".format(self.base_url)
        return self._make_request(
            url, method="POST", data={"session_id": session_id}
        )

    def resume_heartbeat_monitoring(self, session_id):
        """
        Tell the daemon to resume heartbeat monitoring for this session

        After calling this, the application should resume sending regular heartbeats.

        Args:
            session_id (str): Session ID

        Returns:
            dict: Response from the daemon

        Raises:
            Exception: If the request fails
        """
        url = "{0}/resume_heartbeat".format(self.base_url)
        return self._make_request(
            url, method="POST", data={"session_id": session_id}
        )


class HeartbeatMonitor(object):
    """
    Hardware thread-based heartbeat monitor for detecting application hangs.

    This class runs a separate thread that monitors heartbeats from the application.
    If no heartbeat is received within the timeout period, it logs an event indicating
    the application appears to be hung.
    """

    def __init__(self, client, session_id, timeout=60, check_interval=5):
        """
        Initialize the heartbeat monitor

        Args:
            client (DatacatClient): Client instance for logging events
            session_id (str): Session ID to monitor
            timeout (int): Seconds without heartbeat before logging hung event (default: 60)
            check_interval (int): Seconds between heartbeat checks (default: 5)
        """
        self.client = client
        self.session_id = session_id
        self.timeout = timeout
        self.check_interval = check_interval
        self._last_heartbeat = time.time()
        self._running = False
        self._thread = None
        self._lock = threading.Lock()
        self._hung_logged = False

    def _monitor_loop(self):
        """Internal monitor loop that runs in the background thread"""
        while self._running:
            time.sleep(self.check_interval)

            # When using daemon, the daemon handles hang detection
            # This thread just tracks local state
            if self.client.use_daemon:
                continue

            with self._lock:
                time_since_heartbeat = time.time() - self._last_heartbeat

                # Check if we've exceeded the timeout
                if time_since_heartbeat > self.timeout and not self._hung_logged:
                    # Log the hung event
                    try:
                        self.client.log_event(
                            self.session_id,
                            "application_appears_hung",
                            {
                                "seconds_since_heartbeat": int(time_since_heartbeat),
                                "timeout": self.timeout,
                            },
                        )
                        self._hung_logged = True
                    except Exception as e:
                        # Silently continue if logging fails
                        pass

    def start(self):
        """
        Start the heartbeat monitor thread

        Returns:
            HeartbeatMonitor: self for chaining
        """
        if self._running:
            return self

        self._running = True
        self._last_heartbeat = time.time()
        self._hung_logged = False

        # Create and start the monitor thread as a daemon
        self._thread = threading.Thread(target=self._monitor_loop)
        self._thread.daemon = True
        self._thread.start()

        return self

    def heartbeat(self):
        """
        Send a heartbeat to indicate the application is alive

        This should be called regularly by the application (more frequently
        than the timeout period) to prevent the hung event from being logged.
        """
        with self._lock:
            self._last_heartbeat = time.time()

            # Send heartbeat to daemon
            try:
                url = "{0}/heartbeat".format(self.client.base_url)
                data = {"session_id": self.session_id}
                self.client._make_request(url, method="POST", data=data)
            except Exception:
                # Silently continue if heartbeat fails
                pass

            # Reset hung flag if we receive a heartbeat after being hung
            if self._hung_logged:
                self._hung_logged = False
                # Optionally log recovery
                try:
                    self.client.log_event(self.session_id, "application_recovered", {})
                except Exception:
                    pass

    def stop(self):
        """Stop the heartbeat monitor thread"""
        self._running = False
        if self._thread and self._thread.is_alive():
            self._thread.join(timeout=self.check_interval + 1)

    def is_running(self):
        """Check if the monitor is running"""
        return self._running and self._thread and self._thread.is_alive()


# Convenience class for session management
class Session(object):
    """Represents a datacat session with convenience methods"""

    def __init__(self, client, session_id):
        """
        Initialize a session

        Args:
            client (DatacatClient): Client instance
            session_id (str): Session ID
        """
        self.client = client
        self.session_id = session_id
        self._heartbeat_monitor = None

    def update_state(self, state):
        """Update session state"""
        return self.client.update_state(self.session_id, state)

    def log_event(self, name, data=None):
        """Log an event"""
        return self.client.log_event(self.session_id, name, data)

    def log_metric(self, name, value, tags=None):
        """Log a metric"""
        return self.client.log_metric(self.session_id, name, value, tags)

    def log_exception(self, exc_info=None, extra_data=None):
        """
        Log an exception

        Args:
            exc_info (tuple): Exception info from sys.exc_info(), if None uses current exception
            extra_data (dict): Optional additional data to include
        """
        return self.client.log_exception(self.session_id, exc_info, extra_data)

    def end(self):
        """End the session"""
        # Stop heartbeat monitor if running
        if self._heartbeat_monitor:
            self._heartbeat_monitor.stop()
        return self.client.end_session(self.session_id)

    def get_details(self):
        """Get session details"""
        return self.client.get_session(self.session_id)

    def start_heartbeat_monitor(self, timeout=60, check_interval=5):
        """
        Start a heartbeat monitor for this session

        The monitor runs in a background thread and will log an event if
        no heartbeat is received within the timeout period.

        Args:
            timeout (int): Seconds without heartbeat before logging hung event (default: 60)
            check_interval (int): Seconds between heartbeat checks (default: 5)

        Returns:
            HeartbeatMonitor: The monitor instance
        """
        if self._heartbeat_monitor is None:
            self._heartbeat_monitor = HeartbeatMonitor(
                self.client,
                self.session_id,
                timeout=timeout,
                check_interval=check_interval,
            )
        self._heartbeat_monitor.start()
        return self._heartbeat_monitor

    def heartbeat(self):
        """
        Send a heartbeat to indicate the application is alive

        Must be called after start_heartbeat_monitor() has been called.
        Should be called regularly (more frequently than the timeout period).
        """
        if self._heartbeat_monitor:
            self._heartbeat_monitor.heartbeat()
        else:
            raise Exception(
                "Heartbeat monitor not started. Call start_heartbeat_monitor() first."
            )

    def stop_heartbeat_monitor(self):
        """Stop the heartbeat monitor if running"""
        if self._heartbeat_monitor:
            self._heartbeat_monitor.stop()
            self._heartbeat_monitor = None

    def pause_heartbeat_monitoring(self):
        """
        Pause heartbeat monitoring for this session

        Use this when your application will be intentionally unresponsive
        (e.g., long-running computation, blocking I/O, etc.)

        After pausing, the daemon will not report this session as hung
        until you call resume_heartbeat_monitoring().
        """
        return self.client.pause_heartbeat_monitoring(self.session_id)

    def resume_heartbeat_monitoring(self):
        """
        Resume heartbeat monitoring for this session

        Call this after calling pause_heartbeat_monitoring() to resume
        normal heartbeat monitoring. Remember to start sending heartbeats
        again regularly.
        """
        return self.client.resume_heartbeat_monitoring(self.session_id)


# Factory function for convenience
def create_session(
    base_url="http://localhost:9090",
    daemon_port="8079",
    product=None,
    version=None,
):
    """
    Create a new session and return a Session object

    The client always uses a local daemon subprocess for batching and crash detection.

    Args:
        base_url (str): Base URL of the datacat server (daemon's upstream)
        daemon_port (str): Port for the local daemon
        product (str): Product name (required)
        version (str): Product version (required)

    Returns:
        Session: Session object ready to use

    Raises:
        Exception: If session creation fails or if product/version are not provided
    """
    if not product or not version:
        raise Exception("Product and version are required to create a session")

    client = DatacatClient(base_url, daemon_port=daemon_port)
    session_id = client.register_session(product, version)
    return Session(client, session_id)
