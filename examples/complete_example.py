#!/usr/bin/env python
"""
Complete example: Demonstrating all datacat features

This example shows a complete application using all features:
- Session lifecycle management
- Nested state tracking
- Event and metric logging
- Heartbeat monitoring for hang detection
"""

from __future__ import print_function
import sys
import os
import time
import random

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def main():
    print("=" * 60)
    print("Complete datacat Example - All Features Demo")
    print("=" * 60)
    print()

    # 1. Create and start a session
    print("1. Creating session...")
    session = create_session(
        "http://localhost:9090", product="CompleteExample", version="1.0.0"
    )
    print("   Session ID:", session.session_id)

    # 2. Log session start event
    print("\n2. Logging session start event...")
    session.log_event(
        "session_started",
        {"application": "CompleteExample", "version": "1.0.0", "user": "demo_user"},
    )

    # 3. Set initial nested state
    print("\n3. Setting initial nested state...")
    session.update_state(
        {
            "product": "CompleteExample",  # Top-level product name for web UI
            "version": "1.0.0",  # Top-level version for web UI
            "application": {
                "name": "CompleteExample",
                "version": "1.0.0",
                "status": "initializing",
            },
            "resources": {"memory_mb": 50.0, "cpu_percent": 5.0},
            "windows": {"open": [], "active": None},
            "settings": {"theme": "dark", "autosave": True, "refresh_rate": 60},
        }
    )
    print("   Initial state set")

    # 4. Start heartbeat monitoring
    print("\n4. Starting heartbeat monitor...")
    session.start_heartbeat_monitor(timeout=30, check_interval=5)
    print("   Monitor started (timeout: 30s)")

    # 5. Simulate application startup
    print("\n5. Simulating application startup...")
    session.update_state({"application": {"status": "starting"}})
    time.sleep(1)
    session.heartbeat()

    # Open some windows (randomize the number)
    num_windows = random.randint(2, 4)
    for i in range(num_windows):
        window_name = "Window {}".format(i + 1)
        print("   Opening {}...".format(window_name))

        # Update state with new window (randomized memory usage)
        current_windows = ["Window {}".format(j + 1) for j in range(i + 1)]
        memory_usage = round(50.0 + (i + 1) * random.uniform(20.0, 30.0), 1)
        session.update_state(
            {
                "windows": {"open": current_windows, "active": window_name},
                "resources": {"memory_mb": memory_usage},
            }
        )

        # Log event
        session.log_event("window_opened", {"window": window_name})

        # Log metric
        session.log_metric("window_count", i + 1)
        session.log_metric("memory_mb", memory_usage)

        time.sleep(0.5)
        session.heartbeat()

    # 6. Simulate running application
    print("\n6. Simulating running application...")
    session.update_state({"application": {"status": "running"}})
    session.log_event("application_ready")

    # Simulate work cycles (randomize number of cycles)
    num_cycles = random.randint(3, 7)
    for cycle in range(num_cycles):
        print("   Work cycle {}...".format(cycle + 1))

        # Update metrics
        cpu_usage = random.uniform(20.0, 80.0)
        memory_usage = random.uniform(100.0, 150.0)

        session.log_metric(
            "cpu_percent", cpu_usage, tags=["cycle:{}".format(cycle + 1)]
        )
        session.log_metric(
            "memory_mb", memory_usage, tags=["cycle:{}".format(cycle + 1)]
        )

        # Update resource state
        session.update_state(
            {"resources": {"cpu_percent": cpu_usage, "memory_mb": memory_usage}}
        )

        # Simulate occasional errors during work cycles
        if random.random() < 0.3:  # 30% chance of error per cycle
            try:
                error_types = [
                    ("cache", "Cache miss ratio exceeded threshold"),
                    ("timeout", "Background task timeout"),
                    ("resource", "Memory allocation warning"),
                ]
                error_type, error_msg = random.choice(error_types)
                raise RuntimeError(error_msg)
            except RuntimeError as e:
                print("   ⚠ Error in cycle {}: {}".format(cycle + 1, str(e)))
                session.log_exception(
                    extra_data={
                        "cycle": cycle + 1,
                        "error_category": error_type,
                        "severity": "warning",
                    }
                )

        time.sleep(1)
        session.heartbeat()

    # 7. Change settings
    print("\n7. Changing user settings...")
    session.update_state({"settings": {"theme": "light", "refresh_rate": 120}})
    session.log_event("settings_changed", {"changed": ["theme", "refresh_rate"]})
    session.heartbeat()

    # 8. Close some windows
    print("\n8. Closing windows...")
    session.update_state({"windows": {"open": ["Window 1"], "active": "Window 1"}})
    session.log_event("window_closed", {"window": "Window 2"})
    session.log_event("window_closed", {"window": "Window 3"})
    session.log_metric("window_count", 1)
    session.heartbeat()

    # 9. Get session details
    print("\n9. Retrieving session details...")
    details = session.get_details()
    print("   Session active:", details["active"])
    print("   Application status:", details["state"]["application"]["status"])
    print("   Windows open:", details["state"]["windows"]["open"])
    print("   Current theme:", details["state"]["settings"]["theme"])
    print("   Total events logged:", len(details["events"]))
    print("   Total metrics logged:", len(details["metrics"]))

    # Count errors
    error_count = sum(1 for event in details["events"] if event["name"] == "exception")
    print("   Total errors logged:", error_count)

    # 10. Shutdown
    print("\n10. Shutting down application...")
    session.log_event("application_stopping")
    session.update_state({"application": {"status": "stopping"}})
    time.sleep(0.5)

    # Stop heartbeat monitor
    session.stop_heartbeat_monitor()
    print("    Heartbeat monitor stopped")

    # Log final event and end session
    session.log_event("session_ended")
    session.end()
    print("    Session ended")

    # 11. Verify session ended
    print("\n11. Verifying session ended...")
    final_details = session.get_details()
    print("    Session active:", final_details["active"])
    print("    Ended at:", final_details.get("ended_at", "N/A"))

    print("\n" + "=" * 60)
    print("Demo Complete!")
    print("=" * 60)
    print("\nSession ID: {}".format(session.session_id))
    print("\nYou can now:")
    print("1. Query the data endpoint: curl http://localhost:9090/api/data/sessions")
    print(
        "2. Get this session: curl http://localhost:9090/api/sessions/{}".format(
            session.session_id
        )
    )
    print("3. Build dashboards to visualize the data")


if __name__ == "__main__":
    main()
