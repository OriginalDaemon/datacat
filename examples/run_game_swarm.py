#!/usr/bin/env python
"""
Run Game Swarm - Launch multiple game instances simultaneously

This launches multiple instances of example_game.py with different configurations
to simulate multiple players. Some will run normally, some will hang, and some
will crash - demonstrating DataCat's crash and hang detection capabilities.

Usage:
    python run_game_swarm.py --count 10 --duration 60
    python run_game_swarm.py --count 20 --duration 120 --hang-rate 0.2 --crash-rate 0.1
"""

from __future__ import print_function
import sys
import os
import subprocess
import time
import random
import argparse
import signal

# Track spawned processes
processes = []


def signal_handler(sig, frame):
    """Handle Ctrl+C gracefully"""
    print("\n\nShutting down all game instances...")
    for proc in processes:
        try:
            proc.terminate()
        except Exception:
            pass

    # Wait a bit for graceful shutdown
    time.sleep(2)

    # Force kill any remaining
    for proc in processes:
        try:
            proc.kill()
        except Exception:
            pass

    print("All instances stopped.")
    sys.exit(0)


def launch_game(player_name, mode, duration, no_async=False):
    """
    Launch a single game instance

    Args:
        player_name (str): Player identifier
        mode (str): Game mode ('normal', 'hang', 'crash')
        duration (int): How long to run (seconds)
        no_async (bool): Disable async logging

    Returns:
        subprocess.Popen: The spawned process
    """
    python_exe = sys.executable
    script_path = os.path.join(os.path.dirname(__file__), 'example_game.py')

    cmd = [
        python_exe,
        script_path,
        '--player', player_name,
        '--mode', mode,
        '--duration', str(duration)
    ]

    if no_async:
        cmd.append('--no-async')

    # Launch process
    proc = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        universal_newlines=True
    )

    return proc


def main():
    parser = argparse.ArgumentParser(description='Launch multiple game instances')
    parser.add_argument(
        '--count',
        type=int,
        default=10,
        help='Number of game instances to launch (default: 10)'
    )
    parser.add_argument(
        '--duration',
        type=int,
        default=60,
        help='How long each game runs in seconds (default: 60)'
    )
    parser.add_argument(
        '--hang-rate',
        type=float,
        default=0.15,
        help='Fraction of games that will hang (default: 0.15)'
    )
    parser.add_argument(
        '--crash-rate',
        type=float,
        default=0.15,
        help='Fraction of games that will crash (default: 0.15)'
    )
    parser.add_argument(
        '--stagger',
        type=float,
        default=1.0,
        help='Seconds to wait between launching instances (default: 1.0)'
    )
    parser.add_argument(
        '--no-async',
        action='store_true',
        help='Disable async logging for all instances'
    )

    args = parser.parse_args()

    # Validate rates
    if args.hang_rate + args.crash_rate > 1.0:
        print("ERROR: hang-rate + crash-rate cannot exceed 1.0")
        sys.exit(1)

    # Register signal handler for graceful shutdown
    signal.signal(signal.SIGINT, signal_handler)

    print("=" * 70)
    print("Game Swarm Launcher - DataCat Demo")
    print("=" * 70)
    print()
    print("Configuration:")
    print("  Instances: %d" % args.count)
    print("  Duration: %d seconds" % args.duration)
    print("  Hang rate: %.1f%%" % (args.hang_rate * 100))
    print("  Crash rate: %.1f%%" % (args.crash_rate * 100))
    print("  Normal rate: %.1f%%" % ((1.0 - args.hang_rate - args.crash_rate) * 100))
    print("  Stagger: %.1f seconds" % args.stagger)
    print("  Async logging: %s" % (not args.no_async))
    print()

    # Calculate how many of each type
    num_hang = int(args.count * args.hang_rate)
    num_crash = int(args.count * args.crash_rate)
    num_normal = args.count - num_hang - num_crash

    print("Expected outcomes:")
    print("  Normal exits: %d" % num_normal)
    print("  Hangs: %d" % num_hang)
    print("  Crashes: %d" % num_crash)
    print()
    print("Launching instances...")
    print("Press Ctrl+C to stop all instances")
    print()

    # Create list of modes
    modes = (
        ['normal'] * num_normal +
        ['hang'] * num_hang +
        ['crash'] * num_crash
    )
    random.shuffle(modes)

    # Launch instances
    for i in range(args.count):
        player_name = "Player_%03d" % (i + 1)
        mode = modes[i]

        # Add some randomness to duration (±10%)
        duration = args.duration + random.randint(-args.duration // 10, args.duration // 10)

        print("[%02d/%02d] Launching %s (mode=%s, duration=%ds)..." %
              (i + 1, args.count, player_name, mode, duration))

        proc = launch_game(player_name, mode, duration, args.no_async)
        processes.append(proc)

        # Stagger launches
        if i < args.count - 1:
            time.sleep(args.stagger)

    print()
    print("All instances launched!")
    print()
    print("=" * 70)
    print("Monitoring (this may take a while)...")
    print("=" * 70)
    print()

    # Monitor processes
    completed = 0
    start_time = time.time()

    while completed < args.count:
        time.sleep(1)

        # Check for completed processes
        for proc in processes:
            if proc.poll() is not None and not hasattr(proc, '_reported'):
                completed += 1
                proc._reported = True

                # Read output
                try:
                    output = proc.stdout.read()
                    if 'crash' in output.lower():
                        status = "CRASHED"
                    elif 'hang' in output.lower():
                        status = "HUNG"
                    else:
                        status = "COMPLETED"
                except Exception:
                    status = "UNKNOWN"

                elapsed = time.time() - start_time
                print("[%.1fs] Instance completed (%d/%d) - Status: %s" %
                      (elapsed, completed, args.count, status))

    print()
    print("=" * 70)
    print("All instances completed!")
    print("=" * 70)
    print()
    print("Total time: %.1f seconds" % (time.time() - start_time))
    print()
    print("Check the DataCat web UI at http://localhost:8080 to see:")
    print("  - All %d sessions" % args.count)
    print("  - Crash detection for crashed instances")
    print("  - Hang detection for hung instances")
    print("  - Live metrics and events")
    print()


if __name__ == '__main__':
    main()

