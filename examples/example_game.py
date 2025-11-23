#!/usr/bin/env python
"""
Example Game - Demonstrates DataCat logging in a game-like application

This simulates a simple game with:
- Main update/render loop running at 60 FPS
- Random events (enemies, powerups, achievements, errors)
- Random errors and exceptions
- Different modes: normal, hang, crash

Demonstrates ALL DataCat metric types:
- GAUGE: fps, memory_mb, player_health, player_score (values that go up/down)
- COUNTER: frames_rendered, enemies_encountered (monotonically increasing)
- HISTOGRAM: fps_distribution with custom buckets (60+, 30-60, 20-30, 10-20, <10 FPS)
- TIMER: frame_render_time (auto-measured duration)

Usage:
    python example_game.py --mode normal --duration 60
    python example_game.py --mode hang --duration 30
    python example_game.py --mode crash --duration 20
"""

from __future__ import print_function
import sys
import os
import time
import random
import argparse

# Add parent directory to path for datacat import
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'python'))

from datacat import create_session

# Try to import psutil for real memory metrics, fall back to fake data
try:
    import psutil
    HAVE_PSUTIL = True
except ImportError:
    HAVE_PSUTIL = False


class GameSimulator(object):
    """Simulates a game with realistic logging patterns"""

    def __init__(self, player_name, mode='normal', duration=None, use_async=True):
        """
        Initialize game simulator

        Args:
            player_name (str): Player/session identifier
            mode (str): Game mode - 'normal', 'hang', or 'crash'
            duration (int): How long to run (seconds), None = infinite
            use_async (bool): Use async logging mode
        """
        self.player_name = player_name
        self.mode = mode
        self.duration = duration
        self.use_async = use_async

        # Game state
        self.frame_count = 0
        self.player_health = 100
        self.player_score = 0
        self.player_x = 0.0
        self.player_y = 0.0
        self.enemies_killed = 0
        self.powerups_collected = 0
        self.level = 1

        # Timing
        self.start_time = None
        self.last_fps_update = None
        self.frame_times = []

        # Counter tracking
        self.total_enemies_encountered = 0

        # FPS histogram buckets - optimized for different frame rates
        # Buckets: 60+ FPS, 30-60 FPS, 20-30 FPS, 10-20 FPS, <10 FPS
        self.fps_buckets = [
            10.0,   # <10 FPS (unplayable)
            20.0,   # 10-20 FPS (very poor)
            30.0,   # 20-30 FPS (poor)
            60.0,   # 30-60 FPS (acceptable)
            1000.0  # 60+ FPS (excellent) - effectively infinity
        ]

        # Create session
        print("[%s] Creating session (mode=%s, duration=%s, async=%s)..." %
              (player_name, mode, duration, use_async))

        self.session = create_session(
            "http://localhost:9090",
            product="ExampleGame",
            version="1.0.0",
            async_mode=use_async,
            queue_size=20000  # Large queue for high-frequency logging
        )

        print("[%s] Session created: %s" % (player_name, self.session.session_id))

        # Log initial state
        self.session.update_state({
            'player_name': player_name,
            'mode': mode,
            'health': self.player_health,
            'score': self.player_score,
            'level': self.level,
            'position': {'x': self.player_x, 'y': self.player_y}
        })

        self.session.log_event(
            'game_started',
            level='info',
            data={
                'player': player_name,
                'mode': mode,
                'async_logging': use_async
            }
        )

    def get_memory_mb(self):
        """Get current memory usage in MB"""
        if HAVE_PSUTIL:
            process = psutil.Process(os.getpid())
            return process.memory_info().rss / 1024 / 1024
        else:
            # Fake memory that increases slowly
            return 50.0 + (self.frame_count * 0.001)

    def get_fps(self):
        """Calculate current FPS"""
        if len(self.frame_times) < 2:
            return 60.0

        recent_times = self.frame_times[-60:]  # Last 60 frames
        if len(recent_times) < 2:
            return 60.0

        total_time = recent_times[-1] - recent_times[0]
        if total_time > 0:
            return len(recent_times) / total_time
        return 60.0

    def update_game_logic(self):
        """Update game state (simulated)"""
        # Move player
        self.player_x += random.uniform(-1.0, 1.0)
        self.player_y += random.uniform(-1.0, 1.0)

        # Random events
        rand = random.random()

        if rand < 0.01:  # 1% chance - enemy encounter
            damage = random.randint(5, 15)
            self.player_health -= damage
            self.total_enemies_encountered += 1

            # Log counter for enemies encountered
            self.session.log_counter('enemies_encountered', tags=['gameplay'])

            self.session.log_event(
                'enemy_encountered',
                level='warning',
                data={
                    'damage': damage,
                    'remaining_health': self.player_health,
                    'total_encountered': self.total_enemies_encountered
                }
            )

            if random.random() < 0.7:  # 70% chance to kill enemy
                self.enemies_killed += 1
                score_gain = random.randint(10, 50)
                self.player_score += score_gain
                self.session.log_event(
                    'enemy_killed',
                    level='info',
                    data={
                        'score_gain': score_gain,
                        'total_killed': self.enemies_killed
                    }
                )

        elif rand < 0.02:  # 1% chance - collect powerup
            self.powerups_collected += 1
            health_gain = random.randint(10, 30)
            self.player_health = min(100, self.player_health + health_gain)
            self.session.log_event(
                'powerup_collected',
                level='info',
                data={
                    'type': random.choice(['health', 'shield', 'speed']),
                    'health_gain': health_gain,
                    'total_collected': self.powerups_collected
                }
            )

        elif rand < 0.025:  # 0.5% chance - level up
            self.level += 1
            self.session.log_event(
                'level_up',
                level='info',
                data={
                    'new_level': self.level,
                    'score': self.player_score
                }
            )
            self.session.update_state({'level': self.level})

        elif rand < 0.03:  # 0.5% chance - achievement
            achievements = [
                'first_blood',
                'speed_demon',
                'treasure_hunter',
                'survivor',
                'combo_master'
            ]
            achievement = random.choice(achievements)
            self.session.log_event(
                'achievement_unlocked',
                level='info',
                data={
                    'achievement': achievement,
                    'score_bonus': 100
                }
            )
            self.player_score += 100

        # Random errors (rare)
        if random.random() < 0.005:  # 0.5% chance
            error_types = [
                'texture_load_failed',
                'sound_playback_error',
                'network_timeout',
                'save_file_corrupted'
            ]
            self.session.log_event(
                'game_error',
                level='error',
                data={
                    'error_type': random.choice(error_types),
                    'recoverable': True
                }
            )

        # Random exceptions (very rare)
        if random.random() < 0.001:  # 0.1% chance
            try:
                # Simulate an exception
                if random.random() < 0.5:
                    raise ValueError("Invalid game state detected")
                else:
                    raise RuntimeError("Asset loading failed")
            except Exception:
                self.session.log_exception(
                    extra_data={
                        'context': 'game_update',
                        'frame': self.frame_count
                    }
                )

        # Check for death
        if self.player_health <= 0:
            self.session.log_event(
                'player_died',
                level='warning',
                data={
                    'frame': self.frame_count,
                    'score': self.player_score,
                    'enemies_killed': self.enemies_killed
                }
            )
            return False  # Game over

        return True  # Continue

    def render_frame(self):
        """Simulate rendering (just sleep to maintain FPS)"""
        # Use TIMER to measure render time every frame
        # This is a histogram under the hood, auto-measuring duration
        with self.session.timer('frame_render_time', unit='seconds', tags=['performance']):
            # Simulate some render work
            time.sleep(random.uniform(0.001, 0.003))  # 1-3ms render time

        # Log render event every 60 frames (once per second @ 60 FPS)
        if self.frame_count % 60 == 0:
            self.session.log_event(
                'frame_rendered',
                level='debug',
                data={
                    'frame': self.frame_count,
                    'position': {'x': self.player_x, 'y': self.player_y}
                }
            )

    def log_metrics(self):
        """Log game metrics - demonstrates all metric types"""
        # Log metrics every 30 frames (twice per second @ 60 FPS)
        if self.frame_count % 30 == 0:
            fps = self.get_fps()
            memory_mb = self.get_memory_mb()

            # GAUGE metrics - current values that can go up or down
            self.session.log_gauge('fps', fps, unit='fps', tags=['performance'])
            self.session.log_gauge('memory_mb', memory_mb, unit='MB', tags=['performance'])
            self.session.log_gauge('player_health', float(self.player_health), unit='hp', tags=['gameplay'])
            self.session.log_gauge('player_score', float(self.player_score), tags=['gameplay'])

            # HISTOGRAM - FPS distribution with custom buckets
            # This lets us analyze what percentage of frames hit different FPS targets
            self.session.log_histogram('fps_distribution', fps,
                                      unit='fps',
                                      buckets=self.fps_buckets,
                                      tags=['performance'])

            # COUNTER - total frames rendered (monotonically increasing)
            # Daemon automatically accumulates these
            self.session.log_counter('frames_rendered', tags=['performance'])

        # Update state every 120 frames (every 2 seconds @ 60 FPS)
        if self.frame_count % 120 == 0:
            self.session.update_state({
                'health': self.player_health,
                'score': self.player_score,
                'level': self.level,
                'position': {'x': round(self.player_x, 2), 'y': round(self.player_y, 2)},
                'stats': {
                    'enemies_killed': self.enemies_killed,
                    'powerups_collected': self.powerups_collected,
                    'frames': self.frame_count
                }
            })

    def should_hang(self):
        """Determine if game should hang (for hang mode)"""
        if self.mode != 'hang':
            return False

        # Hang after 50% of duration
        if self.duration:
            elapsed = time.time() - self.start_time
            return elapsed > (self.duration * 0.5)

        return False

    def should_crash(self):
        """Determine if game should crash (for crash mode)"""
        if self.mode != 'crash':
            return False

        # Crash after 75% of duration
        if self.duration:
            elapsed = time.time() - self.start_time
            return elapsed > (self.duration * 0.75)

        return False

    def run(self):
        """Run the game loop"""
        print("[%s] Starting game loop..." % self.player_name)

        self.start_time = time.time()
        self.last_fps_update = self.start_time
        running = True

        try:
            while running:
                frame_start = time.time()
                self.frame_count += 1
                self.frame_times.append(frame_start)

                # Keep only last 120 frame times
                if len(self.frame_times) > 120:
                    self.frame_times = self.frame_times[-120:]

                # Update game state
                running = self.update_game_logic()

                # Render frame
                self.render_frame()

                # Log metrics
                self.log_metrics()

                # Check for hang mode
                if self.should_hang():
                    print("[%s] Entering hang state (simulating freeze)..." % self.player_name)
                    self.session.log_event(
                        'game_hanging',
                        level='error',
                        data={'reason': 'simulated_hang', 'frame': self.frame_count}
                    )
                    # Stop sending logs but keep process alive
                    while True:
                        time.sleep(1)  # Hang forever

                # Check for crash mode
                if self.should_crash():
                    print("[%s] Simulating crash..." % self.player_name)
                    self.session.log_event(
                        'game_crashing',
                        level='error',
                        data={'reason': 'simulated_crash', 'frame': self.frame_count}
                    )
                    if self.use_async:
                        self.session.flush(timeout=1.0)  # Try to send crash log
                    # Crash hard
                    raise RuntimeError("Simulated game crash!")

                # Check duration limit
                if self.duration:
                    elapsed = time.time() - self.start_time
                    if elapsed >= self.duration:
                        print("[%s] Duration limit reached (%.1fs)" % (self.player_name, elapsed))
                        running = False

                # Maintain ~60 FPS
                frame_time = time.time() - frame_start
                target_frame_time = 1.0 / 60.0  # 16.7ms
                sleep_time = target_frame_time - frame_time
                if sleep_time > 0:
                    time.sleep(sleep_time)

                # Print status every second
                if self.frame_count % 60 == 0:
                    elapsed = time.time() - self.start_time
                    fps = self.get_fps()
                    print("[%s] Frame %d | Time: %.1fs | FPS: %.1f | HP: %d | Score: %d" %
                          (self.player_name, self.frame_count, elapsed, fps,
                           self.player_health, self.player_score))

        except KeyboardInterrupt:
            print("\n[%s] Interrupted by user" % self.player_name)

        finally:
            # Shutdown
            print("[%s] Shutting down..." % self.player_name)
            elapsed = time.time() - self.start_time

            self.session.log_event(
                'game_ended',
                level='info',
                data={
                    'duration_seconds': elapsed,
                    'total_frames': self.frame_count,
                    'final_score': self.player_score,
                    'enemies_killed': self.enemies_killed,
                    'enemies_encountered': self.total_enemies_encountered,
                    'powerups_collected': self.powerups_collected,
                    'exit_reason': 'normal'
                }
            )

            # Get stats if async
            if self.use_async:
                stats = self.session.get_stats()
                print("[%s] Async stats: sent=%d, dropped=%d, queued=%d" %
                      (self.player_name, stats['sent'], stats['dropped'], stats['queued']))
                self.session.shutdown(timeout=5.0)
            else:
                self.session.end()

            print("[%s] Game ended after %.1fs (%d frames)" %
                  (self.player_name, elapsed, self.frame_count))


def main():
    parser = argparse.ArgumentParser(description='Example game with DataCat logging')
    parser.add_argument(
        '--player',
        type=str,
        default='Player_%d' % random.randint(1000, 9999),
        help='Player name/identifier'
    )
    parser.add_argument(
        '--mode',
        type=str,
        choices=['normal', 'hang', 'crash'],
        default='normal',
        help='Game mode (default: normal)'
    )
    parser.add_argument(
        '--duration',
        type=int,
        default=None,
        help='How long to run in seconds (default: infinite)'
    )
    parser.add_argument(
        '--no-async',
        action='store_true',
        help='Disable async logging (use blocking mode)'
    )

    args = parser.parse_args()

    print("=" * 70)
    print("Example Game - DataCat Logging Demo")
    print("=" * 70)
    print()
    print("Player: %s" % args.player)
    print("Mode: %s" % args.mode)
    print("Duration: %s" % (("%d seconds" % args.duration) if args.duration else "infinite"))
    print("Async logging: %s" % (not args.no_async))
    print()
    print("Press Ctrl+C to stop")
    print()

    game = GameSimulator(
        player_name=args.player,
        mode=args.mode,
        duration=args.duration,
        use_async=not args.no_async
    )

    game.run()


if __name__ == '__main__':
    main()

