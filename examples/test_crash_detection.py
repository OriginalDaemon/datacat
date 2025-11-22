#!/usr/bin/env python
"""
Test Crash Detection - Verify daemon properly detects and reports crashed sessions

This script:
1. Creates a session with the daemon
2. Deliberately crashes (exits without calling session.end())
3. The daemon should detect this and mark the session as crashed

Run this and then check the web UI to verify the session is marked as "Crashed"
"""

from __future__ import print_function
import sys
import os
import time

# Add parent directory to path for datacat import
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'python'))

from datacat import create_session

def main():
    print("=" * 70)
    print("Crash Detection Test")
    print("=" * 70)
    print()

    # Create a session
    print("Creating session...")
    session = create_session(
        "http://localhost:9090",
        product="CrashTest",
        version="1.0.0"
    )

    print("Session created: %s" % session.session_id)
    print()

    # Log some activity
    session.update_state({
        'test': 'crash_detection',
        'status': 'running'
    })

    session.log_event(
        'test_started',
        level='info',
        message='Testing crash detection'
    )

    print("Logged some activity...")
    print()

    # Wait a moment
    time.sleep(2)

    # Now deliberately crash without calling session.end()
    print("Simulating crash (exiting without calling session.end())...")
    print()
    print("The daemon should detect this within 5 seconds and:")
    print("  1. Log a 'parent_process_crashed' event")
    print("  2. Mark the session as 'Crashed' on the server")
    print("  3. Shut down the daemon (no more sessions)")
    print()
    print("Check the web UI at http://localhost:8080 to verify!")
    print()

    # Exit without calling session.end() - this simulates a crash
    os._exit(1)  # Use os._exit to avoid cleanup

if __name__ == '__main__':
    main()

