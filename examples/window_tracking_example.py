#!/usr/bin/env python
"""
Example: Window state tracking with nested data structures
Demonstrates tracking complex application state with hierarchical updates
"""

from __future__ import print_function
import sys
import os
import time

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def main():
    # Create a session for this application instance
    session = create_session(
        "http://localhost:9090", product="WindowTrackingExample", version="1.0.0"
    )
    print("Application Session:", session.session_id)
    print()

    # Initialize application - register session start
    session.log_event("session_started", {"application": "WindowManager"})
    print("Session started")

    # Set initial state with nested structure
    session.update_state(
        {
            "application": {"name": "WindowManager", "version": "1.0.0"},
            "window_state": {"open": [], "active": None},
            "memory": {"footprint_mb": 50.2},
            "settings": {"theme": "dark", "autosave": True},
        }
    )
    print("Initial state set")
    print()

    # Simulate opening windows - partial state updates
    time.sleep(0.5)
    print("Opening window 1...")
    session.update_state({"window_state": {"open": ["window 1"]}})
    session.log_event("window_opened", {"window": "window 1"})
    session.log_metric("window_count", 1)

    time.sleep(0.5)
    print("Opening window 2...")
    session.update_state(
        {
            "window_state": {"open": ["window 1", "window 2"], "active": "window 2"},
            "memory": {"footprint_mb": 75.3},
        }
    )
    session.log_event("window_opened", {"window": "window 2"})
    session.log_metric("window_count", 2)
    session.log_metric("memory_mb", 75.3)

    time.sleep(0.5)
    print("Opening dynamic window...")
    session.update_state(
        {
            "window_state": {
                "open": ["window 1", "window 2", "some dynamic window"],
                "active": "some dynamic window",
            },
            "memory": {"footprint_mb": 92.7},
        }
    )
    session.log_event("window_opened", {"window": "some dynamic window"})
    session.log_metric("window_count", 3)
    session.log_metric("memory_mb", 92.7)

    # Update settings while keeping other state intact
    time.sleep(0.5)
    print("Changing theme to light...")
    session.update_state({"settings": {"theme": "light"}})
    session.log_event("settings_changed", {"setting": "theme", "value": "light"})

    # Track window activity
    time.sleep(0.5)
    print("Switching to window 1...")
    session.update_state({"window_state": {"active": "window 1"}})
    session.log_event("window_activated", {"window": "window 1"})

    # Close a window
    time.sleep(0.5)
    print("Closing window 2...")
    session.update_state(
        {
            "window_state": {"open": ["window 1", "some dynamic window"]},
            "memory": {"footprint_mb": 68.5},
        }
    )
    session.log_event("window_closed", {"window": "window 2"})
    session.log_metric("window_count", 2)
    session.log_metric("memory_mb", 68.5)

    # Get current state
    print()
    print("Getting current session state...")
    details = session.get_details()
    print("Current state:")
    print("  Active:", details["active"])
    print("  Windows open:", details["state"].get("window_state", {}).get("open", []))
    print("  Active window:", details["state"].get("window_state", {}).get("active"))
    print(
        "  Memory footprint:",
        details["state"].get("memory", {}).get("footprint_mb"),
        "MB",
    )
    print("  Theme:", details["state"].get("settings", {}).get("theme"))
    print("  Total events:", len(details["events"]))
    print("  Total metrics:", len(details["metrics"]))

    # End session
    print()
    print("Ending session...")
    session.end()
    session.log_event("session_ended")

    # Verify session ended
    final_state = session.get_details()
    print("Session ended. Active:", final_state["active"])
    print("Session ID:", session.session_id)


if __name__ == "__main__":
    main()
