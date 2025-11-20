#!/usr/bin/env python
"""
datacat Demo GUI

A modern web-based GUI demonstrating all features of the datacat Python client:
- State changes with JSON editing
- Event logging
- Metrics tracking
- Exception handling with custom logging handler
- Error message logging

Install requirements: pip install gradio
Run: python demo_gui.py
"""

from __future__ import print_function
import sys
import os
import json
import logging
import traceback
import time
import random
from datetime import datetime

# Add the python directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "python"))

from datacat import create_session, DatacatClient, Session as DatacatSession

try:
    import gradio as gr
except ImportError:
    print("Error: Gradio is not installed.")
    print("Install it with: pip install gradio")
    sys.exit(1)


# Global session and client objects
session = None
session_info = {"created": False, "id": None}
datacat_client = None  # Reuse client to avoid multiple daemon instances


class DatacatLoggingHandler(logging.Handler):
    """
    Custom logging handler that sends log messages to datacat.

    Formats exceptions with full stack traces and sends them as events.
    """

    def __init__(self, datacat_session):
        super(DatacatLoggingHandler, self).__init__()
        self.datacat_session = datacat_session

    def emit(self, record):
        """Emit a log record to datacat"""
        try:
            # Format the log message
            log_entry = self.format(record)

            # Prepare event data
            event_data = {
                "level": record.levelname,
                "logger": record.name,
                "message": record.getMessage(),
                "timestamp": datetime.fromtimestamp(record.created).isoformat(),
            }

            # If there's exception info, include formatted traceback
            if record.exc_info:
                event_data["exception"] = {
                    "type": record.exc_info[0].__name__,
                    "message": str(record.exc_info[1]),
                    "traceback": traceback.format_exception(*record.exc_info)
                }

            # Add any extra fields
            if hasattr(record, 'extra_data'):
                event_data.update(record.extra_data)

            # Send to datacat
            event_name = "log_" + record.levelname.lower()
            if record.exc_info:
                event_name = "exception"

            self.datacat_session.log_event(event_name, event_data)

        except Exception:
            # Don't let logging errors crash the application
            self.handleError(record)


def toggle_dark_mode(is_dark):
    """Toggle dark mode theme"""
    print(f"DEBUG: Dark mode toggled to: {is_dark}")
    # Return a simple acknowledgment
    return is_dark


def toggle_session(server_url, use_daemon, product_name, product_version):
    """Toggle session - create if none exists, end if one exists"""
    global session, session_info, datacat_client

    # If session exists, end it
    if session and session_info["created"]:
        try:
            session_id = session.session_id
            session.log_event("demo_session_ended", {
                "ended_at": datetime.now().isoformat()
            })
            session.end()

            result = f"✅ Session ended successfully!\n\nSession ID: {session_id}"

            session = None
            session_info = {"created": False, "id": None}

            # Return updates: status, session_id, button_update, controls_enabled
            return (
                result,
                "N/A",
                gr.update(value="Start New Session", variant="primary"),
                gr.update(interactive=False),  # state_json
                gr.update(interactive=False),  # update_state_btn
                gr.update(interactive=False),  # event_name
                gr.update(interactive=False),  # event_data
                gr.update(interactive=False),  # log_event_btn
                gr.update(interactive=False),  # metric_name
                gr.update(interactive=False),  # metric_value
                gr.update(interactive=False),  # metric_tags
                gr.update(interactive=False),  # log_metric_btn
                gr.update(interactive=False),  # error_level
                gr.update(interactive=False),  # error_message
                gr.update(interactive=False),  # log_error_btn
                gr.update(interactive=False),  # exception_type
                gr.update(interactive=False),  # include_nested
                gr.update(interactive=False),  # generate_exception_btn
                gr.update(interactive=False),  # refresh_stats_btn
            )
        except Exception as e:
            return (
                f"❌ Error ending session: {str(e)}",
                session_info.get("id", "N/A"),
                gr.update(value="End Session", variant="stop"),
            ) + tuple([gr.update(interactive=True)] * 16)

    # Otherwise, create a new session
    # Validate inputs
    if not product_name or not product_name.strip():
        return (
            "❌ Product name is required",
            "N/A",
            gr.update(value="Start New Session", variant="primary"),
        ) + tuple([gr.update(interactive=False)] * 16)

    if not product_version or not product_version.strip():
        return (
            "❌ Product version is required",
            "N/A",
            gr.update(value="Start New Session", variant="primary"),
        ) + tuple([gr.update(interactive=False)] * 16)

    product_name = product_name.strip()
    product_version = product_version.strip()

    try:
        # Create or reuse the datacat client
        # This is important for daemon mode - we want to reuse the same daemon process
        # across multiple sessions to avoid port conflicts
        if datacat_client is None:
            datacat_client = DatacatClient(server_url, use_daemon=use_daemon, daemon_port="8079")

        # Register a new session with the existing client (product and version required)
        session_id = datacat_client.register_session(product_name, product_version)

        # Create Session object
        session = DatacatSession(datacat_client, session_id)
        # Update session info
        session_info["created"] = True
        session_info["id"] = session.session_id

        # Set initial state with user-provided product info
        session.update_state({
            "application": {
                "name": product_name,
                "version": product_version,
                "started_at": datetime.now().isoformat()
            },
            "statistics": {
                "events_sent": 0,
                "metrics_sent": 0,
                "states_updated": 0,
                "errors_logged": 0
            }
        })

        result = (
            f"✅ Session Created!\n\n"
            f"Product: {product_name} v{product_version}\n"
            f"Session ID: {session.session_id}\n\n"
            f"View at: http://localhost:8080/session/{session.session_id}"
        )

        # Return updates: status, session_id, button_update, controls_enabled
        return (
            result,
            session.session_id,
            gr.update(value="End Session", variant="stop"),
            gr.update(interactive=True),  # state_json
            gr.update(interactive=True),  # update_state_btn
            gr.update(interactive=True),  # event_name
            gr.update(interactive=True),  # event_data
            gr.update(interactive=True),  # log_event_btn
            gr.update(interactive=True),  # metric_name
            gr.update(interactive=True),  # metric_value
            gr.update(interactive=True),  # metric_tags
            gr.update(interactive=True),  # log_metric_btn
            gr.update(interactive=True),  # error_level
            gr.update(interactive=True),  # error_message
            gr.update(interactive=True),  # log_error_btn
            gr.update(interactive=True),  # exception_type
            gr.update(interactive=True),  # include_nested
            gr.update(interactive=True),  # generate_exception_btn
            gr.update(interactive=True),  # refresh_stats_btn
        )
    except Exception as e:
        return (
            f"❌ Error creating session: {str(e)}",
            "N/A",
            gr.update(value="Start New Session", variant="primary"),
        ) + tuple([gr.update(interactive=False)] * 16)


def update_state(state_json):
    """Update session state with JSON"""
    global session

    if not session:
        return "❌ No active session. Create a session first."

    try:
        state = json.loads(state_json)
        session.update_state(state)

        # Update statistics
        details = session.get_details()
        stats = details["state"].get("statistics", {})
        stats["states_updated"] = stats.get("states_updated", 0) + 1
        session.update_state({"statistics": stats})

        return f"✅ State updated successfully!\n\nCurrent state:\n{json.dumps(details['state'], indent=2)}"
    except json.JSONDecodeError as e:
        return f"❌ Invalid JSON: {str(e)}"
    except Exception as e:
        return f"❌ Error updating state: {str(e)}"


def log_event(event_name, event_data_json):
    """Log an event"""
    global session

    if not session:
        return "❌ No active session. Create a session first."

    try:
        event_data = {}
        if event_data_json.strip():
            event_data = json.loads(event_data_json)

        session.log_event(event_name, event_data)

        # Update statistics
        details = session.get_details()
        stats = details["state"].get("statistics", {})
        stats["events_sent"] = stats.get("events_sent", 0) + 1
        session.update_state({"statistics": stats})

        return f"✅ Event '{event_name}' logged successfully!\n\nData: {json.dumps(event_data, indent=2)}"
    except json.JSONDecodeError as e:
        return f"❌ Invalid JSON: {str(e)}"
    except Exception as e:
        return f"❌ Error logging event: {str(e)}"


def log_metric(metric_name, metric_value, metric_tags):
    """Log a metric"""
    global session

    if not session:
        return "❌ No active session. Create a session first."

    try:
        value = float(metric_value)
        tags = [tag.strip() for tag in metric_tags.split(",") if tag.strip()]

        session.log_metric(metric_name, value, tags=tags)

        # Update statistics
        details = session.get_details()
        stats = details["state"].get("statistics", {})
        stats["metrics_sent"] = stats.get("metrics_sent", 0) + 1
        session.update_state({"statistics": stats})

        return f"✅ Metric logged successfully!\n\nName: {metric_name}\nValue: {value}\nTags: {tags}"
    except ValueError:
        return "❌ Metric value must be a number"
    except Exception as e:
        return f"❌ Error logging metric: {str(e)}"


def log_error_message(error_level, error_message):
    """Log an error message using the custom logging handler"""
    global session

    if not session:
        return "❌ No active session. Create a session first."

    try:
        # Create logger with custom handler
        logger = logging.getLogger("datacat_demo")
        logger.handlers = []  # Clear existing handlers
        logger.setLevel(logging.DEBUG)

        # Add our custom handler
        handler = DatacatLoggingHandler(session)
        formatter = logging.Formatter(
            '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        handler.setFormatter(formatter)
        logger.addHandler(handler)

        # Log the message at the appropriate level
        level_map = {
            "DEBUG": logger.debug,
            "INFO": logger.info,
            "WARNING": logger.warning,
            "ERROR": logger.error,
            "CRITICAL": logger.critical
        }

        log_func = level_map.get(error_level, logger.error)
        log_func(error_message)

        # Update statistics
        details = session.get_details()
        stats = details["state"].get("statistics", {})
        stats["errors_logged"] = stats.get("errors_logged", 0) + 1
        session.update_state({"statistics": stats})

        return f"✅ Error logged via logging handler!\n\nLevel: {error_level}\nMessage: {error_message}"
    except Exception as e:
        return f"❌ Error logging message: {str(e)}"


def generate_exception(exception_type, include_nested):
    """Generate and log a Python exception with full stack trace"""
    global session

    if not session:
        return "❌ No active session. Create a session first."

    try:
        # Create logger with custom handler
        logger = logging.getLogger("datacat_demo")
        logger.handlers = []
        logger.setLevel(logging.DEBUG)

        handler = DatacatLoggingHandler(session)
        formatter = logging.Formatter(
            '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        handler.setFormatter(formatter)
        logger.addHandler(handler)

        exception_generated = False
        exception_info = ""

        try:
            if include_nested:
                # Generate nested exception
                try:
                    if exception_type == "ValueError":
                        raise ValueError("Inner exception: Invalid value provided")
                    elif exception_type == "TypeError":
                        raise TypeError("Inner exception: Type mismatch")
                    elif exception_type == "KeyError":
                        raise KeyError("Inner exception: Missing required key")
                    elif exception_type == "IndexError":
                        raise IndexError("Inner exception: List index out of range")
                    elif exception_type == "ZeroDivisionError":
                        raise ZeroDivisionError("Inner exception: Division by zero")
                except Exception as inner_e:
                    # Re-raise with context
                    raise RuntimeError(f"Outer exception wrapping: {type(inner_e).__name__}") from inner_e
            else:
                # Generate single exception
                if exception_type == "ValueError":
                    int("not_a_number")  # Will raise ValueError
                elif exception_type == "TypeError":
                    "string" + 123  # Will raise TypeError
                elif exception_type == "KeyError":
                    {}["missing_key"]  # Will raise KeyError
                elif exception_type == "IndexError":
                    [][999]  # Will raise IndexError
                elif exception_type == "ZeroDivisionError":
                    1 / 0  # Will raise ZeroDivisionError

        except Exception as e:
            exception_generated = True
            exception_info = f"Exception Type: {type(e).__name__}\nMessage: {str(e)}"

            # Log via logging handler (captures full traceback)
            logger.error("Exception occurred in demo", exc_info=True)

            # Also log directly via datacat for comparison
            session.log_exception(extra_data={
                "demo_exception": True,
                "exception_type": exception_type,
                "nested": include_nested,
                "generated_at": datetime.now().isoformat()
            })

        # Update statistics
        details = session.get_details()
        stats = details["state"].get("statistics", {})
        stats["errors_logged"] = stats.get("errors_logged", 0) + 1
        session.update_state({"statistics": stats})

        if exception_generated:
            nested_info = " (nested)" if include_nested else ""
            return (
                f"✅ Exception generated and logged!{nested_info}\n\n"
                f"{exception_info}\n\n"
                f"The exception was logged via:\n"
                f"1. Custom logging handler (with formatted traceback)\n"
                f"2. Direct session.log_exception() call\n\n"
                f"Check the session to see both formats!"
            )
        else:
            return "❌ Failed to generate exception"

    except Exception as e:
        return f"❌ Error in exception generation: {str(e)}"


def get_session_stats():
    """Get current session statistics"""
    global session

    if not session:
        return "❌ No active session. Create a session first."

    try:
        details = session.get_details()
        stats = details["state"].get("statistics", {})

        return (
            f"📊 Session Statistics\n\n"
            f"Session ID: {session.session_id}\n\n"
            f"Events Sent: {stats.get('events_sent', 0)}\n"
            f"Metrics Sent: {stats.get('metrics_sent', 0)}\n"
            f"State Updates: {stats.get('states_updated', 0)}\n"
            f"Errors Logged: {stats.get('errors_logged', 0)}\n\n"
            f"Total Events in Session: {len(details.get('events', []))}\n"
            f"Total Metrics in Session: {len(details.get('metrics', []))}\n"
            f"Total State History: {len(details.get('state_history', []))}\n\n"
            f"View full session at:\n"
            f"http://localhost:8080/session/{session.session_id}"
        )
    except Exception as e:
        return f"❌ Error getting stats: {str(e)}"




# JavaScript to enable Gradio's built-in dark mode on startup
ENABLE_DARK_MODE_JS = """
function() {
    // Enable Gradio's native dark mode
    document.body.classList.add('dark');
    return [];
}
"""


# Build the Gradio interface
def create_ui():
    with gr.Blocks(
        title="DataCat Demo GUI",
        theme=gr.themes.Soft(
            primary_hue="blue",
            secondary_hue="slate",
        ),
        js=ENABLE_DARK_MODE_JS
    ) as demo:
        gr.Markdown(
            """
            # 🐱 DataCat Demo GUI

            A modern demonstration of the datacat Python client features.
            """
        )

        gr.Markdown(
            """
            **Getting Started:**
            1. Enter your product name and version
            2. Click "Start New Session"
            3. Explore all features (automatically enabled)

            **Features:**
            - Create and manage sessions with product metadata
            - Send state updates with JSON
            - Log events and metrics
            - Log errors via custom logging handler
            - Generate and handle Python exceptions with full stack traces

            **Tip:** Use Gradio's built-in theme switcher (⚙️ Settings) to toggle light/dark mode
            """
        )

        with gr.Row():
            with gr.Column(scale=2):
                # Session Management
                gr.Markdown("## 🔧 Session Management")

                gr.Markdown("**Product Information** (Required)")
                with gr.Row():
                    product_name = gr.Textbox(
                        label="Product Name",
                        placeholder="my-application",
                        value="",
                        info="Enter your application/product name"
                    )
                    product_version = gr.Textbox(
                        label="Product Version",
                        placeholder="1.0.0",
                        value="",
                        info="Enter your product version"
                    )

                gr.Markdown("**Connection Settings**")
                with gr.Row():
                    server_url = gr.Textbox(
                        label="Server URL",
                        value="http://localhost:9090",
                        placeholder="http://localhost:9090"
                    )
                    use_daemon = gr.Checkbox(
                        label="Use Daemon",
                        value=False
                    )

                session_toggle_btn = gr.Button("Start New Session", variant="primary", size="lg")

                session_output = gr.Textbox(
                    label="Session Status",
                    lines=4,
                    interactive=False
                )

                current_session = gr.Textbox(
                    label="Current Session ID",
                    value="N/A",
                    interactive=False
                )

            with gr.Column(scale=1):
                gr.Markdown("## 📊 Statistics")
                stats_output = gr.Textbox(
                    label="Session Statistics",
                    lines=15,
                    interactive=False
                )
                refresh_stats_btn = gr.Button("Refresh Stats", size="sm", interactive=False)

        # State Management
        gr.Markdown("## 📝 State Management")
        with gr.Row():
            with gr.Column():
                state_json = gr.Code(
                    label="State JSON",
                    language="json",
                    value=json.dumps({
                        "user": {"name": "demo_user", "role": "admin"},
                        "settings": {"theme": "dark", "notifications": True}
                    }, indent=2),
                    lines=8,
                    interactive=False
                )
                update_state_btn = gr.Button("Update State", variant="primary", interactive=False)
                state_output = gr.Textbox(label="State Update Result", lines=4)

        # Event Logging
        gr.Markdown("## 📢 Event Logging")
        with gr.Row():
            with gr.Column():
                event_name = gr.Textbox(
                    label="Event Name",
                    placeholder="user_action",
                    value="user_click",
                    interactive=False
                )
                event_data = gr.Code(
                    label="Event Data (JSON)",
                    language="json",
                    value=json.dumps({"button": "submit", "page": "home"}, indent=2),
                    lines=5,
                    interactive=False
                )
                log_event_btn = gr.Button("Log Event", variant="primary", interactive=False)
                event_output = gr.Textbox(label="Event Result", lines=4)

        # Metrics Logging
        gr.Markdown("## 📈 Metrics Logging")
        with gr.Row():
            with gr.Column():
                metric_name = gr.Textbox(
                    label="Metric Name",
                    placeholder="cpu_usage",
                    value="response_time",
                    interactive=False
                )
            with gr.Column():
                metric_value = gr.Number(
                    label="Metric Value",
                    value=42.5,
                    interactive=False
                )
            with gr.Column():
                metric_tags = gr.Textbox(
                    label="Tags (comma-separated)",
                    placeholder="env:prod, service:api",
                    value="env:demo, type:performance",
                    interactive=False
                )
        log_metric_btn = gr.Button("Log Metric", variant="primary", interactive=False)
        metric_output = gr.Textbox(label="Metric Result", lines=4)

        # Error Logging
        gr.Markdown("## ⚠️ Error Logging (via Logging Handler)")
        gr.Markdown("Uses a custom logging handler that formats errors and sends them to datacat")
        with gr.Row():
            with gr.Column():
                error_level = gr.Dropdown(
                    label="Log Level",
                    choices=["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"],
                    value="ERROR",
                    interactive=False
                )
            with gr.Column():
                error_message = gr.Textbox(
                    label="Error Message",
                    placeholder="Something went wrong...",
                    value="Database connection timeout after 30 seconds",
                    interactive=False
                )
        log_error_btn = gr.Button("Log Error", variant="primary", interactive=False)
        error_output = gr.Textbox(label="Error Result", lines=4)

        # Exception Generation
        gr.Markdown("## 💥 Exception Generation & Logging")
        gr.Markdown(
            "Generate Python exceptions with full stack traces. "
            "Exceptions are logged via both the custom logging handler and direct API."
        )
        with gr.Row():
            with gr.Column():
                exception_type = gr.Dropdown(
                    label="Exception Type",
                    choices=[
                        "ValueError",
                        "TypeError",
                        "KeyError",
                        "IndexError",
                        "ZeroDivisionError"
                    ],
                    value="ValueError",
                    interactive=False
                )
            with gr.Column():
                include_nested = gr.Checkbox(
                    label="Include Nested Exception",
                    value=False,
                    info="Wraps the exception in a RuntimeError for nested exception demo",
                    interactive=False
                )
        generate_exception_btn = gr.Button("Generate & Log Exception", variant="primary", interactive=False)
        exception_output = gr.Textbox(label="Exception Result", lines=8)

        # Wire up the session toggle button
        session_toggle_btn.click(
            fn=toggle_session,
            inputs=[server_url, use_daemon, product_name, product_version],
            outputs=[
                session_output,
                current_session,
                session_toggle_btn,  # Update button text and variant
                state_json,
                update_state_btn,
                event_name,
                event_data,
                log_event_btn,
                metric_name,
                metric_value,
                metric_tags,
                log_metric_btn,
                error_level,
                error_message,
                log_error_btn,
                exception_type,
                include_nested,
                generate_exception_btn,
                refresh_stats_btn,
            ]
        )

        update_state_btn.click(
            fn=update_state,
            inputs=[state_json],
            outputs=[state_output]
        )

        log_event_btn.click(
            fn=log_event,
            inputs=[event_name, event_data],
            outputs=[event_output]
        )

        log_metric_btn.click(
            fn=log_metric,
            inputs=[metric_name, metric_value, metric_tags],
            outputs=[metric_output]
        )

        log_error_btn.click(
            fn=log_error_message,
            inputs=[error_level, error_message],
            outputs=[error_output]
        )

        generate_exception_btn.click(
            fn=generate_exception,
            inputs=[exception_type, include_nested],
            outputs=[exception_output]
        )

        refresh_stats_btn.click(
            fn=get_session_stats,
            outputs=[stats_output]
        )

        # Footer
        gr.Markdown(
            """
            ---
            **Instructions:**
            1. Make sure the datacat server is running on localhost:9090
            2. Enter your product name and version (required)
            3. Click "Start New Session" to begin
            4. Try out different features (enabled once session is active)
            5. View the session in the web UI at http://localhost:8080
            6. Click "End Session" when done

            **Note:** All feature controls are disabled until you start a session.

            **Theme:** Use Gradio's settings menu (⚙️) in the bottom left to toggle light/dark mode
            """
        )

    return demo


if __name__ == "__main__":
    demo = create_ui()

    print("\n" + "="*60)
    print("Starting datacat Demo GUI")
    print("="*60)
    print("\nMake sure the datacat server is running:")
    print("  Server: http://localhost:9090")
    print("  Web UI: http://localhost:8080")
    print("\nThe demo GUI will open in your browser...")
    print("="*60 + "\n")

    demo.launch(
        server_name="127.0.0.1",
        server_port=7860,
        share=False,
        inbrowser=True,
        favicon_path="logo-small.png"  # DataCat favicon
    )

