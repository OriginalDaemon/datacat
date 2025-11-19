#!/usr/bin/env python
"""
Example: Basic usage of the datacat Python client
"""

from __future__ import print_function
import sys
import os

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def main():
    # Create a new session
    session = create_session("http://localhost:8080")
    print("Created session:", session.session_id)

    # Update application state
    session.update_state(
        {"app": "example_app", "status": "starting", "version": "1.0.0"}
    )
    print("Updated state: starting")

    # Log startup event
    session.log_event("app_started", {"user": "admin", "environment": "production"})
    print("Logged event: app_started")

    # Log some metrics
    session.log_metric("startup_time", 2.5, tags=["env:prod"])
    session.log_metric("memory_usage", 128.5, tags=["env:prod", "unit:mb"])
    print("Logged metrics")

    # Simulate running
    session.update_state({"status": "running"})
    print("Updated state: running")

    # Log some operational metrics
    session.log_metric("requests_per_second", 1000, tags=["env:prod"])
    session.log_metric("cpu_usage", 45.2, tags=["env:prod", "unit:percent"])
    print("Logged operational metrics")

    # Get session details
    details = session.get_details()
    print("\nSession details:")
    print("  ID:", details["id"])
    print("  State:", details["state"])
    print("  Events count:", len(details["events"]))
    print("  Metrics count:", len(details["metrics"]))

    # Shutdown
    session.log_event("app_stopped")
    session.update_state({"status": "stopped"})
    print("\nApplication stopped")


if __name__ == "__main__":
    main()
