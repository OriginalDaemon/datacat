#!/usr/bin/env python
"""
Example: FPS Histogram with Custom Buckets

This example demonstrates using custom histogram buckets to track frame times
in a game or graphics application, with buckets that correspond to meaningful
FPS thresholds: +60fps, 60fps, 30fps, 20fps, 10fps, <10fps.
"""

from __future__ import print_function
import sys
import os
import time
import random

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "python"))

from datacat import create_session


def simulate_frame(quality="normal"):
    """Simulate a frame render with varying performance"""
    if quality == "excellent":
        # +60 FPS (< 16.67ms per frame)
        return random.uniform(0.008, 0.016)
    elif quality == "good":
        # 30-60 FPS (16.67-33.33ms)
        return random.uniform(0.017, 0.032)
    elif quality == "acceptable":
        # 20-30 FPS (33.33-50ms)
        return random.uniform(0.033, 0.049)
    elif quality == "poor":
        # 10-20 FPS (50-100ms)
        return random.uniform(0.050, 0.099)
    elif quality == "bad":
        # <10 FPS (>100ms)
        return random.uniform(0.100, 0.300)
    else:  # normal - mixed
        roll = random.random()
        if roll < 0.70:
            return random.uniform(0.010, 0.016)  # Usually good (60+ fps)
        elif roll < 0.85:
            return random.uniform(0.017, 0.032)  # Sometimes 30-60 fps
        elif roll < 0.93:
            return random.uniform(0.033, 0.049)  # Occasionally 20-30 fps
        elif roll < 0.98:
            return random.uniform(0.050, 0.099)  # Rarely 10-20 fps
        else:
            return random.uniform(0.100, 0.300)  # Very rarely <10 fps


def main():
    print("=" * 70)
    print("FPS Histogram Example - Custom Buckets")
    print("=" * 70)
    print()

    session = create_session(
        "http://localhost:9090", product="GameEngine", version="1.0.0"
    )
    print("Created session:", session.session_id)
    print()

    # Define FPS buckets
    # Bucket boundaries are frame times (in seconds)
    # - +60fps: < 1/60 (0.0167s)
    # - 60fps: 1/60 to 1/30 (0.0167 to 0.0333s)
    # - 30fps: 1/30 to 1/20 (0.0333 to 0.0500s)
    # - 20fps: 1/20 to 1/10 (0.0500 to 0.1000s)
    # - <10fps: > 0.1000s (covered by 10.0 - way beyond any reasonable frame time)

    fps_buckets = [
        1.0/60.0,   # 0.0167s - anything less is +60 FPS
        1.0/30.0,   # 0.0333s - 30-60 FPS range
        1.0/20.0,   # 0.0500s - 20-30 FPS range
        1.0/10.0,   # 0.1000s - 10-20 FPS range
        10.0        # 10 seconds - anything higher is <10 FPS (values beyond last bucket are included)
    ]

    print("FPS Bucket Configuration:")
    print("  +60 FPS: frame_time <= {:.4f}s (16.67ms)".format(fps_buckets[0]))
    print("  60 FPS:  frame_time <= {:.4f}s (33.33ms)".format(fps_buckets[1]))
    print("  30 FPS:  frame_time <= {:.4f}s (50.00ms)".format(fps_buckets[2]))
    print("  20 FPS:  frame_time <= {:.4f}s (100.0ms)".format(fps_buckets[3]))
    print("  <10 FPS: frame_time > {:.4f}s".format(fps_buckets[3]))
    print()

    # =========================================================================
    # SCENARIO 1: Normal Mixed Performance
    # =========================================================================
    print("Scenario 1: Normal Mixed Performance (1000 frames)")
    print("-" * 70)
    print("Simulating typical gameplay with occasional frame drops...")
    print()

    for i in range(1000):
        frame_time = simulate_frame("normal")
        session.log_histogram(
            "frame_time",
            frame_time,
            unit="seconds",
            tags=["scenario:normal"],
            buckets=fps_buckets
        )

        if i % 200 == 0:
            fps = 1.0 / frame_time if frame_time > 0 else 0
            print("  Frame {}: {:.1f} FPS ({:.4f}s)".format(i, fps, frame_time))

    print()
    print("  * Daemon accumulated 1000 samples into 5 FPS buckets")
    print()

    # =========================================================================
    # SCENARIO 2: High-Performance Mode
    # =========================================================================
    print("Scenario 2: High-Performance Mode (500 frames)")
    print("-" * 70)
    print("Simulating optimal performance (low settings, simple scene)...")
    print()

    for i in range(500):
        frame_time = simulate_frame("excellent")
        session.log_histogram(
            "frame_time",
            frame_time,
            unit="seconds",
            tags=["scenario:high_performance"],
            buckets=fps_buckets
        )

    print("  * All frames should be in the +60 FPS bucket")
    print()

    # =========================================================================
    # SCENARIO 3: Performance Degradation
    # =========================================================================
    print("Scenario 3: Performance Degradation (500 frames)")
    print("-" * 70)
    print("Simulating performance drop (complex scene, high settings)...")
    print()

    qualities = [
        ("excellent", 100),
        ("good", 150),
        ("acceptable", 150),
        ("poor", 75),
        ("bad", 25),
    ]

    for quality, count in qualities:
        for i in range(count):
            frame_time = simulate_frame(quality)
            session.log_histogram(
                "frame_time",
                frame_time,
                unit="seconds",
                tags=["scenario:degradation"],
                buckets=fps_buckets
            )
        print("  * {} frames at '{}' quality".format(count, quality))

    print()
    print("  * Histogram shows distribution across all FPS buckets")
    print()

    # =========================================================================
    # SCENARIO 4: Different Graphics Settings
    # =========================================================================
    print("Scenario 4: Compare Graphics Settings (300 frames each)")
    print("-" * 70)

    settings = [
        ("low", "excellent"),
        ("medium", "good"),
        ("high", "acceptable"),
        ("ultra", "poor"),
    ]

    for setting_name, quality in settings:
        for i in range(300):
            frame_time = simulate_frame(quality)
            session.log_histogram(
                "frame_time",
                frame_time,
                unit="seconds",
                tags=["graphics:{}".format(setting_name)],
                buckets=fps_buckets
            )
        print("  * Logged 300 frames for '{}' settings".format(setting_name))

    print()
    print("  * Separate histograms per graphics setting (different tags)")
    print()

    # =========================================================================
    # SCENARIO 5: Real-World Example with Render Phases
    # =========================================================================
    print("Scenario 5: Render Phase Breakdown (200 frames)")
    print("-" * 70)
    print("Tracking frame time for different render phases...")
    print()

    phases = ["physics", "ai", "rendering", "post_processing"]

    for frame_num in range(200):
        # Each phase has different performance characteristics
        for phase in phases:
            if phase == "physics":
                time_spent = random.uniform(0.001, 0.005)  # Fast
            elif phase == "ai":
                time_spent = random.uniform(0.002, 0.008)  # Fast
            elif phase == "rendering":
                time_spent = random.uniform(0.008, 0.015)  # Main work
            else:  # post_processing
                time_spent = random.uniform(0.001, 0.003)  # Fast

            session.log_histogram(
                "phase_time",
                time_spent,
                unit="seconds",
                tags=["phase:{}".format(phase)],
                buckets=fps_buckets
            )

    print("  * Logged timings for each render phase")
    print("  * Can identify which phase is the bottleneck")
    print()

    # =========================================================================
    # SCENARIO 6: Default Buckets vs Custom Buckets
    # =========================================================================
    print("Scenario 6: Comparison - Default vs Custom Buckets (200 samples each)")
    print("-" * 70)
    print()

    for i in range(200):
        latency = random.uniform(0.001, 0.100)  # 1-100ms

        # Default buckets - good for general latency tracking
        session.log_histogram(
            "api_latency",
            latency,
            unit="seconds",
            tags=["bucket_type:default"]
            # No buckets parameter - uses default
        )

        # Custom FPS buckets - for frame time specific tracking
        session.log_histogram(
            "api_latency",
            latency,
            unit="seconds",
            tags=["bucket_type:fps_custom"],
            buckets=fps_buckets
        )

    print("  * Logged same data with default AND custom buckets")
    print("  * Default buckets: better for general percentile analysis")
    print("  * Custom buckets: better for domain-specific thresholds")
    print()

    # =========================================================================
    # Summary
    # =========================================================================
    print("=" * 70)
    print("Summary: Why Custom Histogram Buckets Matter")
    print("=" * 70)
    print()
    print("Benefits of Custom FPS Buckets:")
    print("  * Meaningful thresholds: 60fps, 30fps, etc.")
    print("  * Easy to interpret: count in each performance tier")
    print("  * Efficient: daemon aggregates, only bucket counts sent")
    print("  * Flexible: different buckets for different metrics")
    print()
    print("Use Cases:")
    print("  * FPS tracking: frame times with FPS-aligned buckets")
    print("  * Network latency: buckets for 'good', 'ok', 'bad' response times")
    print("  * File sizes: buckets for 'tiny', 'small', 'medium', 'large'")
    print("  * Any metric with meaningful threshold ranges")
    print()
    print("Default buckets are used when not specified - good for general use!")
    print()

    session.end()
    print("Session ended")
    print()
    print("View session at: http://localhost:8080/session/" + session.session_id)
    print()
    print("Check the histogram buckets in the metrics!")


if __name__ == "__main__":
    main()

