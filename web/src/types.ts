export type Status = "online" | "idle" | "dnd";

export type ConnectionStatus =
  | "connected"
  | "connecting"
  | "disconnected"
  | "error"
  | "backoff";

export interface ServerEntry {
  id: string;
  guild_id: string;
  channel_id: string;
  connect_on_start: boolean;
  priority: number;
}

export interface Configuration {
  servers: ServerEntry[];
  status: Status;
  tos_acknowledged: boolean;
}

export interface LogEntry {
  time: Date;
  level: "info" | "warn" | "error" | "debug";
  message: string;
}

export interface WebSocketMessage {
  type: "status" | "log" | "config_changed" | "error";
  server_id?: string;
  status?: ConnectionStatus;
  message?: string;
  level?: string;
  config?: Configuration;
  code?: string;
}
