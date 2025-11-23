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
        self, daemon_port="auto", server_url="http://localhost:9090", daemon_binary=None
    ):
        """
        Initialize the daemon manager

        Args:
            daemon_port (str): Port for the daemon to listen on ("auto" finds available port)
            server_url (str): URL of the datacat server
            daemon_binary (str): Path to the daemon binary (auto-detected if None)
        """
        self.daemon_port = daemon_port  # Will be resolved in start()
        self.server_url = server_url
        self.daemon_binary = daemon_binary or self._find_daemon_binary()
        self.process = None
        self._started = False
        self.config_path = None  # Will be set in start()

    def _find_daemon_binary(self):
        """Find the daemon binary in common locations"""
        # Determine binary name based on platform
        binary_name = (
            "datacat-daemon.exe" if sys.platform == "win32" else "datacat-daemon"
        )

        # Check common locations
        possible_paths = [
            "./" + binary_name,  # Current directory
            "../" + binary_name,  # Parent directory (when running from tests/)
            "./cmd/datacat-daemon/" + binary_name,  # Development
            os.path.join(
                os.path.dirname(__file__),
                "..",
                binary_name,
            ),  # Repo root relative to python/
            os.path.join(
                os.path.dirname(__file__),
                "..",
                "cmd",
                "datacat-daemon",
                binary_name,
            ),
            "./bin/" + binary_name,  # Built binaries
            "../bin/" + binary_name,  # Built binaries (from tests/)
            os.path.join(
                os.path.dirname(__file__),
                "..",
                "bin",
                binary_name,
            ),
            binary_name,  # In PATH (check last since we prefer explicit paths)
        ]

        for path in possible_paths:
            if os.path.exists(path):
                return path

        # Check if it's in PATH
        if self._is_in_path(binary_name):
            return binary_name

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

    def _find_available_port(self):
        """Find an available port for the daemon"""
        import socket
        # Create a socket to find an available port
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.bind(('', 0))  # Bind to port 0 to get an available port
            s.listen(1)
            port = s.getsockname()[1]
        return str(port)

    def start(self):
        """Start the daemon subprocess"""
        if self._started and self.process and self.process.poll() is None:
            return  # Already running

        # Find an available port if using auto mode
        if self.daemon_port == "auto" or self.daemon_port == "8079":
            self.daemon_port = self._find_available_port()

        # Set config path now that we have the port
        # Create tmp directory for daemon configs
        config_dir = os.path.join("tmp", "daemon_configs")
        if not os.path.exists(config_dir):
            os.makedirs(config_dir)
        self.config_path = os.path.join(config_dir, "daemon_config_{}.json".format(self.daemon_port))

        # Create config for daemon with this instance's unique port
        config = {
            "daemon_port": self.daemon_port,
            "server_url": self.server_url,
            "batch_interval_seconds": 5,
            "max_batch_size": 100,
            "heartbeat_timeout_seconds": 60,
        }

        # Write config to instance-specific file
        with open(self.config_path, "w") as f:
            json.dump(config, f, indent=2)

        # Start daemon process with instance-specific config
        try:
            # Change directory to where config file is, then run daemon
            self.process = subprocess.Popen(
                [self.daemon_binary],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                stdin=subprocess.PIPE,
                env=dict(os.environ, DATACAT_CONFIG=self.config_path)
            )
            self._started = True

            # Wait a bit for daemon to start
            time.sleep(1)

            # Check if daemon is running
            if self.process.poll() is not None:
                stderr_output = (
                    self.process.stderr.read().decode()
                    if self.process.stderr
                    else "No stderr"
                )
                stdout_output = (
                    self.process.stdout.read().decode()
                    if self.process.stdout
                    else "No stdout"
                )
                raise Exception(
                    f"Daemon failed to start. Stderr: {stderr_output}, Stdout: {stdout_output}"
                )

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

        # Clean up instance-specific config file
        try:
            if os.path.exists(self.config_path):
                os.remove(self.config_path)
        except Exception:
            pass  # Best effort cleanup

    def is_running(self):
        """Check if daemon is running"""
        return self._started and self.process and self.process.poll() is None


class DatacatClient(object):
    """Client for interacting with the datacat daemon (daemon mode only)"""

    def __init__(self, base_url="http://localhost:9090", daemon_port="auto"):
        """
        Initialize the datacat client

        Args:
            base_url (str): Base URL of the datacat server (used as daemon's upstream)
            daemon_port (str): Port for the local daemon ("auto" finds available port)
        """
        self.use_daemon = True  # Always use daemon
        self.daemon_manager = DaemonManager(
            daemon_port=daemon_port, server_url=base_url
        )
        self.daemon_manager.start()
        # Point to daemon instead of server (port determined after start)
        self.base_url = "http://localhost:{}".format(self.daemon_manager.daemon_port)

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

        State updates are merged with the existing state. To delete a key from
        the state, pass None as its value.

        Args:
            session_id (str): Session ID
            state (dict): State data to update. Use None values to delete keys.

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails

        Example:
            # Add or update keys
            session.update_state({"user": "alice", "count": 5})

            # Delete a key
            session.update_state({"user": None})
        """
        url = "{0}/state".format(self.base_url)
        data = {"session_id": session_id, "state": state}
        return self._make_request(url, method="POST", data=data)

    def log_event(
        self,
        session_id,
        name,
        level=None,
        category=None,
        labels=None,
        message=None,
        data=None,
        exception_type=None,
        exception_msg=None,
        stacktrace=None,
        source_file=None,
        source_line=None,
        source_function=None,
    ):
        """
        Log an event to a session

        Args:
            session_id (str): Session ID
            name (str): Event name
            level (str): Optional event level (debug, info, warning, error, critical)
            category (str): Optional category (e.g., logger name, component name)
            labels (list): Optional list of labels/tags
            message (str): Optional human-readable message
            data (dict): Optional event data
            exception_type (str): Optional exception type (for exception events)
            exception_msg (str): Optional exception message (for exception events)
            stacktrace (list): Optional stack trace lines (for exception events)
            source_file (str): Optional source file (for exception events)
            source_line (int): Optional source line (for exception events)
            source_function (str): Optional source function (for exception events)

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails
        """
        if data is None:
            data = {}

        url = "{0}/event".format(self.base_url)
        request_data = {"session_id": session_id, "name": name, "data": data}

        # Add optional fields if provided
        if level:
            request_data["level"] = level
        if category:
            request_data["category"] = category
        if labels:
            request_data["labels"] = labels
        if message:
            request_data["message"] = message
        if exception_type:
            request_data["exception_type"] = exception_type
        if exception_msg:
            request_data["exception_msg"] = exception_msg
        if stacktrace:
            request_data["stacktrace"] = stacktrace
        if source_file:
            request_data["source_file"] = source_file
        if source_line is not None:
            request_data["source_line"] = source_line
        if source_function:
            request_data["source_function"] = source_function

        return self._make_request(url, method="POST", data=request_data)

    def log_metric(self, session_id, name, value, tags=None, metric_type="gauge",
                   count=None, unit=None, metadata=None, delta=None):
        """
        Log a metric to a session

        Args:
            session_id (str): Session ID
            name (str): Metric name
            value (float): Metric value
            tags (list): Optional list of tags
            metric_type (str): Metric type - "gauge", "counter", "histogram", or "timer"
            count (int): Optional count (for timers - number of iterations)
            unit (str): Optional unit (e.g., "seconds", "milliseconds", "bytes")
            metadata (dict): Optional additional metadata
            delta (float): Optional delta (for incremental counters)

        Returns:
            dict: Response from the server

        Raises:
            Exception: If the request fails
        """
        url = "{0}/metric".format(self.base_url)
        metric_data = {
            "session_id": session_id,
            "name": name,
            "type": metric_type,
            "value": float(value),
            "tags": tags or [],
        }
        if count is not None:
            metric_data["count"] = int(count)
        if unit:
            metric_data["unit"] = unit
        if metadata:
            metric_data["metadata"] = metadata
        if delta is not None:
            metric_data["delta"] = float(delta)
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
        return self._make_request(url, method="POST", data={"session_id": session_id})

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

        exception_type = exc_type.__name__ if exc_type else "Unknown"
        exception_msg = str(exc_value) if exc_value else ""

        # Format stack trace as list of strings
        stacktrace_lines = traceback.format_exception(
            exc_type, exc_value, exc_traceback
        )

        # Extract source file, line, and function from the innermost frame
        source_file = None
        source_line = None
        source_function = None

        if exc_traceback:
            # Get the innermost frame (where the exception occurred)
            tb = exc_traceback
            while tb.tb_next:
                tb = tb.tb_next
            frame = tb.tb_frame
            source_file = frame.f_code.co_filename
            source_line = tb.tb_lineno
            source_function = frame.f_code.co_name

        return self.log_event(
            session_id=session_id,
            name="exception",
            level="error",
            category="exception",
            labels=["exception", exception_type],
            message=exception_msg,
            data=extra_data if extra_data else {},
            exception_type=exception_type,
            exception_msg=exception_msg,
            stacktrace=stacktrace_lines,
            source_file=source_file,
            source_line=source_line,
            source_function=source_function,
        )

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
        return self._make_request(url, method="POST", data={"session_id": session_id})

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
        return self._make_request(url, method="POST", data={"session_id": session_id})


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


# Python logging integration
try:
    import logging

    class DatacatHandler(logging.Handler):
        """
        A logging.Handler that sends log records to datacat

        Example usage:
            import logging
            from datacat import DatacatHandler

            session = client.create_session("MyApp", "1.0.0")
            handler = DatacatHandler(session)
            logger = logging.getLogger()
            logger.addHandler(handler)
            logger.setLevel(logging.INFO)

            logger.info("Application started")
            logger.error("Something went wrong", exc_info=True)
        """

        def __init__(self, session, include_exceptions=True):
            """
            Initialize the handler

            Args:
                session (Session): The datacat session to log to
                include_exceptions (bool): Whether to capture exception info (default: True)
            """
            logging.Handler.__init__(self)
            self.session = session
            self.include_exceptions = include_exceptions

            # Map Python logging levels to datacat levels
            self.level_mapping = {
                logging.DEBUG: "debug",
                logging.INFO: "info",
                logging.WARNING: "warning",
                logging.ERROR: "error",
                logging.CRITICAL: "critical",
            }

        def emit(self, record):
            """
            Emit a log record to datacat

            Args:
                record (logging.LogRecord): The log record to emit
            """
            try:
                # Map logging level to datacat level
                level = self.level_mapping.get(record.levelno, "info")

                # Build the message
                message = self.format(record)

                # Build event name from logger name
                event_name = "log.{0}".format(record.name)

                # Build labels
                labels = [record.levelname.lower(), record.name]
                if record.funcName:
                    labels.append(record.funcName)

                # Build data dict
                data = {
                    "logger": record.name,
                    "level_name": record.levelname,
                    "pathname": record.pathname,
                    "lineno": record.lineno,
                    "funcName": record.funcName,
                    "module": record.module,
                }

                # If there's exception info and we're including it
                if self.include_exceptions and record.exc_info:
                    self.session.log_exception(
                        exc_info=record.exc_info, extra_data=data
                    )
                else:
                    # Regular event log
                    self.session.log_event(
                        name=event_name,
                        level=level,
                        category=record.name,
                        labels=labels,
                        message=message,
                        data=data,
                    )
            except Exception:
                # Don't let logging failures crash the application
                self.handleError(record)

    # Export the handler
    __all__ = ["DatacatClient", "Session", "HeartbeatMonitor", "DatacatHandler"]

except ImportError:
    # logging module not available (shouldn't happen in modern Python)
    __all__ = ["DatacatClient", "Session", "AsyncSession", "HeartbeatMonitor"]


# Async logging support for real-time applications (games, etc.)
try:
    import queue  # Python 3
except ImportError:
    import Queue as queue  # Python 2


class AsyncSession(object):
    """
    Non-blocking session wrapper for real-time applications (e.g., games).

    All logging operations return immediately (< 0.01ms) by queueing them
    for processing in a background thread. This prevents blocking the main
    thread during network I/O.

    Ideal for applications with strict frame timing requirements where
    logging overhead must be minimal.

    Example usage:
        session = create_session("http://localhost:9090")
        async_session = AsyncSession(session, queue_size=10000)

        # In game loop - returns immediately!
        async_session.log_event("player_moved", data={"x": 10, "y": 20})
        async_session.log_metric("fps", 60.0)

        # Graceful shutdown - flushes remaining logs
        async_session.shutdown()
    """

    def __init__(self, session, queue_size=10000, drop_on_full=True):
        """
        Initialize async session wrapper

        Args:
            session (Session): The underlying Session to wrap
            queue_size (int): Maximum queue size (default: 10000)
            drop_on_full (bool): If True, drop logs when queue is full.
                                 If False, block until space available.
        """
        self.session = session
        self.drop_on_full = drop_on_full
        self.queue = queue.Queue(maxsize=queue_size)
        self.running = True

        # Statistics
        self.sent_count = 0
        self.dropped_count = 0

        # Start background sender thread
        self.thread = threading.Thread(target=self._background_sender)
        self.thread.daemon = True
        self.thread.start()

    def log_event(self, name, level=None, category=None, labels=None, message=None, data=None):
        """
        Log an event (non-blocking, returns immediately)

        Args:
            name (str): Event name
            level (str): Optional event level
            category (str): Optional category
            labels (list): Optional labels
            message (str): Optional message
            data (dict): Optional event data
        """
        self._queue_item('event', {
            'name': name,
            'level': level,
            'category': category,
            'labels': labels,
            'message': message,
            'data': data
        })

    def log_metric(self, name, value, tags=None):
        """
        Log a metric (non-blocking, returns immediately)

        Args:
            name (str): Metric name
            value (float): Metric value
            tags (list): Optional tags
        """
        self._queue_item('metric', {
            'name': name,
            'value': value,
            'tags': tags
        })

    def update_state(self, state):
        """
        Update session state (non-blocking, returns immediately)

        Args:
            state (dict): State data to update
        """
        self._queue_item('state', {'state': state})

    def log_exception(self, exc_info=None, extra_data=None):
        """
        Log an exception (non-blocking, returns immediately)

        Args:
            exc_info (tuple): Exception info from sys.exc_info()
            extra_data (dict): Optional additional data
        """
        self._queue_item('exception', {
            'exc_info': exc_info,
            'extra_data': extra_data
        })

    def heartbeat(self):
        """Send heartbeat (non-blocking, returns immediately)"""
        if self.session._heartbeat_monitor:
            self.session._heartbeat_monitor.heartbeat()

    def _queue_item(self, item_type, data):
        """Internal method to queue an item for async processing"""
        try:
            if self.drop_on_full:
                self.queue.put_nowait({'type': item_type, 'data': data})
            else:
                self.queue.put({'type': item_type, 'data': data})
        except queue.Full:
            self.dropped_count += 1

    def _background_sender(self):
        """Background thread that processes queued items"""
        while self.running:
            try:
                # Get item with small timeout for batching
                item = self.queue.get(timeout=0.01)

                # Process based on type
                item_type = item['type']
                data = item['data']

                try:
                    if item_type == 'event':
                        self.session.log_event(
                            data['name'],
                            level=data.get('level'),
                            category=data.get('category'),
                            labels=data.get('labels'),
                            message=data.get('message'),
                            data=data.get('data')
                        )
                    elif item_type == 'metric':
                        self.session.log_metric(
                            data['name'],
                            data['value'],
                            tags=data.get('tags')
                        )
                    elif item_type == 'state':
                        self.session.update_state(data['state'])
                    elif item_type == 'exception':
                        self.session.log_exception(
                            exc_info=data.get('exc_info'),
                            extra_data=data.get('extra_data')
                        )

                    self.sent_count += 1

                except Exception as e:
                    # Don't crash background thread on logging errors
                    print("AsyncSession error: {0}".format(str(e)))

            except queue.Empty:
                continue

    def get_stats(self):
        """
        Get logging statistics

        Returns:
            dict: Statistics including sent, dropped, and queued counts
        """
        return {
            'sent': self.sent_count,
            'dropped': self.dropped_count,
            'queued': self.queue.qsize()
        }

    def flush(self, timeout=2.0):
        """
        Wait for queue to drain (blocks until queue is empty or timeout)

        Args:
            timeout (float): Maximum seconds to wait
        """
        deadline = time.time() + timeout
        while not self.queue.empty() and time.time() < deadline:
            time.sleep(0.01)

    def shutdown(self, timeout=2.0):
        """
        Gracefully shutdown async logging

        Flushes remaining logs, stops background thread, and ends session.

        Args:
            timeout (float): Maximum seconds to wait for queue to drain
        """
        # Flush remaining items
        self.flush(timeout)

        # Stop background thread
        self.running = False
        if self.thread.is_alive():
            self.thread.join(timeout=1.0)

        # End underlying session
        return self.session.end()

    def end(self):
        """Alias for shutdown() for consistency with Session API"""
        return self.shutdown()

    # Forward other methods to underlying session
    def get_details(self):
        """Get session details"""
        return self.session.get_details()

    def start_heartbeat_monitor(self, timeout=60, check_interval=5):
        """Start heartbeat monitor"""
        return self.session.start_heartbeat_monitor(timeout, check_interval)

    @property
    def session_id(self):
        """Get session ID"""
        return self.session.session_id

    @property
    def client(self):
        """Get underlying client"""
        return self.session.client


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

    def log_event(
        self, name, level=None, category=None, labels=None, message=None, data=None
    ):
        """Log an event"""
        return self.client.log_event(
            self.session_id,
            name,
            level=level,
            category=category,
            labels=labels,
            message=message,
            data=data,
        )

    def log_metric(self, name, value, tags=None, metric_type="gauge",
                   count=None, unit=None, metadata=None):
        """Log a metric"""
        return self.client.log_metric(self.session_id, name, value, tags,
                                      metric_type, count, unit, metadata)

    def log_gauge(self, name, value, tags=None, unit=None):
        """Log a gauge metric (current value)"""
        return self.log_metric(name, value, tags=tags, metric_type="gauge", unit=unit)

    def log_counter(self, name, delta=1, tags=None):
        """
        Increment a counter metric by delta

        The daemon tracks the cumulative total and sends it to the server.
        You don't need to track the total yourself!

        Args:
            name (str): Counter name
            delta (float): Amount to increment (default: 1)
            tags (list): Optional tags

        Examples:
            # Increment by 1 (most common)
            session.log_counter("requests")

            # Increment by specific amount
            session.log_counter("bytes_sent", delta=1024)

            # With tags
            session.log_counter("errors", tags=["type:validation"])
        """
        return self.client.log_metric(
            self.session_id,
            name,
            value=0,  # Value is ignored for delta counters
            tags=tags,
            metric_type="counter",
            delta=delta
        )

    def log_histogram(self, name, value, unit=None, tags=None, buckets=None, metadata=None):
        """
        Log a histogram metric (value distribution)

        The daemon accumulates samples into buckets and sends bucket counts to the server.

        Args:
            name (str): Histogram name
            value (float): Sample value
            unit (str): Optional unit (e.g., "seconds", "bytes")
            tags (list): Optional tags
            buckets (list): Optional bucket boundaries. If not specified, uses default buckets.
            metadata (dict): Optional additional metadata

        Examples:
            # Default buckets (covers microseconds to minutes)
            session.log_histogram("request_latency", 0.045)

            # Custom buckets for FPS (frame times)
            # +60fps, 60fps, 30fps, 20fps, 10fps, <10fps
            fps_buckets = [1/60, 1/30, 1/20, 1/10, float('inf')]
            session.log_histogram("frame_time", 0.016, buckets=fps_buckets)
        """
        # Add buckets to metadata if specified
        if buckets is not None:
            if metadata is None:
                metadata = {}
            metadata["buckets"] = buckets

        return self.log_metric(name, value, unit=unit, tags=tags, metric_type="histogram", metadata=metadata)

    def log_timer(self, name, duration, count=None, tags=None, unit="seconds"):
        """Log a timer metric (duration measurement)"""
        return self.log_metric(name, duration, tags=tags, metric_type="timer",
                              count=count, unit=unit)

    def timer(self, name, count=None, tags=None, unit="seconds"):
        """
        Context manager for timing code execution

        Args:
            name (str): Timer name
            count (int): Optional count (for iterations)
            tags (list): Optional tags
            unit (str): Time unit ("seconds" or "milliseconds")

        Returns:
            Timer: Timer object (can access .count property)

        Examples:
            # Basic timer
            with session.timer("load_data"):
                data = load_data()

            # Timer with known count
            items = get_items()
            with session.timer("process_items", count=len(items)):
                for item in items:
                    process(item)

            # Timer with incremental count
            with session.timer("process_queue") as t:
                while queue.has_items():
                    t.count += 1
                    process(queue.pop())
        """
        return Timer(self, name, count=count, tags=tags, unit=unit)

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
class Timer(object):
    """
    Context manager for timing code execution

    Automatically measures duration and logs as a timer metric when exiting context.
    Supports counting iterations for average time calculations.
    """

    def __init__(self, session, name, count=None, tags=None, unit="seconds"):
        """
        Initialize timer

        Args:
            session: Session object
            name (str): Timer name
            count (int): Optional initial count
            tags (list): Optional tags
            unit (str): Time unit ("seconds" or "milliseconds")
        """
        self.session = session
        self.name = name
        self.count = count if count is not None else 0
        self.tags = tags
        self.unit = unit
        self.start_time = None

    def __enter__(self):
        """Start timing"""
        self.start_time = time.time()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Stop timing and log metric"""
        end_time = time.time()
        duration = end_time - self.start_time

        # Convert to specified unit
        if self.unit == "milliseconds":
            duration = duration * 1000

        # Log the timer metric
        self.session.log_timer(
            self.name,
            duration,
            count=self.count if self.count > 0 else None,
            tags=self.tags,
            unit=self.unit
        )

        # Don't suppress exceptions
        return False


def create_session(
    base_url="http://localhost:9090",
    daemon_port="auto",
    product=None,
    version=None,
    async_mode=False,
    queue_size=10000
):
    """
    Create a new session and return a Session object

    The client always uses a local daemon subprocess for batching and crash detection.

    Args:
        base_url (str): Base URL of the datacat server (daemon's upstream)
        daemon_port (str): Port for the local daemon ("auto" finds available port)
        product (str): Product name (required)
        version (str): Product version (required)
        async_mode (bool): If True, return AsyncSession for non-blocking logging
                           (recommended for games and real-time applications)
        queue_size (int): Queue size for async mode (default: 10000)

    Returns:
        Session or AsyncSession: Session object (or AsyncSession if async_mode=True)

    Raises:
        Exception: If session creation fails or if product/version are not provided

    Examples:
        # Standard blocking mode
        session = create_session("http://localhost:9090", product="MyApp", version="1.0")
        session.update_state({"status": "running"})
        session.log_event("app_started")
        session.end()

        # Async mode for games (non-blocking, < 0.01ms per call)
        session = create_session("http://localhost:9090", product="MyGame", version="1.0", async_mode=True)
        session.log_event("player_moved", data={"x": 10, "y": 20})  # Returns immediately!
        session.shutdown()  # Flushes queue and ends session
    """
    if not product or not version:
        raise Exception("Product and version are required to create a session")

    client = DatacatClient(base_url, daemon_port=daemon_port)
    session_id = client.register_session(product, version)
    session = Session(client, session_id)

    if async_mode:
        return AsyncSession(session, queue_size=queue_size)
    else:
        return session
