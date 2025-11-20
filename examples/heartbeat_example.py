#!/usr/bin/env python
"""
Example: Heartbeat monitoring to detect application hangs

This example demonstrates how to use the heartbeat monitor to detect
when an application appears to be hung (not responding).
"""

from __future__ import print_function
import sys
import os
import time

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def simulate_healthy_application():
    """Simulate a healthy application that sends regular heartbeats"""
    print("=== Simulating Healthy Application ===")

    session = create_session(
        "http://localhost:9090", product="HeartbeatExample", version="1.0.0"
    )
    print("Session ID:", session.session_id)

    # Start heartbeat monitor with 10 second timeout for demo purposes
    session.start_heartbeat_monitor(timeout=10, check_interval=2)
    print("Heartbeat monitor started (timeout: 10s, check interval: 2s)")

    session.log_event("application_started")
    print("Application started")

    # Simulate normal operation with regular heartbeats
    for i in range(15):
        time.sleep(1)
        session.heartbeat()
        print(".", end="", flush=True)

        # Do some work
        session.update_state({"iteration": i, "status": "running"})

    print("\nApplication completed normally")
    session.log_event("application_completed")
    session.end()
    print("Session ended normally")
    print()


def simulate_hanging_application():
    """Simulate an application that hangs (stops sending heartbeats)"""
    print("=== Simulating Hanging Application ===")

    session = create_session(
        "http://localhost:9090", product="HeartbeatExample", version="1.0.0"
    )
    print("Session ID:", session.session_id)

    # Start heartbeat monitor with 10 second timeout for demo purposes
    session.start_heartbeat_monitor(timeout=10, check_interval=2)
    print("Heartbeat monitor started (timeout: 10s, check interval: 2s)")

    session.log_event("application_started")
    print("Application started")

    # Simulate normal operation for a bit
    for i in range(5):
        time.sleep(1)
        session.heartbeat()
        print(".", end="", flush=True)
        session.update_state({"iteration": i, "status": "running"})

    print("\nApplication appears to hang (no more heartbeats)...")
    session.update_state({"status": "hung"})

    # Simulate hang - no heartbeats sent
    # Wait long enough for the monitor to detect the hang
    time.sleep(15)

    print("Checking session details...")
    details = session.get_details()

    # Look for the hung event
    hung_event = None
    for event in details["events"]:
        if event["name"] == "application_appears_hung":
            hung_event = event
            break

    if hung_event:
        print("✓ Hung event detected!")
        print("  Event:", hung_event["name"])
        print("  Data:", hung_event["data"])
    else:
        print("✗ No hung event found")

    # Note: We deliberately don't call session.end() to simulate a crashed app
    print("Session not ended (simulating crash)")
    print()


def simulate_recovering_application():
    """Simulate an application that hangs but then recovers"""
    print("=== Simulating Application with Recovery ===")

    session = create_session(
        "http://localhost:9090", product="HeartbeatExample", version="1.0.0"
    )
    print("Session ID:", session.session_id)

    # Start heartbeat monitor with 10 second timeout for demo purposes
    session.start_heartbeat_monitor(timeout=10, check_interval=2)
    print("Heartbeat monitor started (timeout: 10s, check interval: 2s)")

    session.log_event("application_started")
    print("Application started")

    # Normal operation
    for i in range(3):
        time.sleep(1)
        session.heartbeat()
        print(".", end="", flush=True)

    print("\nApplication hangs...")
    time.sleep(15)  # Wait for hang detection

    print("Application recovers and sends heartbeat...")
    session.heartbeat()  # This should trigger a recovery event

    # Continue normal operation
    for i in range(3):
        time.sleep(1)
        session.heartbeat()
        print(".", end="", flush=True)

    print("\nApplication completed")
    details = session.get_details()

    print("Events logged:")
    for event in details["events"]:
        print("  -", event["name"])

    session.end()
    print("Session ended normally")
    print()


if __name__ == "__main__":
    # Run the examples
    simulate_healthy_application()
    time.sleep(1)

    simulate_hanging_application()
    time.sleep(1)

    simulate_recovering_application()

    print("\n=== Summary ===")
    print("You can now query Grafana to find:")
    print("1. Sessions that started but never ended (active=true)")
    print("2. Sessions with 'application_appears_hung' as the last event")
    print("3. Sessions that recovered after hanging")
