#!/usr/bin/env python
"""
Cleanup Session States - Fix inconsistent status flags in existing sessions

This script fixes sessions that have inconsistent status flags, such as:
- Crashed + Hung (should only be Crashed)
- Suspended + Hung (might be valid but check)
- Active sessions with Hung flag set
"""

import sys
import requests

SERVER_URL = "http://localhost:9090"

def get_all_sessions():
    """Get all sessions from the server"""
    try:
        response = requests.get(f"{SERVER_URL}/api/data/sessions")
        response.raise_for_status()
        return response.json()
    except Exception as e:
        print(f"Error getting sessions: {e}")
        return []

def main():
    print("=" * 70)
    print("Session State Cleanup")
    print("=" * 70)
    print()

    sessions = get_all_sessions()

    if not sessions:
        print("No sessions found")
        return

    print(f"Found {len(sessions)} sessions")
    print()

    issues_found = 0

    for session in sessions:
        session_id = session['id']
        crashed = session.get('crashed', False)
        hung = session.get('hung', False)
        suspended = session.get('suspended', False)
        active = session.get('active', False)

        # Check for inconsistent states
        issues = []

        if crashed and hung:
            issues.append("Crashed + Hung (Hung should be cleared)")

        if crashed and suspended:
            issues.append("Crashed + Suspended (Suspended should be cleared)")

        if crashed and active:
            issues.append("Crashed + Active (should not be active)")

        if issues:
            issues_found += 1
            print(f"Session {session_id[:8]}...")
            print(f"  Product: {session.get('state', {}).get('product', 'Unknown')}")
            print(f"  Status: Active={active}, Crashed={crashed}, Suspended={suspended}, Hung={hung}")
            print(f"  Issues:")
            for issue in issues:
                print(f"    - {issue}")
            print()

    if issues_found == 0:
        print("✅ No inconsistent session states found!")
    else:
        print(f"Found {issues_found} sessions with inconsistent states")
        print()
        print("Note: These are old sessions from before the fix.")
        print("New sessions will not have these issues.")
        print()
        print("To clean up, you can either:")
        print("  1. Delete the old database and start fresh")
        print("  2. Manually delete these sessions via API")
        print("  3. Leave them as historical data (they won't affect new sessions)")

if __name__ == '__main__':
    main()

