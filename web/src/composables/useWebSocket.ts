import { ref, onUnmounted } from "vue";
import type {
  ConnectionStatus,
  LogEntry,
  WebSocketMessage,
  Configuration,
} from "@/types";

const MAX_RECONNECT_ATTEMPTS = 10;
const MAX_LOG_ENTRIES = 100;

const wsStatus = ref<"connected" | "connecting" | "disconnected" | "error">(
  "disconnected",
);
const serverStatuses = ref<Map<string, ConnectionStatus>>(new Map());
const logs = ref<LogEntry[]>([]);

let ws: WebSocket | null = null;
let reconnectAttempt = 0;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

function disconnect() {
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
  if (ws) {
    ws.close();
    ws = null;
  }
}

export function useWebSocket() {
  let onConfigChanged: ((config: Configuration) => void) | null = null;

  async function loadStatuses() {
    try {
      const response = await fetch("/api/statuses");
      if (!response.ok) return;

      const statuses: Record<string, ConnectionStatus> = await response.json();
      for (const [serverId, status] of Object.entries(statuses)) {
        serverStatuses.value.set(serverId, status);
      }
    } catch {
      // Silently fail - will get updates via WebSocket
    }
  }

  function connect() {
    if (ws?.readyState === WebSocket.OPEN) return;

    wsStatus.value = "connecting";

    const protocol = globalThis.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${globalThis.location.host}/ws`;

    try {
      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        reconnectAttempt = 0;
        wsStatus.value = "connected";
        addLog("info", "WebSocket connected");

        ws?.send(JSON.stringify({ type: "subscribe", channel: "logs" }));

        // Load current statuses after connecting
        loadStatuses();
      };

      ws.onclose = () => {
        wsStatus.value = "disconnected";
        addLog("warn", "WebSocket disconnected");
        scheduleReconnect();
      };

      ws.onerror = () => {
        wsStatus.value = "error";
        addLog("error", "WebSocket error");
      };

      ws.onmessage = (event) => {
        try {
          const msg: WebSocketMessage = JSON.parse(event.data);
          handleMessage(msg);
        } catch {
          addLog("error", "Failed to parse WebSocket message");
        }
      };
    } catch {
      addLog("error", "Failed to connect WebSocket");
      scheduleReconnect();
    }
  }

  function handleMessage(msg: WebSocketMessage) {
    switch (msg.type) {
      case "status":
        if (msg.server_id && msg.status) {
          serverStatuses.value.set(msg.server_id, msg.status);
          if (msg.message) {
            addLog("info", `[${msg.server_id}] ${msg.message}`);
          }
        }
        break;

      case "log":
        if (msg.message) {
          addLog((msg.level as LogEntry["level"]) || "info", msg.message);
        }
        break;

      case "config_changed":
        if (msg.config && onConfigChanged) {
          onConfigChanged(msg.config);
        }
        addLog("info", "Configuration updated");
        break;

      case "error":
        addLog("error", `[${msg.code}] ${msg.message}`);
        if (msg.server_id) {
          serverStatuses.value.set(msg.server_id, "error");
        }
        break;
    }
  }

  function scheduleReconnect() {
    if (reconnectAttempt >= MAX_RECONNECT_ATTEMPTS) {
      addLog("error", "Max WebSocket reconnection attempts reached");
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, reconnectAttempt), 30000);
    reconnectAttempt++;

    addLog("info", `Reconnecting in ${delay / 1000}s...`);
    reconnectTimeout = setTimeout(connect, delay);
  }

  function addLog(level: LogEntry["level"], message: string) {
    logs.value.push({
      time: new Date(),
      level,
      message,
    });

    // Keep only the last N entries
    if (logs.value.length > MAX_LOG_ENTRIES) {
      logs.value = logs.value.slice(-MAX_LOG_ENTRIES);
    }
  }

  function clearLogs() {
    logs.value = [];
  }

  function getServerStatus(serverId: string): ConnectionStatus {
    return serverStatuses.value.get(serverId) || "disconnected";
  }

  function setOnConfigChanged(callback: (config: Configuration) => void) {
    onConfigChanged = callback;
  }

  onUnmounted(() => {
    disconnect();
  });

  return {
    wsStatus,
    serverStatuses,
    logs,
    connect,
    disconnect,
    loadStatuses,
    addLog,
    clearLogs,
    getServerStatus,
    setOnConfigChanged,
  };
}
