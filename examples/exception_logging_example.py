#!/usr/bin/env python
"""
Example: Python application with exception logging

Demonstrates how to use the datacat Python client with exception logging
for error tracking and monitoring.
"""

from __future__ import print_function
import sys
import os
import time
import random

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def risky_operation(value):
    """Simulates an operation that might fail"""
    if value < 0:
        raise ValueError("Value cannot be negative")
    if value > 100:
        raise RuntimeError("Value too large")
    return value * 2


def main():
    print("=" * 60)
    print("Python Application with Exception Logging")
    print("=" * 60)
    print()

    # Create session
    session = create_session(
        "http://localhost:9090", product="ExceptionLoggingExample", version="1.0.0"
    )
    print("Session ID:", session.session_id)

    # Start heartbeat monitor
    session.start_heartbeat_monitor(timeout=60)
    print("Heartbeat monitor started")

    # Set initial state
    session.update_state(
        {
            "application": {"name": "exception-example", "version": "1.0.0"},
            "statistics": {"total_operations": 0, "successful": 0, "failed": 0},
        }
    )

    session.log_event("application_started")

    # Perform operations with exception handling
    test_values = [10, 50, -5, 75, 150, 30, 95]

    for i, value in enumerate(test_values):
        print(f"\nOperation {i+1}: Processing value {value}")
        session.heartbeat()

        try:
            result = risky_operation(value)

            # Log successful operation
            session.log_event(
                "operation_completed",
                {"operation_id": i + 1, "input": value, "output": result},
            )

            session.log_metric("operation_value", result, tags=["status:success"])

            # Update success count
            session.update_state(
                {
                    "statistics": {
                        "total_operations": i + 1,
                        "successful": session.get_details()["state"]["statistics"][
                            "successful"
                        ]
                        + 1,
                    }
                }
            )

            print(f"  ✓ Success: {value} → {result}")

        except (ValueError, RuntimeError) as e:
            # Log exception with context
            print(f"  ✗ Error: {type(e).__name__}: {e}")

            session.log_exception(
                extra_data={
                    "operation_id": i + 1,
                    "input_value": value,
                    "operation": "risky_operation",
                }
            )

            # Update failure count
            stats = session.get_details()["state"]["statistics"]
            session.update_state(
                {
                    "statistics": {
                        "total_operations": i + 1,
                        "failed": stats.get("failed", 0) + 1,
                    }
                }
            )

        time.sleep(1)

    # Demonstrate nested exception logging
    print("\n\nDemonstrating nested exception handling:")

    try:
        try:
            # Inner exception
            result = risky_operation(-10)
        except ValueError as inner_error:
            # Re-raise with additional context
            raise RuntimeError("Failed to process operation") from inner_error
    except RuntimeError as outer_error:
        print(f"  Caught outer exception: {outer_error}")
        session.log_exception(
            extra_data={"context": "nested_exception_demo", "severity": "high"}
        )

    # Get final statistics
    print("\n" + "=" * 60)
    print("Final Statistics")
    print("=" * 60)

    details = session.get_details()
    stats = details["state"]["statistics"]

    print(f"Total Operations: {stats['total_operations']}")
    print(f"Successful: {stats.get('successful', 0)}")
    print(f"Failed: {stats.get('failed', 0)}")
    print(f"Total Events: {len(details['events'])}")
    print(f"Total Metrics: {len(details['metrics'])}")

    # Count exceptions
    exception_count = sum(
        1 for event in details["events"] if event["name"] == "exception"
    )
    print(f"Exceptions Logged: {exception_count}")

    # End session
    session.log_event("application_completed", {"total_exceptions": exception_count})
    session.stop_heartbeat_monitor()
    session.end()

    print("\nSession ended successfully")
    print("Session ID:", session.session_id)
    print("\nView session at: http://localhost:8080/session/" + session.session_id)


if __name__ == "__main__":
    main()
