#!/usr/bin/env python
"""
Manual test to demonstrate offline mode functionality.

This script shows that:
1. Client can create sessions when server is down
2. Client can log events, metrics, and state updates when server is down
3. All operations succeed and are queued locally in the daemon
"""

from __future__ import print_function
import os
import sys
import time

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def main():
    print("=" * 60)
    print("Offline Mode Demonstration")
    print("=" * 60)
    print()
    print("This test demonstrates that the daemon can work offline.")
    print("The daemon is configured to connect to a non-existent server.")
    print()

    # Create session with daemon pointing to non-existent server
    # The daemon will create the session locally
    print("1. Creating session (server unavailable)...")
    session = create_session(
        base_url="http://localhost:19999",  # Invalid server
        daemon_port="8079",
        product="OfflineDemo",
        version="1.0.0",
    )
    print("   Session created: {}".format(session.session_id))
    print(
        "   (Note: session ID starts with 'local-session-' indicating offline creation)"
    )
    print()

    # Update state - should work offline
    print("2. Updating state...")
    session.update_state(
        {
            "application": {
                "name": "offline-demo",
                "status": "running",
                "version": "1.0.0",
            },
            "environment": "test",
        }
    )
    print("   State updated successfully")
    print()

    # Log events - should work offline
    print("3. Logging events...")
    session.log_event("app_started", {"timestamp": time.time(), "mode": "offline"})
    print("   Event logged successfully")
    print()

    # Log metrics - should work offline
    print("4. Logging metrics...")
    session.log_metric("cpu_usage", 45.2, tags=["offline", "test"])
    session.log_metric("memory_mb", 256.5, tags=["offline", "test"])
    print("   Metrics logged successfully")
    print()

    # Start heartbeat monitor
    print("5. Starting heartbeat monitor...")
    session.start_heartbeat_monitor(timeout=60)
    session.heartbeat()
    print("   Heartbeat sent successfully")
    print()

    # Get session details from daemon
    print("6. Retrieving session details from daemon...")
    details = session.get_details()
    print("   Session ID: {}".format(details["id"]))
    print("   Active: {}".format(details["active"]))
    print("   State: {}".format(details.get("state", {})))
    print()

    # Update state again to show deep merge
    print("7. Updating state again (testing deep merge)...")
    session.update_state({"application": {"status": "processing"}})
    details = session.get_details()
    print("   Updated state: {}".format(details.get("state", {})))
    print("   (Note: 'name' and 'version' fields are preserved)")
    print()

    # End session
    print("8. Ending session...")
    session.end()
    print("   Session ended successfully")
    print()

    print("=" * 60)
    print("Success! All operations completed successfully in offline mode.")
    print()
    print("What happened:")
    print("- Daemon created session locally when server was unavailable")
    print("- All state updates, events, and metrics were queued locally")
    print("- When server becomes available, daemon will retry sending queued data")
    print("=" * 60)


if __name__ == "__main__":
    main()
