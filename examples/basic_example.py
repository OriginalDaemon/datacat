#!/usr/bin/env python
"""
Example: Basic usage of the datacat Python client
"""

from __future__ import print_function
import sys
import os
import random

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def main():
    # Create a new session
    session = create_session("http://localhost:9090", product="BasicExample", version="1.0.0")
    print("Created session:", session.session_id)

    # Update application state
    session.update_state(
        {"app": "example_app", "status": "starting", "version": "1.0.0"}
    )
    print("Updated state: starting")

    # Log startup event
    session.log_event("app_started", {"user": "admin", "environment": "production"})
    print("Logged event: app_started")

    # Log some metrics with randomness for varied results
    startup_time = round(random.uniform(1.5, 4.0), 2)
    memory_usage = round(random.uniform(100.0, 150.0), 1)
    session.log_metric("startup_time", startup_time, tags=["env:prod"])
    session.log_metric("memory_usage", memory_usage, tags=["env:prod", "unit:mb"])
    print(
        "Logged metrics: startup_time={0}, memory_usage={1}".format(
            startup_time, memory_usage
        )
    )

    # Simulate running
    session.update_state({"status": "running"})
    print("Updated state: running")

    # Log some operational metrics with randomness
    requests_per_second = random.randint(800, 1200)
    cpu_usage = round(random.uniform(30.0, 60.0), 1)
    session.log_metric("requests_per_second", requests_per_second, tags=["env:prod"])
    session.log_metric("cpu_usage", cpu_usage, tags=["env:prod", "unit:percent"])
    print(
        "Logged operational metrics: rps={0}, cpu={1}%".format(
            requests_per_second, cpu_usage
        )
    )

    # Simulate some errors during operation
    error_occurred = random.random() < 0.7  # 70% chance of an error
    if error_occurred:
        try:
            # Simulate an error condition
            error_type = random.choice(["database", "network", "timeout"])
            if error_type == "database":
                raise Exception(
                    "Database connection failed: Connection timeout after 30s"
                )
            elif error_type == "network":
                raise Exception("Network error: Failed to reach external API")
            else:
                raise Exception("Operation timeout: Request exceeded 60s limit")
        except Exception as e:
            print("\nError occurred:", str(e))
            session.log_exception(
                extra_data={
                    "error_type": error_type,
                    "severity": "warning",
                    "context": "operational_error",
                }
            )
            print("Logged error to session")

    # Get session details
    details = session.get_details()
    print("\nSession details:")
    print("  ID:", details["id"])
    print("  State:", details["state"])
    print("  Events count:", len(details["events"]))
    print("  Metrics count:", len(details["metrics"]))

    # Count errors
    error_count = sum(1 for event in details["events"] if event["name"] == "exception")
    print("  Errors logged:", error_count)

    # Shutdown
    session.log_event("app_stopped")
    session.update_state({"status": "stopped"})
    print("\nApplication stopped")


if __name__ == "__main__":
    main()
