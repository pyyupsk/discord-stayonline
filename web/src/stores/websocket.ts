import { defineStore } from "pinia";
import { computed, ref } from "vue";

import type { Configuration, ConnectionStatus, LogEntry, WebSocketMessage } from "@/types";

const MAX_RECONNECT_ATTEMPTS = 10;
const MAX_LOG_ENTRIES = 500;

export const useWebSocketStore = defineStore("websocket", () => {
  const wsStatus = ref<"connected" | "connecting" | "disconnected" | "error">("disconnected");
  const serverStatuses = ref<Map<string, ConnectionStatus>>(new Map());
  const serverNames = ref<Map<string, string>>(new Map());
  const logs = ref<LogEntry[]>([]);
  const logFilter = ref<"all" | LogEntry["level"]>("all");

  let ws: null | WebSocket = null;
  let reconnectAttempt = 0;
  let reconnectTimeout: null | ReturnType<typeof setTimeout> = null;
  let onConfigChanged: ((_config: Configuration) => void) | null = null;

  const filteredLogs = computed(() => {
    const filtered =
      logFilter.value === "all"
        ? logs.value
        : logs.value.filter((log) => log.level === logFilter.value);
    // Return newest first
    return [...filtered].reverse();
  });

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

  async function loadLogs() {
    try {
      const response = await fetch("/api/logs");
      if (!response.ok) return;

      const serverLogs: Array<{
        level: LogEntry["level"];
        message: string;
        timestamp: string;
      }> = await response.json();

      const existingTimestamps = new Set(logs.value.map((l) => l.time.getTime()));
      for (const log of serverLogs) {
        const logTime = new Date(log.timestamp);
        if (!existingTimestamps.has(logTime.getTime())) {
          const enriched = parseAndEnrichLogMessage(log.message, log.level);
          logs.value.push({
            ...enriched,
            time: logTime,
          });
          existingTimestamps.add(logTime.getTime());
        }
      }

      logs.value.sort((a, b) => a.time.getTime() - b.time.getTime());
      if (logs.value.length > MAX_LOG_ENTRIES) {
        logs.value = logs.value.slice(-MAX_LOG_ENTRIES);
      }
    } catch {
      // Silently fail
    }
  }

  function connect() {
    if (ws?.readyState === WebSocket.OPEN || ws?.readyState === WebSocket.CONNECTING) return;

    wsStatus.value = "connecting";

    const protocol = globalThis.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${globalThis.location.host}/ws`;

    try {
      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        reconnectAttempt = 0;
        wsStatus.value = "connected";
        addLog("info", "WebSocket connected");

        ws?.send(JSON.stringify({ channel: "logs", type: "subscribe" }));

        loadStatuses();
        loadLogs();
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
    const msgTime = msg.timestamp ? new Date(msg.timestamp) : new Date();

    switch (msg.type) {
      case "config_changed":
        if (msg.config && onConfigChanged) {
          onConfigChanged(msg.config);
          updateServerNamesFromConfig(msg.config);
        }
        addLogEntry(
          {
            action: "config",
            level: "info",
            message: "Configuration updated",
          },
          msgTime,
        );
        break;

      case "error":
        addLogEntry(
          {
            action: "error",
            level: "error",
            message: msg.message || "Unknown error",
            serverId: msg.server_id,
            serverName: getServerName(msg.server_id),
          },
          msgTime,
        );
        if (msg.server_id) {
          serverStatuses.value.set(msg.server_id, "error");
        }
        break;

      case "log":
        if (msg.message) {
          addLogEntry(
            {
              action: "system",
              level: (msg.level as LogEntry["level"]) || "info",
              message: msg.message,
            },
            msgTime,
          );
        }
        break;

      case "status":
        if (msg.server_id && msg.status) {
          serverStatuses.value.set(msg.server_id, msg.status);
          const serverName = getServerName(msg.server_id);
          const action = mapStatusToAction(msg.status);
          const friendlyMessage = getFriendlyStatusMessage(msg.status, serverName, msg.message);
          const level = getLogLevelForStatus(msg.status);

          addLogEntry(
            {
              action,
              level,
              message: friendlyMessage,
              serverId: msg.server_id,
              serverName,
            },
            msgTime,
          );
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

  function addLog(level: LogEntry["level"], message: string, action?: LogEntry["action"]) {
    addLogEntry({ action: action || "system", level, message });
  }

  function addLogEntry(entry: Omit<LogEntry, "time">, time?: Date) {
    const logTime = time ?? new Date();
    const timestamp = logTime.getTime();

    const isDuplicate = logs.value.some((l) => l.time.getTime() === timestamp);
    if (isDuplicate) return;

    logs.value.push({
      ...entry,
      time: logTime,
    });

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

  function setOnConfigChanged(callback: (_config: Configuration) => void) {
    onConfigChanged = callback;
  }

  function setLogFilter(filter: "all" | LogEntry["level"]) {
    logFilter.value = filter;
  }

  function updateServerNamesFromConfig(config: Configuration) {
    for (const server of config.servers) {
      const name = server.guild_name || `Server ${server.guild_id.slice(-4)}`;
      serverNames.value.set(server.id, name);
    }
  }

  function getServerName(serverId?: string): string | undefined {
    if (!serverId) return undefined;
    return serverNames.value.get(serverId);
  }

  function parseAndEnrichLogMessage(
    message: string,
    level: LogEntry["level"],
  ): Omit<LogEntry, "time"> {
    const match = new RegExp(/^\[([^\]]+)\]\s*(.+)$/).exec(message);

    if (!match) {
      return { action: "system", level, message };
    }

    const serverId = match[1] ?? "";
    const content = match[2] ?? "";
    const serverName = getServerName(serverId);
    const action = detectActionFromMessage(content);
    const friendlyMessage = createFriendlyMessage(action, serverName, content);

    return {
      action,
      level,
      message: friendlyMessage,
      serverId,
      serverName,
    };
  }

  return {
    addLog,
    clearLogs,
    connect,
    disconnect,
    filteredLogs,
    getServerStatus,
    loadLogs,
    loadStatuses,
    logFilter,
    logs,
    serverStatuses,
    setLogFilter,
    setOnConfigChanged,
    updateServerNamesFromConfig,
    wsStatus,
  };
});

function createFriendlyMessage(
  action: LogEntry["action"],
  serverName: string | undefined,
  originalContent: string,
): string {
  const name = serverName || "Server";

  switch (action) {
    case "backoff":
      return `Reconnecting to ${name}...`;
    case "connected":
      return `${name} is now online`;
    case "connecting":
      return `Connecting to ${name}...`;
    case "disconnected":
      return `${name} disconnected`;
    case "error":
      return `${name}: ${originalContent}`;
    default:
      return originalContent;
  }
}

function detectActionFromMessage(content: string): LogEntry["action"] {
  const lower = content.toLowerCase();

  if (lower.includes("connected") && !lower.includes("disconnected")) {
    return "connected";
  }
  if (lower.includes("connecting")) {
    return "connecting";
  }
  if (lower.includes("disconnected") || lower.includes("exit")) {
    return "disconnected";
  }
  if (lower.includes("reconnect") || lower.includes("waiting")) {
    return "backoff";
  }
  if (lower.includes("error") || lower.includes("failed")) {
    return "error";
  }
  if (lower.includes("config")) {
    return "config";
  }
  return "system";
}

function getFriendlyStatusMessage(
  status: ConnectionStatus,
  serverName?: string,
  originalMessage?: string,
): string {
  const name = serverName || "Server";

  switch (status) {
    case "backoff":
      return `Reconnecting to ${name}...`;
    case "connected":
      return `${name} is now online`;
    case "connecting":
      return `Connecting to ${name}...`;
    case "disconnected":
      return `${name} disconnected`;
    case "error":
      return originalMessage ? `${name}: ${originalMessage}` : `${name} encountered an error`;
    default:
      return originalMessage || `${name} status changed`;
  }
}

function getLogLevelForStatus(status: ConnectionStatus): LogEntry["level"] {
  switch (status) {
    case "backoff":
      return "warn";
    case "error":
      return "error";
    default:
      return "info";
  }
}

function mapStatusToAction(status: ConnectionStatus): LogEntry["action"] {
  switch (status) {
    case "backoff":
      return "backoff";
    case "connected":
      return "connected";
    case "connecting":
      return "connecting";
    case "disconnected":
      return "disconnected";
    case "error":
      return "error";
    default:
      return "system";
  }
}
