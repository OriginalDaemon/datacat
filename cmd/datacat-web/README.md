# datacat-web

Interactive web dashboard for browsing datacat sessions and visualizing metrics.

## Features

### UI Features
- **Dark/Light theme toggle** - Switch between themes with localStorage persistence
- Browse all sessions with real-time updates
- **Session pagination** - 20 sessions per page with Previous/Next navigation
- **Multi-field sorting** - Sort by Created Time, Updated Time, Status, Event Count, Metric Count
- **JSON state filtering** - Filter sessions by state history using JSON format

### Session Detail Page
- View session metadata (Created, Updated, Ended, Status)
- **Interactive zoomable timeline** - Click and drag to zoom into any time range
- **Color-coded timeline points** - Blue (state), Green (events), Red (exceptions)
- **State context viewer** - Click any point to see cumulative state at that moment
- **Current state display** - Full nested state structure

### Metrics Visualization
- **Expandable metric summaries** - Click to expand/collapse charts
- **Comprehensive statistics** for each metric:
  - **Average** - Mean value across all data points
  - **Standard Deviation** - Measure of value spread
  - **Median** - Middle value in sorted dataset
  - **Min/Max** - Peak values
  - **Point count** - Number of measurements
- **Timeseries charts** - Interactive Chart.js graphs with zoom and pan
- Metrics grouped by name with intelligent aggregation

### Advanced Features
- Filter sessions by current state or state history
- Query sessions by array contains (e.g., find sessions with specific windows open)
- **Intelligent event stacking** - Events at similar times stack vertically
- **Adaptive bucketing** - Timeline groups items into time tranches based on zoom level
- Built with htmx for reactive UI without complex frameworks

## Running

```bash
cd cmd/datacat-web
go run main.go
```

The web UI will be available at `http://localhost:8080` by default.

## Building

```bash
cd cmd/datacat-web
go build -o datacat-web
./datacat-web
```

## Configuration

- `PORT` - Web server port (default: 8080)
- `API_URL` - datacat-server API URL (default: http://localhost:9090)

## Theme Switcher

The UI supports both dark and light themes. Your preference is saved to localStorage and persists across sessions.

- **Dark theme** (default) - Easy on the eyes with high contrast
- **Light theme** - Clean, bright interface

Click the "🌓 Toggle Theme" button in the navigation to switch between themes.

## Metrics Display

### Viewing Metrics

1. Navigate to a session detail page
2. Scroll to the "Metrics Summary" section
3. Each metric shows:
   - Name and overall statistics
   - Click the row to expand the timeseries chart
   - Click again to collapse

### Statistics Explained

- **Average**: Mean value = Σ(values) / count
- **Std Dev**: Standard deviation = √(Σ(x - avg)² / count)  
- **Median**: Middle value when sorted (50th percentile)
- **Min/Max**: Lowest and highest values recorded
- **Points**: Total number of data points

### Timeseries Charts

- **Hover** over data points to see exact values
- **Scroll** to zoom in/out (when implemented by Chart.js)
- Charts render on-demand when expanded (performance optimization)

## Timeline Features

### Zoom Functionality

1. **Click and drag** on the timeline to select a time range
2. Release to zoom into that range
3. Timeline re-renders with adaptive bucketing
4. Click **Reset Zoom** to return to full view

### Timeline Elements

- **Blue dots** - State changes (click to see cumulative state)
- **Green dots** - Events (startup, heartbeat, user actions)
- **Red dots** - Exceptions/errors (immediately visible for debugging)

### Use Cases

- **Debug errors**: Find red dots and see application state when error occurred
- **Track user journey**: Follow green event dots chronologically
- **Understand state evolution**: Click blue dots to see state at any point
- **Analyze performance**: Zoom into specific time periods for detailed analysis

## Example Queries

**Peak memory for sessions with "space probe" window:**
- Metric: `memory_usage`
- Aggregation: `peak per session`
- Filter Mode: `State Array Contains`
- Filter Path: `window_state.open`
- Filter Value: `space probe`

**CPU usage for currently running applications:**
- Metric: `cpu_usage`
- Aggregation: `all values`
- Filter Mode: `Current State Equals`
- Filter Path: `status`
- Filter Value: `running`

