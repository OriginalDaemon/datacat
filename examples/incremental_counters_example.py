#!/usr/bin/env python
"""
Example: Incremental Counters with Daemon-Side Aggregation

This example demonstrates the improved counter API where the daemon tracks
cumulative totals, so you don't need to maintain counters in your application.

Just call log_counter() whenever an event happens, and the daemon accumulates!
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
    print("Incremental Counters Example")
    print("=" * 70)
    print()

    session = create_session(
        "http://localhost:9090", product="IncrementalCounters", version="1.0.0"
    )
    print("Created session:", session.session_id)
    print()

    # =========================================================================
    # EXAMPLE 1: Simple Event Counting
    # =========================================================================
    print("Example 1: Simple Event Counting")
    print("-" * 70)
    print("Simulating a web server handling requests...")
    print()

    # NO NEED TO TRACK TOTALS!
    # Just call log_counter() when events happen
    for i in range(30):
        # Simulate handling requests
        requests_this_second = random.randint(5, 15)
        errors_this_second = random.randint(0, 2)

        for _ in range(requests_this_second):
            # Just increment! Daemon tracks the total
            session.log_counter("http_requests", tags=["method:GET"])

        for _ in range(errors_this_second):
            session.log_counter("http_errors", tags=["type:500"])

        # Log current rate as gauge for comparison
        session.log_gauge("requests_per_sec", requests_this_second, tags=["instant"])

        if i % 5 == 0:
            print(
                "  Second {}: {} requests, {} errors".format(
                    i, requests_this_second, errors_this_second
                )
            )

        time.sleep(0.1)  # Simulate time passing

    print()
    print("  * Daemon accumulated all counters automatically!")
    print("  * Server can calculate rate from cumulative totals")
    print()

    # =========================================================================
    # EXAMPLE 2: Byte Counting
    # =========================================================================
    print("Example 2: Byte Counting with Delta")
    print("-" * 70)
    print("Simulating file transfers...")
    print()

    files = [
        ("document.pdf", 1024 * 512),  # 512 KB
        ("image.jpg", 1024 * 256),  # 256 KB
        ("video.mp4", 1024 * 1024 * 5),  # 5 MB
        ("archive.zip", 1024 * 1024 * 2),  # 2 MB
    ]

    for filename, size_bytes in files:
        # Increment counter by the number of bytes
        session.log_counter(
            "bytes_transferred", delta=size_bytes, tags=["protocol:https"]
        )
        print(
            "  Transferred: {} ({:.2f} MB)".format(filename, size_bytes / 1024 / 1024)
        )
        time.sleep(0.05)

    print()
    print("  * Daemon accumulated total bytes automatically")
    print()

    # =========================================================================
    # EXAMPLE 3: Multi-threaded Scenario (Simulated)
    # =========================================================================
    print("Example 3: Concurrent Operations")
    print("-" * 70)
    print("Simulating multiple threads processing items...")
    print()

    # In real multi-threaded code, you'd just call from different threads
    # The daemon handles thread-safety automatically!
    operations = [
        ("Thread-1", 25),
        ("Thread-2", 30),
        ("Thread-3", 18),
        ("Thread-4", 22),
    ]

    for thread_name, item_count in operations:
        for _ in range(item_count):
            session.log_counter(
                "items_processed", tags=["worker:{}".format(thread_name)]
            )
        print("  {}: processed {} items".format(thread_name, item_count))

    print()
    print("  * Each thread just increments - daemon aggregates safely")
    print()

    # =========================================================================
    # EXAMPLE 4: Cache Statistics
    # =========================================================================
    print("Example 4: Cache Statistics")
    print("-" * 70)
    print("Simulating cache hits and misses...")
    print()

    for i in range(100):
        # 80% cache hit rate
        if random.random() < 0.8:
            session.log_counter("cache_hits", tags=["cache:redis"])
        else:
            session.log_counter("cache_misses", tags=["cache:redis"])

    print("  * Logged 100 cache accesses (~80 hits, ~20 misses)")
    print("  * Server can calculate hit rate from the counts")
    print()

    # =========================================================================
    # EXAMPLE 5: Comparison with Gauge
    # =========================================================================
    print("Example 5: Counter vs Gauge Comparison")
    print("-" * 70)
    print()

    total = 0
    for i in range(3):
        count_this_period = random.randint(5, 15)
        total += count_this_period

        # Counter: Daemon tracks cumulative total
        for _ in range(count_this_period):
            session.log_counter("cumulative_events")

        # Gauge: Just the current value (resets each time conceptually)
        session.log_gauge("events_this_period", count_this_period)

        print(
            "  Period {}: {} events (cumulative total: {})".format(
                i + 1, count_this_period, total
            )
        )
        time.sleep(0.2)

    print()
    print("  * Counter: Daemon maintains running total")
    print("  * Gauge: Each log is independent point-in-time value")
    print()

    # =========================================================================
    # Summary
    # =========================================================================
    print("=" * 70)
    print("Summary: Why Incremental Counters Are Better")
    print("=" * 70)
    print()
    print("OLD WAY (Manual Tracking):")
    print("  total_requests = 0")
    print("  def handle_request():")
    print("      total_requests += 1")
    print("      session.log_counter('requests', value=total_requests)")
    print()
    print("NEW WAY (Daemon Tracking):")
    print("  def handle_request():")
    print("      session.log_counter('requests')  # Just increment!")
    print()
    print("Benefits:")
    print("  * No need to maintain counter variables")
    print("  * Thread-safe by default (daemon handles concurrency)")
    print("  * Cleaner code - just log when events happen")
    print("  * Daemon accumulates and sends totals to server")
    print()

    session.end()
    print("Session ended")
    print()
    print("View session at: http://localhost:8080/session/" + session.session_id)


if __name__ == "__main__":
    main()
