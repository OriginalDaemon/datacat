/**
 * JSON Viewer Component
 * Provides expandable tree view for JSON data with copy and raw/formatted view toggle
 */

class JSONViewer {
  constructor(containerId, jsonData, options = {}) {
    this.container = document.getElementById(containerId);
    if (!this.container) {
      console.error(`Container with id '${containerId}' not found`);
      return;
    }

    this.jsonData = jsonData;
    this.options = {
      collapsed: options.collapsed || false,
      showCopyButton: options.showCopyButton !== false,
      showRawButton: options.showRawButton !== false,
      maxHeight: options.maxHeight || null,
      highlightAdded: options.highlightAdded || [],
      highlightModified: options.highlightModified || [],
      highlightRemoved: options.highlightRemoved || [],
      ...options,
    };

    this.viewMode = "tree"; // 'tree' or 'raw'
    this.render();
  }

  render() {
    this.container.innerHTML = "";

    // Create controls container
    const controls = document.createElement("div");
    controls.style.cssText =
      "display: flex; gap: 10px; margin-bottom: 10px; justify-content: flex-end;";

    if (this.options.showCopyButton) {
      const copyBtn = document.createElement("button");
      copyBtn.className = "btn-json-action";
      copyBtn.innerHTML = "📋 Copy";
      copyBtn.title = "Copy JSON to clipboard";
      copyBtn.onclick = () => this.copyToClipboard();
      controls.appendChild(copyBtn);
    }

    if (this.options.showRawButton) {
      const rawBtn = document.createElement("button");
      rawBtn.className = "btn-json-action";
      rawBtn.innerHTML = this.viewMode === "tree" ? "📄 Raw" : "🌲 Tree";
      rawBtn.title =
        this.viewMode === "tree" ? "Show raw JSON" : "Show tree view";
      rawBtn.onclick = () => this.toggleViewMode(rawBtn);
      controls.appendChild(rawBtn);
    }

    this.container.appendChild(controls);

    // Create content container
    const content = document.createElement("div");
    content.className = "json-viewer-content";
    content.style.cssText = `
            background: var(--bg-hover);
            padding: 15px;
            border-radius: 4px;
            overflow: auto;
            ${
              this.options.maxHeight
                ? `max-height: ${this.options.maxHeight};`
                : ""
            }
        `;

    if (this.viewMode === "tree") {
      content.appendChild(this.renderTree(this.jsonData, 0, ""));
    } else {
      const pre = document.createElement("pre");
      pre.style.cssText =
        "margin: 0; color: var(--text-primary); font-family: monospace; font-size: 13px;";
      pre.textContent = JSON.stringify(this.jsonData, null, 2);
      content.appendChild(pre);
    }

    this.container.appendChild(content);
  }

  renderTree(data, depth, path) {
    const container = document.createElement("div");
    container.style.marginLeft = depth > 0 ? "20px" : "0";

    // Helper to get highlight style for current path
    const getHighlightStyle = (itemPath) => {
      if (this.options.highlightAdded.includes(itemPath)) {
        return "background: rgba(72, 187, 120, 0.15); border-left: 3px solid #48bb78; padding-left: 8px; margin-left: -8px;";
      }
      if (this.options.highlightModified.includes(itemPath)) {
        return "background: rgba(246, 173, 85, 0.15); border-left: 3px solid #f6ad55; padding-left: 8px; margin-left: -8px;";
      }
      if (this.options.highlightRemoved.includes(itemPath)) {
        return "background: rgba(245, 101, 101, 0.15); border-left: 3px solid #f56565; padding-left: 8px; margin-left: -8px;";
      }
      return "";
    };

    if (data === null) {
      container.innerHTML = `<span style="color: #999;">null</span>`;
      return container;
    }

    if (typeof data !== "object") {
      container.appendChild(this.renderValue(data));
      return container;
    }

    if (Array.isArray(data)) {
      if (data.length === 0) {
        container.innerHTML = `<span style="color: #999;">[]</span>`;
        return container;
      }

      // For arrays, render each item with index
      data.forEach((item, index) => {
        const itemPath = path ? `${path}[${index}]` : `[${index}]`;
        const itemDiv = document.createElement("div");
        itemDiv.style.margin = "4px 0";

        // Apply highlight style if this path matches
        const highlightStyle = getHighlightStyle(itemPath);
        if (highlightStyle) {
          itemDiv.style.cssText += highlightStyle;
        }

        // Check if item is complex (object/array)
        const isComplex = typeof item === "object" && item !== null;

        if (isComplex) {
          const toggleId = `toggle-${Math.random().toString(36).substr(2, 9)}`;
          const isCollapsed = this.options.collapsed && depth > 0;

          const header = document.createElement("span");
          header.style.cssText = "cursor: pointer; user-select: none;";

          const arrow = document.createElement("span");
          arrow.id = toggleId;
          arrow.style.cssText =
            "display: inline-block; width: 16px; transition: transform 0.2s;";
          arrow.textContent = isCollapsed ? "▶" : "▼";

          const indexLabel = document.createElement("span");
          indexLabel.style.cssText =
            "color: #667eea; margin-left: 4px; margin-right: 8px; font-weight: 500;";
          indexLabel.textContent = `[${index}]:`;

          const typeHint = document.createElement("span");
          typeHint.style.cssText =
            "color: #a0aec0; font-style: italic; font-size: 12px;";
          if (Array.isArray(item)) {
            typeHint.textContent = ` (${item.length} items)`;
          } else {
            const keyCount = Object.keys(item).length;
            typeHint.textContent = ` (${keyCount} ${
              keyCount === 1 ? "property" : "properties"
            })`;
          }

          header.appendChild(arrow);
          header.appendChild(indexLabel);
          header.appendChild(typeHint);

          const childContainer = document.createElement("div");
          childContainer.style.display = isCollapsed ? "none" : "block";
          childContainer.appendChild(this.renderTree(item, depth + 1, itemPath));

          header.onclick = () => {
            const isHidden = childContainer.style.display === "none";
            childContainer.style.display = isHidden ? "block" : "none";
            arrow.textContent = isHidden ? "▼" : "▶";
          };

          itemDiv.appendChild(header);
          itemDiv.appendChild(childContainer);
        } else {
          // Primitive value - no expand arrow
          const indexLabel = document.createElement("span");
          indexLabel.style.cssText =
            "color: #667eea; margin-right: 8px; font-weight: 500; margin-left: 16px;";
          indexLabel.textContent = `[${index}]:`;
          itemDiv.appendChild(indexLabel);
          itemDiv.appendChild(this.renderValue(item));
        }

        container.appendChild(itemDiv);
      });
    } else {
      // Object - render each property
      const keys = Object.keys(data);
      if (keys.length === 0) {
        container.innerHTML = `<span style="color: #999;">{}</span>`;
        return container;
      }

      keys.forEach((key) => {
        const itemPath = path ? `${path}.${key}` : key;
        const itemDiv = document.createElement("div");
        itemDiv.style.margin = "4px 0";

        // Apply highlight style if this path matches
        const highlightStyle = getHighlightStyle(itemPath);
        if (highlightStyle) {
          itemDiv.style.cssText += highlightStyle;
        }

        const value = data[key];
        const isComplex = typeof value === "object" && value !== null;

        if (isComplex) {
          // Complex value (object/array) - show with expand arrow
          const toggleId = `toggle-${Math.random().toString(36).substr(2, 9)}`;
          const isCollapsed = this.options.collapsed && depth > 0;

          const header = document.createElement("span");
          header.style.cssText = "cursor: pointer; user-select: none;";

          const arrow = document.createElement("span");
          arrow.id = toggleId;
          arrow.style.cssText =
            "display: inline-block; width: 16px; transition: transform 0.2s;";
          arrow.textContent = isCollapsed ? "▶" : "▼";

          const keyLabel = document.createElement("span");
          keyLabel.style.cssText =
            "color: #667eea; margin-left: 4px; margin-right: 8px; font-weight: 500;";
          keyLabel.textContent = `${key}:`;

          const typeHint = document.createElement("span");
          typeHint.style.cssText =
            "color: #a0aec0; font-style: italic; font-size: 12px;";
          if (Array.isArray(value)) {
            typeHint.textContent = `(${value.length} items)`;
          } else {
            const keyCount = Object.keys(value).length;
            typeHint.textContent = `(${keyCount} ${
              keyCount === 1 ? "property" : "properties"
            })`;
          }

          header.appendChild(arrow);
          header.appendChild(keyLabel);
          header.appendChild(typeHint);

          const childContainer = document.createElement("div");
          childContainer.style.display = isCollapsed ? "none" : "block";
          childContainer.appendChild(this.renderTree(value, depth + 1, itemPath));

          header.onclick = () => {
            const isHidden = childContainer.style.display === "none";
            childContainer.style.display = isHidden ? "block" : "none";
            arrow.textContent = isHidden ? "▼" : "▶";
          };

          itemDiv.appendChild(header);
          itemDiv.appendChild(childContainer);
        } else {
          // Primitive value - no expand arrow
          const keyLabel = document.createElement("span");
          keyLabel.style.cssText =
            "color: #667eea; margin-right: 8px; font-weight: 500; margin-left: 16px;";
          keyLabel.textContent = `${key}:`;
          itemDiv.appendChild(keyLabel);
          itemDiv.appendChild(this.renderValue(value));
        }

        container.appendChild(itemDiv);
      });
    }

    return container;
  }

  renderValue(value) {
    const span = document.createElement("span");

    if (typeof value === "string") {
      span.style.color = "#48bb78";

      // Check if the string contains file paths (common in tracebacks)
      // Pattern: File "path", line N, in function
      const filePathPattern = /File "([^"]+)", line (\d+)/g;

      if (filePathPattern.test(value)) {
        // Reset the regex
        filePathPattern.lastIndex = 0;

        // Build the content with clickable links
        span.innerHTML = '"';
        let lastIndex = 0;
        let match;

        while ((match = filePathPattern.exec(value)) !== null) {
          const beforeMatch = value.substring(lastIndex, match.index);
          const filePath = match[1];
          const lineNumber = match[2];
          const matchedText = match[0];

          // Add text before the match
          span.appendChild(document.createTextNode(beforeMatch));

          // Create clickable link
          const link = document.createElement("a");
          link.textContent = matchedText;
          link.style.cssText =
            "color: #667eea; text-decoration: underline; cursor: pointer;";
          link.title = `Open ${filePath}:${lineNumber} in IDE\n\nClick to use: PyCharm | Ctrl+Click: VSCode | Shift+Click: File only`;

          // Support multiple IDEs based on modifier keys
          link.onclick = (e) => {
            e.preventDefault();
            e.stopPropagation();

            let url;
            if (e.ctrlKey || e.metaKey) {
              // VSCode
              url = `vscode://file${filePath}:${lineNumber}`;
            } else if (e.shiftKey) {
              // Generic file:// protocol (opens file, but not at specific line)
              url = `file://${filePath}`;
            } else {
              // PyCharm (default)
              url = `pycharm://open?file=${encodeURIComponent(
                filePath
              )}&line=${lineNumber}`;
            }

            window.location.href = url;
          };

          span.appendChild(link);
          lastIndex = match.index + matchedText.length;
        }

        // Add remaining text after last match
        const remainingText = value.substring(lastIndex);
        span.appendChild(document.createTextNode(remainingText + '"'));
      } else {
        // No file paths, render normally
        span.textContent = `"${value}"`;
      }
    } else if (typeof value === "number") {
      span.style.color = "#ed8936";
      span.textContent = value.toString();
    } else if (typeof value === "boolean") {
      span.style.color = "#f56565";
      span.textContent = value.toString();
    } else if (value === null) {
      span.style.color = "#999";
      span.textContent = "null";
    } else if (value === undefined) {
      span.style.color = "#999";
      span.textContent = "undefined";
    } else {
      span.style.color = "var(--text-primary)";
      span.textContent = String(value);
    }

    return span;
  }

  toggleViewMode(button) {
    this.viewMode = this.viewMode === "tree" ? "raw" : "tree";
    button.innerHTML = this.viewMode === "tree" ? "📄 Raw" : "🌲 Tree";
    button.title =
      this.viewMode === "tree" ? "Show raw JSON" : "Show tree view";
    this.render();
  }

  copyToClipboard() {
    const jsonString = JSON.stringify(this.jsonData, null, 2);
    navigator.clipboard
      .writeText(jsonString)
      .then(() => {
        // Show success feedback
        const btn = this.container.querySelector(".btn-json-action");
        if (btn) {
          const originalText = btn.innerHTML;
          btn.innerHTML = "✓ Copied!";
          btn.style.background = "var(--success-color)";
          setTimeout(() => {
            btn.innerHTML = originalText;
            btn.style.background = "";
          }, 2000);
        }
      })
      .catch((err) => {
        console.error("Failed to copy to clipboard:", err);
        alert("Failed to copy to clipboard");
      });
  }
}

// Helper function to create a JSON viewer
function createJSONViewer(containerId, jsonData, options = {}) {
  return new JSONViewer(containerId, jsonData, options);
}

// Make it globally available
window.JSONViewer = JSONViewer;
window.createJSONViewer = createJSONViewer;
