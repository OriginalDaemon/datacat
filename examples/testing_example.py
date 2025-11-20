#!/usr/bin/env python
"""
Example: Testing and CI/CD tracking with datacat
"""

from __future__ import print_function
import sys
import os
import time

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def run_test(name, duration):
    """Simulate running a test"""
    time.sleep(duration)
    return {"passed": True, "duration": duration}


def main():
    # Create a session for this test run
    session = create_session("http://localhost:9090", product="TestingExample", version="1.0.0")
    print("Test Run Session:", session.session_id)

    # Initialize test suite
    session.update_state(
        {"test_suite": "integration_tests", "status": "running", "total_tests": 5}
    )
    session.log_event(
        "test_run_started", {"suite": "integration_tests", "runner": "pytest"}
    )

    # Run tests
    tests = [
        ("test_api_authentication", 0.5),
        ("test_data_validation", 0.3),
        ("test_error_handling", 0.4),
        ("test_performance", 1.0),
        ("test_integration", 0.8),
    ]

    passed = 0
    failed = 0

    for test_name, duration in tests:
        print(f"Running {test_name}...")

        session.log_event("test_started", {"name": test_name})

        result = run_test(test_name, duration)

        if result["passed"]:
            passed += 1
            status = "passed"
        else:
            failed += 1
            status = "failed"

        session.log_event(
            "test_completed",
            {"name": test_name, "status": status, "duration": result["duration"]},
        )

        session.log_metric(
            "test_duration",
            result["duration"],
            tags=[f"test:{test_name}", f"status:{status}"],
        )

        print(f"  {status} ({result['duration']}s)")

    # Update final state
    session.update_state(
        {
            "status": "completed",
            "passed": passed,
            "failed": failed,
            "success_rate": (passed / len(tests)) * 100,
        }
    )

    session.log_event(
        "test_run_completed", {"passed": passed, "failed": failed, "total": len(tests)}
    )

    # Log summary metrics
    session.log_metric("tests_passed", passed, tags=["suite:integration"])
    session.log_metric("tests_failed", failed, tags=["suite:integration"])
    session.log_metric(
        "success_rate",
        (passed / len(tests)) * 100,
        tags=["suite:integration", "unit:percent"],
    )

    print(f"\nTest Run Complete:")
    print(f"  Passed: {passed}")
    print(f"  Failed: {failed}")
    print(f"  Success Rate: {(passed / len(tests)) * 100:.1f}%")
    print(f"  Session ID: {session.session_id}")


if __name__ == "__main__":
    main()
