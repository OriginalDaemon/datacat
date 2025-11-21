# DataCat Icons Integration Guide

This guide explains how to integrate the DataCat app icons into the project.

## Icon Files Provided

You have 4 icon variations in different resolutions:
1. **32x32 / 48x48** - Favicon size (smallest)
2. **64x64 / 128x128** - Small logo
3. **256x256** - Medium logo
4. **512x512** - Large/high-res logo

## Where to Place Icon Files

### 1. Web UI (`cmd/datacat-web/static/`)

Place these files in the `cmd/datacat-web/static/` directory:

```
cmd/datacat-web/static/
├── favicon.ico          # 32x32 icon (convert smallest PNG to ICO)
├── favicon.png          # 32x32 or 48x48 PNG (smallest icon)
├── logo-small.png       # 64x64 or 128x128 PNG (second smallest)
├── logo.png             # 256x256 PNG (medium icon)
└── logo-large.png       # 512x512 PNG (largest icon)
```

**What's already updated:**
- ✅ `templates/base.html` - References favicon and displays logo in header
- ✅ `static/ICONS.md` - Documentation for web UI icons

### 2. Demo GUI (`examples/demo_gui/`)

Place this file in the `examples/demo_gui/` directory:

```
examples/demo_gui/
└── logo-small.png       # 64x64 or 128x128 PNG (for browser tab)
```

**What's already updated:**
- ✅ `demo_gui.py` - Set `favicon_path="logo-small.png"` in launch config
- ✅ `ICONS.md` - Documentation for demo GUI icon

### 3. Documentation (`README.md`, etc.)

You can optionally add icons to documentation:

```markdown
![DataCat Logo](cmd/datacat-web/static/logo.png)
```

Or reference them inline in markdown files.

## File Naming Convention

| File Name | Size | Usage |
|-----------|------|-------|
| `favicon.ico` | 32x32 | Browser favicon (ICO format) |
| `favicon.png` | 32x32-48x48 | Browser favicon (PNG fallback) |
| `logo-small.png` | 64x64-128x128 | Headers, mobile, demo GUI |
| `logo.png` | 256x256 | Main logo in docs/UI |
| `logo-large.png` | 512x512 | High-res displays, hero images |

## How to Save the Icons

### From the provided images:

1. **Right-click** on each icon image shown in the chat
2. **Save As...** and save to the appropriate location with the correct filename
3. Follow the directory structure above

### Quick Copy Instructions

**For Web UI:**
```bash
# Save all 4-5 icons to:
cd cmd/datacat-web/static/

# You should have:
# - favicon.ico (convert smallest to ICO if needed)
# - favicon.png (smallest)
# - logo-small.png (second smallest)
# - logo.png (medium)
# - logo-large.png (largest)
```

**For Demo GUI:**
```bash
# Save the small/medium icon to:
cd examples/demo_gui/

# You should have:
# - logo-small.png (same as web UI's logo-small.png)
```

## Converting PNG to ICO

If you need to create `favicon.ico` from the PNG:

### Using Online Tool:
- Visit https://convertico.com/
- Upload the smallest PNG (32x32 or 48x48)
- Download as `favicon.ico`

### Using ImageMagick (if installed):
```bash
convert favicon.png -define icon:auto-resize=32,16 favicon.ico
```

### Using Python (Pillow):
```python
from PIL import Image
img = Image.open('favicon.png')
img.save('favicon.ico', format='ICO', sizes=[(32, 32)])
```

## What's Already Integrated

### ✅ Web UI (`cmd/datacat-web/`)
- Favicon links in `<head>` of `templates/base.html`
- Logo in header with proper styling
- Theme-aware display (works in dark/light modes)

### ✅ Demo GUI (`examples/demo_gui/`)
- Favicon configured in `demo_gui.py` launch parameters
- Will display in browser tab when running

### ✅ Documentation
- ASCII art logo in `README.md` and `examples/README.md`
- "DataCat" branding updated throughout all docs

## Testing

After placing the icons:

### Test Web UI:
```bash
# Start the web UI
cd cmd/datacat-web
go run main.go

# Open browser to http://localhost:8080
# Check:
# - Favicon appears in browser tab
# - Logo appears in header next to "DataCat Dashboard"
```

### Test Demo GUI:
```bash
# Start the demo GUI
cd examples/demo_gui
python demo_gui.py

# When browser opens to http://127.0.0.1:7860
# Check:
# - Favicon appears in browser tab
```

## Icon Design Details

The DataCat icons feature:
- **Cute cat mascot** - Friendly, approachable design
- **Data visualization elements** - Bars/charts integrated into design
- **Blue-to-purple gradient** - Modern color scheme matching UI
- **"DataCat" text** - Clear branding
- **Clean, modern aesthetic** - Professional yet friendly
- **Transparent/white background** - Works on any background

## Troubleshooting

**Favicon not appearing?**
- Hard refresh your browser (Ctrl+F5 or Cmd+Shift+R)
- Clear browser cache
- Check browser developer tools console for 404 errors
- Verify file exists in the correct directory

**Logo not showing in header?**
- Check the browser console for 404 errors on `/static/logo-small.png`
- Verify the file is in `cmd/datacat-web/static/logo-small.png`
- Restart the web server after adding the file

**Demo GUI icon not showing?**
- Verify `logo-small.png` exists in `examples/demo_gui/`
- Gradio may cache the favicon - try clearing browser cache
- Check the Gradio console output for any warnings

## Need Help?

If the icons don't appear after following these steps:
1. Check the file paths match exactly
2. Verify file permissions (should be readable)
3. Restart the applications after adding icons
4. Check for typos in filenames (case-sensitive on Linux/macOS)

