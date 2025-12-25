// Discord Stay Online - Vanilla JS Application

(function() {
    'use strict';

    // State
    let config = { servers: [], tos_acknowledged: false };
    let ws = null;
    let wsReconnectAttempt = 0;
    const MAX_WS_RECONNECT = 10;

    // DOM Elements
    const tosModal = document.getElementById('tos-modal');
    const tosAcknowledgeBtn = document.getElementById('tos-acknowledge-btn');
    const app = document.getElementById('app');
    const serversList = document.getElementById('servers-list');
    const addServerBtn = document.getElementById('add-server-btn');
    const serverFormSection = document.getElementById('server-form-section');
    const serverForm = document.getElementById('server-form');
    const formTitle = document.getElementById('form-title');
    const cancelFormBtn = document.getElementById('cancel-form-btn');
    const logContainer = document.getElementById('log-container');
    const clearLogBtn = document.getElementById('clear-log-btn');
    const confirmModal = document.getElementById('confirm-modal');
    const confirmTitle = document.getElementById('confirm-title');
    const confirmMessage = document.getElementById('confirm-message');
    const confirmYesBtn = document.getElementById('confirm-yes-btn');
    const confirmNoBtn = document.getElementById('confirm-no-btn');
    const connectionStatus = document.getElementById('connection-status');

    // Initialization
    async function init() {
        try {
            // Fetch current config
            const response = await fetch('/api/config');
            if (response.ok) {
                config = await response.json();
            }
        } catch (error) {
            log('error', 'Failed to load configuration: ' + error.message);
        }

        // Check TOS acknowledgment
        if (!config.tos_acknowledged) {
            showTOSModal();
        } else {
            showApp();
        }

        // Set up event listeners
        setupEventListeners();
    }

    // TOS Modal
    function showTOSModal() {
        tosModal.classList.remove('hidden');
        app.classList.add('hidden');

        // Prevent keyboard dismiss (Escape, Enter)
        document.addEventListener('keydown', preventModalDismiss);
    }

    function preventModalDismiss(e) {
        if (!config.tos_acknowledged) {
            if (e.key === 'Escape' || e.key === 'Enter') {
                e.preventDefault();
                e.stopPropagation();
            }
        }
    }

    async function acknowledgeTOS() {
        try {
            const response = await fetch('/api/acknowledge-tos', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ acknowledged: true })
            });

            if (response.ok) {
                config.tos_acknowledged = true;
                document.removeEventListener('keydown', preventModalDismiss);
                showApp();
                log('info', 'Terms of Service acknowledged');
            } else {
                const error = await response.json();
                log('error', 'Failed to acknowledge TOS: ' + error.message);
            }
        } catch (error) {
            log('error', 'Failed to acknowledge TOS: ' + error.message);
        }
    }

    // App Display
    function showApp() {
        tosModal.classList.add('hidden');
        app.classList.remove('hidden');
        renderServers();
        connectWebSocket();
    }

    // Event Listeners
    function setupEventListeners() {
        // TOS acknowledgment - only via click, not keyboard
        tosAcknowledgeBtn.addEventListener('click', acknowledgeTOS);

        // Server form
        addServerBtn.addEventListener('click', showAddServerForm);
        cancelFormBtn.addEventListener('click', hideServerForm);
        serverForm.addEventListener('submit', handleServerFormSubmit);

        // Log
        clearLogBtn.addEventListener('click', clearLog);

        // Confirmation modal
        confirmNoBtn.addEventListener('click', hideConfirmModal);
    }

    // Server List
    function renderServers() {
        serversList.innerHTML = '';

        if (config.servers.length === 0) {
            serversList.innerHTML = '<div class="empty-state">No servers configured. Click "Add Server" to get started.</div>';
            return;
        }

        config.servers.forEach(server => {
            const card = document.createElement('div');
            card.className = 'server-card';
            card.dataset.id = server.id;

            card.innerHTML = `
                <div class="server-info">
                    <span class="server-name">Server Entry</span>
                    <span class="server-ids">Guild: ${server.guild_id} | Channel: ${server.channel_id}</span>
                </div>
                <div class="server-status status-indicator disconnected" data-server-status="${server.id}">
                    <span class="status-dot"></span>
                    <span class="status-text">Disconnected</span>
                </div>
                <div class="server-actions">
                    <button class="btn btn-primary btn-small" onclick="window.app.joinServer('${server.id}')">Join</button>
                    <button class="btn btn-secondary btn-small" onclick="window.app.rejoinServer('${server.id}')">Rejoin</button>
                    <button class="btn btn-secondary btn-small" onclick="window.app.exitServer('${server.id}')">Exit</button>
                    <button class="btn btn-secondary btn-small" onclick="window.app.editServer('${server.id}')">Edit</button>
                    <button class="btn btn-danger btn-small" onclick="window.app.deleteServer('${server.id}')">Delete</button>
                </div>
            `;

            serversList.appendChild(card);
        });
    }

    // Server Form
    function showAddServerForm() {
        formTitle.textContent = 'Add Server';
        serverForm.reset();
        document.getElementById('server-id').value = '';
        serverFormSection.classList.remove('hidden');
    }

    function showEditServerForm(serverId) {
        const server = config.servers.find(s => s.id === serverId);
        if (!server) return;

        formTitle.textContent = 'Edit Server';
        document.getElementById('server-id').value = server.id;
        document.getElementById('guild-id').value = server.guild_id;
        document.getElementById('channel-id').value = server.channel_id;
        document.getElementById('status').value = server.status;
        document.getElementById('connect-on-start').checked = server.connect_on_start;
        serverFormSection.classList.remove('hidden');
    }

    function hideServerForm() {
        serverFormSection.classList.add('hidden');
        serverForm.reset();
    }

    async function handleServerFormSubmit(e) {
        e.preventDefault();

        const formData = new FormData(serverForm);
        const serverId = formData.get('id');
        const isEdit = !!serverId;

        const serverData = {
            id: serverId || generateId(),
            guild_id: formData.get('guild_id'),
            channel_id: formData.get('channel_id'),
            status: formData.get('status'),
            connect_on_start: document.getElementById('connect-on-start').checked,
            priority: 1
        };

        try {
            if (isEdit) {
                // Update existing server
                const idx = config.servers.findIndex(s => s.id === serverId);
                if (idx !== -1) {
                    config.servers[idx] = serverData;
                }
            } else {
                // Check max limit
                if (config.servers.length >= 15) {
                    log('error', 'Maximum 15 server entries allowed');
                    return;
                }
                config.servers.push(serverData);
            }

            // Save config
            const response = await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ servers: config.servers })
            });

            if (response.ok) {
                const result = await response.json();
                if (result.servers) {
                    config.servers = result.servers;
                }
                hideServerForm();
                renderServers();
                log('info', isEdit ? 'Server updated' : 'Server added');
            } else {
                const error = await response.json();
                log('error', 'Failed to save: ' + error.message);
            }
        } catch (error) {
            log('error', 'Failed to save: ' + error.message);
        }
    }

    // Server Actions
    async function executeServerAction(serverId, action) {
        try {
            const response = await fetch(`/api/servers/${serverId}/action`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action })
            });

            if (response.ok) {
                const result = await response.json();
                log('info', `Action '${action}' executed for server ${serverId}`);
                updateServerStatus(serverId, result.new_status);
            } else {
                const error = await response.json();
                log('error', `Action failed: ${error.message}`);
            }
        } catch (error) {
            log('error', `Action failed: ${error.message}`);
        }
    }

    function joinServer(serverId) {
        executeServerAction(serverId, 'join');
    }

    function rejoinServer(serverId) {
        showConfirmModal(
            'Confirm Rejoin',
            'Are you sure you want to rejoin this server? This will close the current connection and reconnect.',
            () => executeServerAction(serverId, 'rejoin')
        );
    }

    function exitServer(serverId) {
        showConfirmModal(
            'Confirm Exit',
            'Are you sure you want to exit this server? This will close the connection.',
            () => executeServerAction(serverId, 'exit')
        );
    }

    function editServer(serverId) {
        showEditServerForm(serverId);
    }

    async function deleteServer(serverId) {
        showConfirmModal(
            'Confirm Delete',
            'Are you sure you want to delete this server entry? This action cannot be undone.',
            async () => {
                config.servers = config.servers.filter(s => s.id !== serverId);

                try {
                    const response = await fetch('/api/config', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ servers: config.servers })
                    });

                    if (response.ok) {
                        renderServers();
                        log('info', 'Server deleted');
                    } else {
                        const error = await response.json();
                        log('error', 'Failed to delete: ' + error.message);
                    }
                } catch (error) {
                    log('error', 'Failed to delete: ' + error.message);
                }
            }
        );
    }

    // Confirmation Modal
    function showConfirmModal(title, message, onConfirm) {
        confirmTitle.textContent = title;
        confirmMessage.textContent = message;
        confirmModal.classList.remove('hidden');

        // Remove previous listener
        confirmYesBtn.onclick = null;

        // Add new listener
        confirmYesBtn.onclick = () => {
            hideConfirmModal();
            onConfirm();
        };
    }

    function hideConfirmModal() {
        confirmModal.classList.add('hidden');
    }

    // Status Updates
    function updateServerStatus(serverId, status) {
        const statusEl = document.querySelector(`[data-server-status="${serverId}"]`);
        if (!statusEl) return;

        // Remove all status classes
        statusEl.classList.remove('connected', 'connecting', 'disconnected', 'error', 'backoff');

        // Add new status class
        statusEl.classList.add(status);

        // Update text
        const textEl = statusEl.querySelector('.status-text');
        if (textEl) {
            textEl.textContent = status.charAt(0).toUpperCase() + status.slice(1);
        }
    }

    function updateConnectionStatus(status) {
        connectionStatus.classList.remove('connected', 'connecting', 'disconnected', 'error');
        connectionStatus.classList.add(status);

        const textEl = connectionStatus.querySelector('.status-text');
        if (textEl) {
            textEl.textContent = status.charAt(0).toUpperCase() + status.slice(1);
        }
    }

    // WebSocket
    function connectWebSocket() {
        if (ws && ws.readyState === WebSocket.OPEN) return;

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        updateConnectionStatus('connecting');

        try {
            ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                wsReconnectAttempt = 0;
                updateConnectionStatus('connected');
                log('info', 'WebSocket connected');

                // Subscribe to logs
                ws.send(JSON.stringify({ type: 'subscribe', channel: 'logs' }));
            };

            ws.onclose = () => {
                updateConnectionStatus('disconnected');
                log('warn', 'WebSocket disconnected');
                scheduleReconnect();
            };

            ws.onerror = (error) => {
                updateConnectionStatus('error');
                log('error', 'WebSocket error');
            };

            ws.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data);
                    handleWebSocketMessage(msg);
                } catch (error) {
                    log('error', 'Failed to parse WebSocket message');
                }
            };
        } catch (error) {
            log('error', 'Failed to connect WebSocket: ' + error.message);
            scheduleReconnect();
        }
    }

    function handleWebSocketMessage(msg) {
        switch (msg.type) {
            case 'status':
                updateServerStatus(msg.server_id, msg.status);
                if (msg.message) {
                    log('info', `[${msg.server_id}] ${msg.message}`);
                }
                break;

            case 'log':
                log(msg.level, msg.message);
                break;

            case 'config_changed':
                config = msg.config;
                renderServers();
                log('info', 'Configuration updated');
                break;

            case 'error':
                log('error', `[${msg.code}] ${msg.message}`);
                if (msg.server_id) {
                    updateServerStatus(msg.server_id, 'error');
                }
                break;
        }
    }

    function scheduleReconnect() {
        if (wsReconnectAttempt >= MAX_WS_RECONNECT) {
            log('error', 'Max WebSocket reconnection attempts reached');
            return;
        }

        const delay = Math.min(1000 * Math.pow(2, wsReconnectAttempt), 30000);
        wsReconnectAttempt++;

        log('info', `Reconnecting in ${delay / 1000}s...`);
        setTimeout(connectWebSocket, delay);
    }

    // Logging
    function log(level, message) {
        const entry = document.createElement('div');
        entry.className = `log-entry log-${level}`;

        const time = new Date().toLocaleTimeString();
        entry.innerHTML = `
            <span class="log-time">${time}</span>
            <span class="log-level">[${level.toUpperCase()}]</span>
            <span class="log-message">${escapeHtml(message)}</span>
        `;

        logContainer.appendChild(entry);
        logContainer.scrollTop = logContainer.scrollHeight;
    }

    function clearLog() {
        logContainer.innerHTML = '';
    }

    // Utilities
    function generateId() {
        return 'xxxxxxxx'.replace(/x/g, () => {
            return Math.floor(Math.random() * 16).toString(16);
        });
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Expose public API
    window.app = {
        joinServer,
        rejoinServer,
        exitServer,
        editServer,
        deleteServer
    };

    // Start application
    document.addEventListener('DOMContentLoaded', init);
})();
