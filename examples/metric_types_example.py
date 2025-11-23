#!/usr/bin/env python
"""
Example: Using different metric types (gauges, counters, histograms, timers)

This example demonstrates the four different metric types supported by datacat:
- Gauges: Point-in-time values (CPU%, memory, temperature)
- Counters: Monotonically increasing values (requests, errors, bytes)
- Histograms: Value distributions (latencies, sizes)
- Timers: Duration measurements (execution time, with optional iteration count)
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
    print("=" * 70)
    print("Datacat Metric Types Example")
    print("=" * 70)
    print()

    # Create session
    session = create_session(
        "http://localhost:9090", product="MetricTypesExample", version="1.0.0"
    )
    print("Created session:", session.session_id)
    print()

    # =========================================================================
    # 1. GAUGES - Current values that can go up or down
    # =========================================================================
    print("1. GAUGES (current values)")
    print("-" * 70)

    # System metrics
    session.log_gauge("cpu_percent", 45.2, tags=["system"], unit="percent")
    session.log_gauge("memory_mb", 1024.5, tags=["system"], unit="megabytes")
    session.log_gauge("temperature_c", 72.3, tags=["hardware"], unit="celsius")

    # Application metrics
    session.log_gauge("active_connections", 42, tags=["network"])
    session.log_gauge("queue_depth", 15, tags=["queue"])

    print("  * Logged gauge metrics (CPU, memory, temperature, connections, queue)")
    print()

    # =========================================================================
    # 2. COUNTERS - Incremental counts (daemon accumulates)
    # =========================================================================
    print("2. COUNTERS (incremental - daemon tracks totals)")
    print("-" * 70)

    # Just increment when events happen - daemon tracks the cumulative total!
    print("  Simulating 100 requests...")
    for i in range(100):
        session.log_counter("http_requests", tags=["method:GET"])
        if i % 10 == 0:  # 10% error rate
            session.log_counter("http_errors", tags=["type:500"])

    # Increment by specific amounts
    print("  Transferring files...")
    session.log_counter("bytes_sent", delta=1024 * 1024, tags=["network"])  # 1 MB
    session.log_counter("bytes_sent", delta=1024 * 512, tags=["network"])  # 512 KB

    print("  * Logged counter increments (daemon accumulates totals automatically)")
    print("  * No need to track totals in your code!")
    print()

    # =========================================================================
    # 3. HISTOGRAMS - Value distributions
    # =========================================================================
    print("3. HISTOGRAMS (value distributions)")
    print("-" * 70)

    # Log multiple samples for histogram
    print("  Recording 100 request latencies...")
    for i in range(100):
        # Simulate varying latencies (mostly fast, some slow)
        if i < 80:
            latency = random.uniform(0.01, 0.1)  # Most requests: 10-100ms
        elif i < 95:
            latency = random.uniform(0.1, 0.5)  # Some slower: 100-500ms
        else:
            latency = random.uniform(0.5, 2.0)  # Few outliers: 500ms-2s

        session.log_histogram(
            "request_latency",
            latency,
            tags=["endpoint:/api/users"],
            metadata={"request_id": i},
        )

    print("  * Logged 100 histogram samples (can calculate percentiles)")
    print()

    # =========================================================================
    # 4. TIMERS - Measure execution time
    # =========================================================================
    print("4. TIMERS (duration measurements)")
    print("-" * 70)

    # Basic timer - no count
    print("  Example 1: Basic timer")
    with session.timer("load_config"):
        time.sleep(0.1)  # Simulate work
    print("    * Timed operation (simple duration)")

    # Timer with known count
    print("  Example 2: Timer with count")
    textures = list(range(50))
    with session.timer("load_textures", count=len(textures), unit="seconds"):
        for texture in textures:
            time.sleep(0.001)  # Simulate loading each texture
    print("    * Timed operation with count (can calculate avg time per item)")

    # Timer with incremental count
    print("  Example 3: Timer with incremental count")
    with session.timer("process_queue", unit="milliseconds") as timer:
        items_to_process = 30
        for i in range(items_to_process):
            timer.count += 1
            time.sleep(0.002)  # Simulate processing
    print("    * Timed operation with dynamic count")

    # Multiple timed operations
    print("  Example 4: Multiple timers")
    operations = ["parse_json", "validate_data", "save_to_db"]
    for op in operations:
        with session.timer(op, tags=["data_pipeline"]):
            time.sleep(random.uniform(0.05, 0.15))
    print("    * Multiple timed operations")

    print()

    # =========================================================================
    # Summary
    # =========================================================================
    print("=" * 70)
    print("Summary of Metric Types")
    print("=" * 70)
    print()
    print("GAUGE:")
    print("  - Current value (can increase or decrease)")
    print("  - Examples: CPU%, memory, temperature, active connections")
    print("  - Use for: Instant measurements")
    print()
    print("COUNTER:")
    print("  - Cumulative value (only increases)")
    print("  - Examples: Total requests, bytes sent, errors")
    print("  - Use for: Counting events over time")
    print("  - Note: Server can calculate rate (requests/second)")
    print()
    print("HISTOGRAM:")
    print("  - Distribution of values")
    print("  - Examples: Request latencies, file sizes")
    print("  - Use for: Understanding value spread (p50, p95, p99)")
    print("  - Note: Log many samples, analyze distribution later")
    print()
    print("TIMER:")
    print("  - Duration measurement")
    print("  - Examples: Function execution time, operation duration")
    print("  - Use for: Performance profiling")
    print("  - Features: Auto timing, optional iteration count")
    print()

    # End session
    session.end()
    print("Session ended")
    print()
    print("View session at: http://localhost:8080/session/" + session.session_id)


if __name__ == "__main__":
    main()
