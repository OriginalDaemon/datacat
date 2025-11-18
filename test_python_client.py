#!/usr/bin/env python3
"""
Test script for the datacat Python client
"""

import sys
import os

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "python"))

from datacat import create_session, DatacatClient


def test_basic_usage():
    print("=== Testing Basic Usage ===")

    # Create a new session
    session = create_session("http://localhost:8080")
    print(f"Created session: {session.session_id}")

    # Update state
    session.update_state({"status": "running", "progress": 0})
    print("Updated state")

    # Log events
    session.log_event("test_started", {"user": "test_user"})
    print("Logged event")

    # Log metrics
    session.log_metric("test_metric", 99.9, tags=["env:test"])
    print("Logged metric")

    # Get session details
    details = session.get_details()
    print(f"Session details: {details}")
    print()


def test_advanced_usage():
    print("=== Testing Advanced Usage ===")

    # Create client
    client = DatacatClient("http://localhost:8080")

    # Register a session manually
    session_id = client.register_session()
    print(f"Registered session: {session_id}")

    # Update state
    client.update_state(session_id, {"phase": "testing"})
    print("Updated state")

    # Log event
    client.log_event(session_id, "phase_change", {"from": "init", "to": "testing"})
    print("Logged event")

    # Log metric
    client.log_metric(session_id, "duration", 1.234, tags=["phase:testing"])
    print("Logged metric")

    # Get session
    session = client.get_session(session_id)
    print(f"Session: {session}")

    # Get all sessions
    all_sessions = client.get_all_sessions()
    print(f"Total sessions: {len(all_sessions)}")
    print()


if __name__ == "__main__":
    test_basic_usage()
    test_advanced_usage()
    print("All tests passed!")
