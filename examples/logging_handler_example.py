#!/usr/bin/env python
"""
Example: Using the custom DatacatLoggingHandler

Demonstrates how to integrate datacat with Python's standard logging module
using a custom handler that formats and sends log messages and exceptions.
"""

from __future__ import print_function
import sys
import os
import logging
import traceback
import time
import random
from datetime import datetime

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


class DatacatLoggingHandler(logging.Handler):
    """
    Custom logging handler that sends log messages to datacat.

    This handler integrates Python's logging module with datacat, automatically
    capturing log messages, formatting stack traces for exceptions, and sending
    them as structured events.

    Features:
    - Captures all log levels (DEBUG, INFO, WARNING, ERROR, CRITICAL)
    - Formats exceptions with full stack traces
    - Includes logger name, level, and timestamp
    - Gracefully handles logging errors
    """

    def __init__(self, datacat_session):
        """
        Initialize the handler

        Args:
            datacat_session: A datacat Session object
        """
        super(DatacatLoggingHandler, self).__init__()
        self.datacat_session = datacat_session

    def emit(self, record):
        """
        Emit a log record to datacat

        Args:
            record: A logging.LogRecord instance
        """
        try:
            # Format the log message
            log_entry = self.format(record)

            # Prepare event data
            event_data = {
                "level": record.levelname,
                "logger": record.name,
                "message": record.getMessage(),
                "module": record.module,
                "function": record.funcName,
                "line": record.lineno,
                "timestamp": datetime.fromtimestamp(record.created).isoformat(),
            }

            # If there's exception info, include formatted traceback
            if record.exc_info:
                event_data["exception"] = {
                    "type": record.exc_info[0].__name__,
                    "message": str(record.exc_info[1]),
                    "traceback": traceback.format_exception(*record.exc_info),
                }

            # Add any extra fields from the record
            for key in record.__dict__:
                if key not in [
                    "name",
                    "msg",
                    "args",
                    "created",
                    "filename",
                    "funcName",
                    "levelname",
                    "levelno",
                    "lineno",
                    "module",
                    "msecs",
                    "message",
                    "pathname",
                    "process",
                    "processName",
                    "relativeCreated",
                    "thread",
                    "threadName",
                    "exc_info",
                    "exc_text",
                    "stack_info",
                ]:
                    event_data[key] = record.__dict__[key]

            # Send to datacat
            event_name = "log_" + record.levelname.lower()
            if record.exc_info:
                event_name = "exception"

            self.datacat_session.log_event(event_name, event_data)

        except Exception:
            # Don't let logging errors crash the application
            self.handleError(record)


def setup_logging(session, level=logging.INFO):
    """
    Setup logging with datacat handler

    Args:
        session: A datacat Session object
        level: Logging level (default: INFO)

    Returns:
        logger: Configured logger instance
    """
    logger = logging.getLogger("myapp")
    logger.setLevel(level)

    # Remove any existing handlers
    logger.handlers = []

    # Console handler for local visibility
    console_handler = logging.StreamHandler()
    console_handler.setLevel(level)
    console_formatter = logging.Formatter(
        "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )
    console_handler.setFormatter(console_formatter)
    logger.addHandler(console_handler)

    # Datacat handler for remote logging
    datacat_handler = DatacatLoggingHandler(session)
    datacat_handler.setLevel(level)
    datacat_formatter = logging.Formatter(
        "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )
    datacat_handler.setFormatter(datacat_formatter)
    logger.addHandler(datacat_handler)

    return logger


def risky_database_operation(record_id):
    """Simulates a database operation that might fail"""
    if record_id < 0:
        raise ValueError("Record ID cannot be negative")
    if record_id > 1000:
        raise KeyError("Record not found: {}".format(record_id))
    return {"id": record_id, "data": "Some data"}


def process_user_request(user_id, action):
    """Simulates processing a user request"""
    if action == "delete" and user_id == 1:
        raise RuntimeError("Cannot delete admin user")
    return {"status": "success", "user_id": user_id, "action": action}


def main():
    print("=" * 70)
    print("Python Application with Custom Logging Handler")
    print("=" * 70)
    print()

    # Create datacat session
    session = create_session(
        "http://localhost:9090", product="LoggingHandlerExample", version="1.0.0"
    )
    print("Session ID:", session.session_id)
    print()

    # Setup logging with datacat handler
    logger = setup_logging(session, level=logging.DEBUG)

    # Set initial state
    session.update_state(
        {
            "application": {
                "name": "logging-handler-example",
                "version": "1.0.0",
                "started_at": datetime.now().isoformat(),
            }
        }
    )

    # Log application startup
    logger.info("Application started", extra={"session_id": session.session_id})

    # Demonstrate different log levels
    print("\n1. Demonstrating different log levels:")
    print("-" * 70)
    logger.debug("Debug message: Detailed diagnostic information")
    logger.info("Info message: Application is running normally")
    logger.warning("Warning message: Something unexpected happened")

    time.sleep(1)

    # Demonstrate exception logging
    print("\n2. Demonstrating exception logging:")
    print("-" * 70)

    test_records = [-5, 50, 1500, 100]
    for record_id in test_records:
        try:
            print("  Processing record {}...".format(record_id))
            result = risky_database_operation(record_id)
            logger.info(
                "Database operation successful",
                extra={"record_id": record_id, "operation": "read"},
            )
            print("    ✓ Success")
        except (ValueError, KeyError) as e:
            print("    ✗ Error: {}".format(str(e)))
            logger.error(
                "Database operation failed: {}".format(str(e)),
                exc_info=True,
                extra={"record_id": record_id, "operation": "read"},
            )

        time.sleep(0.5)

    # Demonstrate nested exceptions
    print("\n3. Demonstrating nested exception handling:")
    print("-" * 70)

    try:
        try:
            print("  Processing user request...")
            result = process_user_request(1, "delete")
        except RuntimeError as inner_error:
            # Re-raise with additional context
            raise Exception("Request processing failed") from inner_error
    except Exception as outer_error:
        print("  ✗ Caught exception: {}".format(str(outer_error)))
        logger.critical(
            "Critical error in request processing",
            exc_info=True,
            extra={"user_id": 1, "action": "delete", "severity": "critical"},
        )

    # Demonstrate logging with custom fields
    print("\n4. Demonstrating custom log fields:")
    print("-" * 70)

    logger.info(
        "User login successful",
        extra={
            "user_id": 42,
            "username": "john_doe",
            "ip_address": "192.168.1.100",
            "login_method": "password",
        },
    )
    print("  Logged user login with custom fields")

    # Simulate some application metrics
    print("\n5. Logging metrics alongside log messages:")
    print("-" * 70)

    for i in range(5):
        response_time = random.uniform(0.1, 2.0)
        logger.info(
            "HTTP request completed",
            extra={
                "method": "GET",
                "path": "/api/users",
                "status_code": 200,
                "response_time_ms": round(response_time * 1000, 2),
            },
        )
        session.log_metric("response_time", response_time, tags=["endpoint:/api/users"])
        print("  Request {}/5 - Response time: {:.2f}s".format(i + 1, response_time))
        time.sleep(0.3)

    # Get session statistics
    print("\n" + "=" * 70)
    print("Session Summary")
    print("=" * 70)

    details = session.get_details()
    events = details.get("events", [])

    # Count log levels
    log_counts = {}
    exception_count = 0

    for event in events:
        if event["name"] == "exception":
            exception_count += 1
        elif event["name"].startswith("log_"):
            level = event["name"][4:].upper()
            log_counts[level] = log_counts.get(level, 0) + 1

    print("\nLog Counts by Level:")
    for level in ["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]:
        count = log_counts.get(level, 0)
        print("  {}: {}".format(level, count))

    print("\nExceptions: {}".format(exception_count))
    print("Metrics: {}".format(len(details.get("metrics", []))))
    print("Total Events: {}".format(len(events)))

    # Application shutdown
    logger.info("Application shutting down")
    session.log_event("application_stopped")
    session.end()

    print("\nSession ended successfully")
    print("Session ID:", session.session_id)
    print("\nView session at: http://localhost:8080/session/" + session.session_id)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInterrupted by user")
        sys.exit(0)
    except Exception as e:
        print("\n\n❌ Fatal error: {}".format(str(e)))
        traceback.print_exc()
        sys.exit(1)
