"""
Test realistic program workflow as described in requirements.

This test simulates a typical application (Graphite) lifecycle:
1. Startup with program metadata
2. Configuration state updates
3. Runtime state changes (scene objects, resource modes)
4. Events (heartbeat, user actions)
5. Metrics logging
6. Exception logging

Verifies that:
- Deep merge correctly combines nested state updates
- State history tracks cumulative state at each point
- All events, metrics, and exceptions are properly logged
- Timeline view can reconstruct application lifecycle
"""

import json
import os
import subprocess
import sys
import time
import unittest

# Add python directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import DatacatClient, create_session


class TestProgramWorkflow(unittest.TestCase):
    """Test realistic application workflow"""

    service_process: subprocess.Popen  # type: ignore
    base_url: str

    @classmethod
    def setUpClass(cls):
        """Start the datacat service before tests"""
        cls.service_process = None  # type: ignore
        cls.base_url = "http://localhost:9090"

        # Build the service
        build_result = subprocess.run(
            ["go", "build", "-o", "datacat", "./cmd/datacat-server"],
            cwd=os.path.join(os.path.dirname(__file__), ".."),
            capture_output=True,
        )

        if build_result.returncode != 0:
            raise Exception(f"Failed to build service: {build_result.stderr.decode()}")

        # Start the service
        cls.service_process = subprocess.Popen(
            ["./datacat"],
            cwd=os.path.join(os.path.dirname(__file__), ".."),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

        # Wait for service to start
        time.sleep(2)

        # Verify service is running
        try:
            client = DatacatClient(cls.base_url)
            client.register_session("WorkflowTest", "1.0.0")
        except Exception as e:
            cls.tearDownClass()
            raise Exception(f"Service failed to start: {e}")

    @classmethod
    def tearDownClass(cls):
        """Stop the datacat service after tests"""
        if cls.service_process:
            cls.service_process.terminate()
            cls.service_process.wait(timeout=5)

    def test_graphite_program_workflow(self):
        """
        Test a complete application lifecycle matching the Graphite example.

        This simulates:
        1. Program startup with metadata
        2. Startup complete with configuration
        3. User interactions that modify state
        4. Heartbeat events
        5. Exception logging
        6. Metrics collection
        """
        session = create_session(
            self.base_url,
            use_daemon=False,
            product="GraphiteWorkflow",
            version="2025.10.1",
        )

        # === STARTUP PHASE ===
        # 1. Log startup event
        session.log_event("startup", {"message": "Application starting"})

        # 2. Log initial program metadata
        session.update_state({"program": {"name": "graphite", "version": "2025.10.1"}})

        # Small delay to simulate startup time
        time.sleep(0.1)

        # === POST-STARTUP CONFIGURATION ===
        # 3. Log startup complete event
        session.log_event("startup_complete", {"message": "Initialization finished"})

        # 4. Log startup metrics
        session.log_metric("startup_time", 123.25)
        session.log_metric("service_count", 6.0)

        # 5. Add configuration state (should merge with existing program state)
        session.update_state(
            {
                "program": {"running_mode": "standalone"},
                "resource": {"mode": "content"},
            }
        )

        # Verify state is cumulative after first merge
        state_after_config = session.get_details()
        self.assertEqual(state_after_config["state"]["program"]["name"], "graphite")
        self.assertEqual(state_after_config["state"]["program"]["version"], "2025.10.1")
        self.assertEqual(
            state_after_config["state"]["program"]["running_mode"], "standalone"
        )
        self.assertEqual(state_after_config["state"]["resource"]["mode"], "content")

        time.sleep(0.1)

        # === RUNTIME PHASE ===
        # 6. Log heartbeat events
        session.log_event("heartbeat", {"status": "alive"})

        # 7. Add scene objects state
        session.update_state(
            {
                "scene": {
                    "objects": ["ab1_t1:amarrbase:amarr", "ab2_t1:amarrbase:gallente"]
                }
            }
        )

        # Verify state continues to accumulate
        state_after_scene = session.get_details()
        self.assertEqual(
            state_after_scene["state"]["program"]["running_mode"], "standalone"
        )
        self.assertEqual(state_after_scene["state"]["resource"]["mode"], "content")
        self.assertEqual(
            state_after_scene["state"]["scene"]["objects"],
            ["ab1_t1:amarrbase:amarr", "ab2_t1:amarrbase:gallente"],
        )

        time.sleep(0.1)

        # 8. User changes resource mode (overwrites existing resource.mode)
        session.log_event("resource_change", {"from": "content", "to": "client"})
        session.update_state({"resource": {"mode": "client"}})

        # Verify that nested update overwrites the specific field
        state_after_change = session.get_details()
        self.assertEqual(
            state_after_change["state"]["resource"]["mode"], "client"
        )  # Changed
        self.assertEqual(
            state_after_change["state"]["program"]["name"], "graphite"
        )  # Preserved
        self.assertEqual(
            len(state_after_change["state"]["scene"]["objects"]), 2
        )  # Preserved

        # 9. User interactions - window opened
        session.log_event("window_opened", {"window_id": "main_viewport"})
        session.update_state({"ui": {"windows": ["main_viewport"]}})

        # 10. More metrics
        session.log_metric("memory_mb", 256.5, ["app:graphite"])
        session.log_metric("cpu_percent", 12.3, ["app:graphite"])
        session.log_metric("fps", 60.0, ["app:graphite"])

        time.sleep(0.1)

        # === ERROR HANDLING ===
        # 11. Simulate an exception
        try:
            raise ValueError("Invalid scene configuration")
        except Exception:
            session.log_exception(
                extra_data={"context": "scene_loading", "scene_id": "test_scene"}
            )

        # 12. Log error event
        session.log_event(
            "error",
            {
                "type": "ValueError",
                "message": "Invalid scene configuration",
                "recovered": True,
            },
        )

        # 13. Another heartbeat after recovery
        session.log_event("heartbeat", {"status": "alive"})

        time.sleep(0.1)

        # === VERIFICATION ===
        # Retrieve final session data
        final_data = session.get_details()

        # Verify final cumulative state has all components
        final_state = final_data["state"]
        self.assertEqual(final_state["program"]["name"], "graphite")
        self.assertEqual(final_state["program"]["version"], "2025.10.1")
        self.assertEqual(final_state["program"]["running_mode"], "standalone")
        self.assertEqual(final_state["resource"]["mode"], "client")  # Updated value
        self.assertEqual(
            final_state["scene"]["objects"],
            ["ab1_t1:amarrbase:amarr", "ab2_t1:amarrbase:gallente"],
        )
        self.assertEqual(final_state["ui"]["windows"], ["main_viewport"])

        # Verify events were logged
        events = final_data["events"]
        event_names = [e["name"] for e in events]
        self.assertIn("startup", event_names)
        self.assertIn("startup_complete", event_names)
        self.assertIn("heartbeat", event_names)
        self.assertIn("resource_change", event_names)
        self.assertIn("window_opened", event_names)
        self.assertIn("error", event_names)
        self.assertIn("exception", event_names)  # From log_exception

        # Verify metrics were logged
        metrics = final_data["metrics"]
        metric_names = [m["name"] for m in metrics]
        self.assertIn("startup_time", metric_names)
        self.assertIn("service_count", metric_names)
        self.assertIn("memory_mb", metric_names)
        self.assertIn("cpu_percent", metric_names)
        self.assertIn("fps", metric_names)

        # Verify state history tracks progression
        state_history = final_data["state_history"]
        self.assertGreater(len(state_history), 0, "State history should have snapshots")

        # Verify each snapshot is cumulative
        for i, snapshot in enumerate(state_history):
            self.assertIn("timestamp", snapshot)
            self.assertIn("state", snapshot)
            # First snapshot should have program metadata
            if i == 0:
                self.assertIn("program", snapshot["state"])
                self.assertEqual(snapshot["state"]["program"]["name"], "graphite")

        # Print summary for manual inspection
        print("\n=== Program Workflow Test Summary ===")
        print(f"Total State Updates: {len(state_history)}")
        print(f"Total Events: {len(events)}")
        print(f"Total Metrics: {len(metrics)}")
        print(f"\nFinal State Keys: {list(final_state.keys())}")
        print(f"Program State: {final_state.get('program', {})}")
        print(f"Resource State: {final_state.get('resource', {})}")
        print(f"Scene State: {final_state.get('scene', {})}")
        print(f"UI State: {final_state.get('ui', {})}")

        # End the session
        session.end()

        # Verify session was marked as ended
        ended_data = session.get_details()
        self.assertFalse(ended_data["active"])
        self.assertIsNotNone(ended_data["ended_at"])

    def test_state_history_timeline(self):
        """
        Test that state history creates a proper timeline that can be
        used to reconstruct application state at any point in time.
        """
        session = create_session(
            self.base_url, use_daemon=False, product="StateHistoryTest", version="1.0"
        )

        # Create a series of state updates
        updates = [
            {"program": {"name": "test_app", "version": "1.0"}},
            {"program": {"status": "initializing"}},
            {"config": {"mode": "debug"}},
            {"program": {"status": "running"}},
            {"ui": {"theme": "dark"}},
            {"config": {"mode": "production"}},
        ]

        for update in updates:
            session.update_state(update)
            time.sleep(0.05)  # Small delay between updates

        # Get session data
        data = session.get_details()
        history = data["state_history"]

        # Should have one snapshot per update
        self.assertEqual(len(history), len(updates))

        # Verify first snapshot
        self.assertEqual(history[0]["state"]["program"]["name"], "test_app")
        self.assertEqual(history[0]["state"]["program"]["version"], "1.0")
        self.assertNotIn("status", history[0]["state"]["program"])

        # Verify second snapshot (cumulative)
        self.assertEqual(history[1]["state"]["program"]["name"], "test_app")
        self.assertEqual(history[1]["state"]["program"]["status"], "initializing")

        # Verify third snapshot (adds new top-level key)
        self.assertEqual(history[2]["state"]["config"]["mode"], "debug")
        self.assertEqual(history[2]["state"]["program"]["status"], "initializing")

        # Verify fourth snapshot (updates nested field)
        self.assertEqual(history[3]["state"]["program"]["status"], "running")
        self.assertEqual(history[3]["state"]["config"]["mode"], "debug")

        # Verify sixth snapshot (final state)
        final_snapshot = history[5]
        self.assertEqual(final_snapshot["state"]["program"]["status"], "running")
        self.assertEqual(final_snapshot["state"]["config"]["mode"], "production")
        self.assertEqual(final_snapshot["state"]["ui"]["theme"], "dark")

        print("\n=== State History Timeline Test ===")
        for i, snapshot in enumerate(history):
            print(f"\nSnapshot {i + 1} @ {snapshot['timestamp']}:")
            print(f"  State: {snapshot['state']}")


if __name__ == "__main__":
    unittest.main()
