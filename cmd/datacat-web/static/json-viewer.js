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
            ...options
        };
        
        this.viewMode = 'tree'; // 'tree' or 'raw'
        this.render();
    }
    
    render() {
        this.container.innerHTML = '';
        
        // Create controls container
        const controls = document.createElement('div');
        controls.style.cssText = 'display: flex; gap: 10px; margin-bottom: 10px; justify-content: flex-end;';
        
        if (this.options.showCopyButton) {
            const copyBtn = document.createElement('button');
            copyBtn.className = 'btn-json-action';
            copyBtn.innerHTML = '📋 Copy';
            copyBtn.title = 'Copy JSON to clipboard';
            copyBtn.onclick = () => this.copyToClipboard();
            controls.appendChild(copyBtn);
        }
        
        if (this.options.showRawButton) {
            const rawBtn = document.createElement('button');
            rawBtn.className = 'btn-json-action';
            rawBtn.innerHTML = this.viewMode === 'tree' ? '📄 Raw' : '🌲 Tree';
            rawBtn.title = this.viewMode === 'tree' ? 'Show raw JSON' : 'Show tree view';
            rawBtn.onclick = () => this.toggleViewMode(rawBtn);
            controls.appendChild(rawBtn);
        }
        
        this.container.appendChild(controls);
        
        // Create content container
        const content = document.createElement('div');
        content.className = 'json-viewer-content';
        content.style.cssText = `
            background: var(--bg-hover);
            padding: 15px;
            border-radius: 4px;
            overflow: auto;
            ${this.options.maxHeight ? `max-height: ${this.options.maxHeight};` : ''}
        `;
        
        if (this.viewMode === 'tree') {
            content.appendChild(this.renderTree(this.jsonData, 0));
        } else {
            const pre = document.createElement('pre');
            pre.style.cssText = 'margin: 0; color: var(--text-primary); font-family: monospace; font-size: 13px;';
            pre.textContent = JSON.stringify(this.jsonData, null, 2);
            content.appendChild(pre);
        }
        
        this.container.appendChild(content);
    }
    
    renderTree(data, depth) {
        const container = document.createElement('div');
        container.style.marginLeft = depth > 0 ? '20px' : '0';
        
        if (data === null) {
            container.innerHTML = `<span style="color: #999;">null</span>`;
            return container;
        }
        
        if (typeof data !== 'object') {
            container.appendChild(this.renderValue(data));
            return container;
        }
        
        if (Array.isArray(data)) {
            if (data.length === 0) {
                container.innerHTML = `<span style="color: #999;">[]</span>`;
                return container;
            }
            
            const toggleId = `toggle-${Math.random().toString(36).substr(2, 9)}`;
            const isCollapsed = this.options.collapsed && depth > 0;
            
            const header = document.createElement('div');
            header.style.cssText = 'cursor: pointer; user-select: none; margin: 2px 0;';
            header.innerHTML = `
                <span id="${toggleId}" style="display: inline-block; width: 16px; transition: transform 0.2s;">
                    ${isCollapsed ? '▶' : '▼'}
                </span>
                <span style="color: #a0aec0;">Array[${data.length}]</span>
            `;
            
            const childContainer = document.createElement('div');
            childContainer.style.display = isCollapsed ? 'none' : 'block';
            
            data.forEach((item, index) => {
                const itemDiv = document.createElement('div');
                itemDiv.style.margin = '4px 0';
                
                const indexLabel = document.createElement('span');
                indexLabel.style.cssText = 'color: #667eea; margin-right: 8px; font-weight: 500;';
                indexLabel.textContent = `[${index}]:`;
                itemDiv.appendChild(indexLabel);
                
                const valueSpan = document.createElement('span');
                if (typeof item === 'object' && item !== null) {
                    valueSpan.appendChild(this.renderTree(item, depth + 1));
                } else {
                    valueSpan.appendChild(this.renderValue(item));
                }
                itemDiv.appendChild(valueSpan);
                
                childContainer.appendChild(itemDiv);
            });
            
            header.onclick = () => {
                const toggle = document.getElementById(toggleId);
                const isHidden = childContainer.style.display === 'none';
                childContainer.style.display = isHidden ? 'block' : 'none';
                toggle.textContent = isHidden ? '▼' : '▶';
            };
            
            container.appendChild(header);
            container.appendChild(childContainer);
        } else {
            // Object
            const keys = Object.keys(data);
            if (keys.length === 0) {
                container.innerHTML = `<span style="color: #999;">{}</span>`;
                return container;
            }
            
            const toggleId = `toggle-${Math.random().toString(36).substr(2, 9)}`;
            const isCollapsed = this.options.collapsed && depth > 0;
            
            const header = document.createElement('div');
            header.style.cssText = 'cursor: pointer; user-select: none; margin: 2px 0;';
            header.innerHTML = `
                <span id="${toggleId}" style="display: inline-block; width: 16px; transition: transform 0.2s;">
                    ${isCollapsed ? '▶' : '▼'}
                </span>
                <span style="color: #a0aec0;">Object{${keys.length}}</span>
            `;
            
            const childContainer = document.createElement('div');
            childContainer.style.display = isCollapsed ? 'none' : 'block';
            
            keys.forEach(key => {
                const itemDiv = document.createElement('div');
                itemDiv.style.margin = '4px 0';
                
                const keyLabel = document.createElement('span');
                keyLabel.style.cssText = 'color: #667eea; margin-right: 8px; font-weight: 500;';
                keyLabel.textContent = `${key}:`;
                itemDiv.appendChild(keyLabel);
                
                const valueSpan = document.createElement('span');
                if (typeof data[key] === 'object' && data[key] !== null) {
                    valueSpan.appendChild(this.renderTree(data[key], depth + 1));
                } else {
                    valueSpan.appendChild(this.renderValue(data[key]));
                }
                itemDiv.appendChild(valueSpan);
                
                childContainer.appendChild(itemDiv);
            });
            
            header.onclick = () => {
                const toggle = document.getElementById(toggleId);
                const isHidden = childContainer.style.display === 'none';
                childContainer.style.display = isHidden ? 'block' : 'none';
                toggle.textContent = isHidden ? '▼' : '▶';
            };
            
            container.appendChild(header);
            container.appendChild(childContainer);
        }
        
        return container;
    }
    
    renderValue(value) {
        const span = document.createElement('span');
        
        if (typeof value === 'string') {
            span.style.color = '#48bb78';
            span.textContent = `"${value}"`;
        } else if (typeof value === 'number') {
            span.style.color = '#ed8936';
            span.textContent = value.toString();
        } else if (typeof value === 'boolean') {
            span.style.color = '#f56565';
            span.textContent = value.toString();
        } else if (value === null) {
            span.style.color = '#999';
            span.textContent = 'null';
        } else if (value === undefined) {
            span.style.color = '#999';
            span.textContent = 'undefined';
        } else {
            span.style.color = 'var(--text-primary)';
            span.textContent = String(value);
        }
        
        return span;
    }
    
    toggleViewMode(button) {
        this.viewMode = this.viewMode === 'tree' ? 'raw' : 'tree';
        button.innerHTML = this.viewMode === 'tree' ? '📄 Raw' : '🌲 Tree';
        button.title = this.viewMode === 'tree' ? 'Show raw JSON' : 'Show tree view';
        this.render();
    }
    
    copyToClipboard() {
        const jsonString = JSON.stringify(this.jsonData, null, 2);
        navigator.clipboard.writeText(jsonString).then(() => {
            // Show success feedback
            const btn = this.container.querySelector('.btn-json-action');
            if (btn) {
                const originalText = btn.innerHTML;
                btn.innerHTML = '✓ Copied!';
                btn.style.background = 'var(--success-color)';
                setTimeout(() => {
                    btn.innerHTML = originalText;
                    btn.style.background = '';
                }, 2000);
            }
        }).catch(err => {
            console.error('Failed to copy to clipboard:', err);
            alert('Failed to copy to clipboard');
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
