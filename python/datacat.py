"""
datacat - Python client for the datacat REST API

This module provides a simple interface to interact with the datacat service
for logging application data, events, and metrics.

Compatible with Python 2.7+ and Python 3.x
"""

from __future__ import print_function
import json
import sys
import threading
import time
import traceback

try:
    # Python 3
    from urllib.request import urlopen, Request
    from urllib.error import URLError, HTTPError
except ImportError:
    # Python 2
    from urllib2 import urlopen, Request, URLError, HTTPError


class DatacatClient(object):
    """Client for interacting with the datacat REST API"""

    def __init__(self, base_url="http://localhost:8080"):
        """
        Initialize the datacat client

        Args:
            base_url (str): Base URL of the datacat service
        """
        self.base_url = base_url.rstrip("/")

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
            request.get_method = lambda: method

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

    def register_session(self):
        """
        Register a new session with the datacat service

        Returns:
            str: Session ID

        Raises:
            Exception: If registration fails
        """
        url = "{0}/api/sessions".format(self.base_url)
        response = self._make_request(url, method="POST")
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
        url = "{0}/api/sessions/{1}".format(self.base_url, session_id)
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
        url = "{0}/api/sessions/{1}/state".format(self.base_url, session_id)
        return self._make_request(url, method="POST", data=state)

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

        event_data = {"name": name, "data": data}
        url = "{0}/api/sessions/{1}/events".format(self.base_url, session_id)
        return self._make_request(url, method="POST", data=event_data)

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
        metric_data = {"name": name, "value": float(value), "tags": tags or []}
        url = "{0}/api/sessions/{1}/metrics".format(self.base_url, session_id)
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
        url = "{0}/api/sessions/{1}/end".format(self.base_url, session_id)
        return self._make_request(url, method="POST")

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
        Get all sessions (Grafana endpoint)

        Returns:
            list: List of all sessions

        Raises:
            Exception: If the request fails
        """
        url = "{0}/api/grafana/sessions".format(self.base_url)
        return self._make_request(url, method="GET")


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


# Factory function for convenience
def create_session(base_url="http://localhost:8080"):
    """
    Create a new session and return a Session object

    Args:
        base_url (str): Base URL of the datacat service

    Returns:
        Session: Session object ready to use

    Raises:
        Exception: If session creation fails
    """
    client = DatacatClient(base_url)
    session_id = client.register_session()
    return Session(client, session_id)
