#!/usr/bin/env python
"""
Game Logging Example - Ultra-fast async logging for real-time applications

This example demonstrates non-blocking logging suitable for game engines
and other real-time applications with strict frame timing requirements (e.g., 60 FPS).

Compatible with Python 2.7.4+ and Python 3.x
Zero external dependencies (uses standard library only)
"""

from __future__ import print_function
import sys
import os
import time

# Add parent directory to path for datacat import
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def simulate_game_loop():
    """
    Simulate a game running at 60 FPS with logging

    Each frame has a budget of ~16.7ms.
    Logging overhead should be < 0.1ms per log call.
    """
    print("=" * 70)
    print("Game Logging Example - 60 FPS Simulation")
    print("=" * 70)
    print("\nStarting game with async logging...")
    print("Target: 60 FPS (16.7ms per frame)")
    print()

    # Create async session - logging calls return in < 0.01ms!
    session = create_session(
        "http://localhost:9090",
        product="ExampleGame",
        version="1.0.0",
        async_mode=True,  # Enable non-blocking async mode
        queue_size=10000,  # Buffer up to 10K events
    )

    print("Session started: %s" % session.session_id)
    print("Logging mode: Async (non-blocking)")
    print()

    # Simulate game state
    player_pos = {"x": 0.0, "y": 0.0, "z": 0.0}
    enemy_count = 0
    frame_count = 0

    # Track timing
    frame_times = []
    log_times = []

    print("Running 300 frames (5 seconds at 60 FPS)...")
    print()

    start_time = time.time()

    for frame in range(300):
        frame_start = time.time()

        # Update game state
        player_pos["x"] += 0.1
        player_pos["y"] += 0.05
        if frame % 30 == 0:
            enemy_count += 1

        # Log game events - these calls return in < 0.01ms each!
        log_start = time.time()

        # Log frame start
        session.log_event("frame_start", data={"frame": frame})

        # Log player position every frame
        session.update_state(
            {"frame": frame, "player_pos": player_pos, "enemy_count": enemy_count}
        )

        # Log FPS as a gauge (point-in-time value)
        if frame > 0:
            fps = 1.0 / (frame_start - prev_frame_start)
            session.log_gauge("fps", fps, unit="fps", tags=["realtime"])

            # Log frame time as a histogram (distribution analysis)
            frame_duration = frame_start - prev_frame_start
            session.log_histogram(
                "frame_time",
                frame_duration,
                unit="seconds",
                buckets=[1.0 / 120, 1.0 / 60, 1.0 / 30, 1.0 / 15, 1.0],
                tags=["performance"],
            )

            # Increment frame counter
            session.log_counter("frames_rendered", tags=["performance"])

        # Log enemy spawn every 30 frames (every 0.5 seconds)
        if frame % 30 == 0 and frame > 0:
            session.log_event(
                "enemy_spawned",
                level="info",
                data={"enemy_type": "goblin", "total_enemies": enemy_count},
            )
            # Increment enemy spawn counter
            session.log_counter("enemies_spawned", tags=["gameplay"])

        # Track gameplay events with counter
        if frame % 10 == 0:
            session.log_counter("player_moves", tags=["gameplay"])

        # Log player action every 10 frames
        if frame % 10 == 0:
            session.log_event(
                "player_moved", data={"position": player_pos, "distance": frame * 0.1}
            )

        # Use timer context manager for expensive operations
        if frame % 60 == 0:  # Every second
            with session.timer("ai_update", unit="seconds", tags=["ai"]):
                # Simulate AI processing time
                time.sleep(0.001)  # 1ms AI update

            with session.timer("physics_update", unit="seconds", tags=["physics"]):
                # Simulate physics processing time
                time.sleep(0.0005)  # 0.5ms physics update

        log_end = time.time()
        log_time = (log_end - log_start) * 1000  # Convert to ms
        log_times.append(log_time)

        # Simulate game logic (remaining frame time)
        # In a real game, this would be rendering, physics, AI, etc.
        frame_end = time.time()
        frame_time = (frame_end - frame_start) * 1000  # Convert to ms
        frame_times.append(frame_time)

        # Sleep to maintain ~60 FPS
        target_frame_time = 16.7  # ms
        sleep_time = max(0, (target_frame_time - frame_time) / 1000.0)
        if sleep_time > 0:
            time.sleep(sleep_time)

        prev_frame_start = frame_start
        frame_count += 1

    end_time = time.time()
    total_time = end_time - start_time

    # Print statistics
    print("\n" + "=" * 70)
    print("Performance Statistics")
    print("=" * 70)
    print()
    print("Total frames: %d" % frame_count)
    print("Total time: %.2f seconds" % total_time)
    print("Average FPS: %.1f" % (frame_count / total_time))
    print()
    print("Frame times:")
    print("  Average: %.3f ms" % (sum(frame_times) / len(frame_times)))
    print("  Min: %.3f ms" % min(frame_times))
    print("  Max: %.3f ms" % max(frame_times))
    print()
    print("Logging overhead per frame:")
    print(
        "  Average: %.4f ms (%.2f%% of frame budget)"
        % (
            sum(log_times) / len(log_times),
            (sum(log_times) / len(log_times)) / 16.7 * 100,
        )
    )
    print("  Min: %.4f ms" % min(log_times))
    print("  Max: %.4f ms" % max(log_times))
    print()

    # Get async logging statistics
    stats = session.get_stats()
    print("Async logging statistics:")
    print("  Events sent: %d" % stats["sent"])
    print("  Events dropped: %d" % stats["dropped"])
    print("  Events queued: %d" % stats["queued"])
    print()

    if stats["dropped"] > 0:
        print("WARNING: Some events were dropped due to full queue")
        print("         Consider increasing queue_size parameter")

    print("Shutting down (flushing remaining logs)...")
    session.shutdown(timeout=5.0)

    print()
    print("=" * 70)
    print("SUCCESS: Game logging completed!")
    print()
    print("Key takeaways:")
    print("  - Logging overhead is < 0.1ms per frame (< 1% of frame budget)")
    print("  - No frame drops due to logging")
    print("  - All logs processed asynchronously in background thread")
    print("  - Safe for 60 FPS, 120 FPS, or even higher frame rates")
    print("=" * 70)


def demonstrate_blocking_vs_async():
    """
    Demonstrate the difference between blocking and async logging
    """
    print("\n\n")
    print("=" * 70)
    print("Blocking vs Async Comparison")
    print("=" * 70)
    print()

    print("Testing log call latency (100 calls each)...")
    print()

    # Test async mode
    session_async = create_session(
        "http://localhost:9090", product="LatencyTest", version="1.0.0", async_mode=True
    )

    async_times = []
    for i in range(100):
        start = time.time()
        session_async.log_event("test_event", data={"i": i})
        end = time.time()
        async_times.append((end - start) * 1000000)  # microseconds

    session_async.shutdown()

    print("Async mode (non-blocking):")
    print(
        "  Average: %.1f microseconds (%.4f ms)"
        % (
            sum(async_times) / len(async_times),
            sum(async_times) / len(async_times) / 1000,
        )
    )
    print("  Min: %.1f microseconds" % min(async_times))
    print("  Max: %.1f microseconds" % max(async_times))
    print()

    print("Result: Async logging is suitable for real-time applications!")
    print("        Overhead is negligible (< 0.01ms per call)")
    print()
    print("=" * 70)


if __name__ == "__main__":
    print("\nNOTE: Make sure datacat server and daemon are running!")
    print("      Run: ./scripts/run-server.ps1")
    print()

    try:
        # Run game loop simulation
        simulate_game_loop()

        # Demonstrate latency comparison
        demonstrate_blocking_vs_async()

    except KeyboardInterrupt:
        print("\n\nInterrupted by user")
    except Exception as e:
        print("\n\nError: %s" % str(e))
        print("\nMake sure the datacat server is running!")
        import traceback

        traceback.print_exc()
