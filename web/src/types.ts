export type Configuration = {
  servers: ServerEntry[];
  status: Status;
  tos_acknowledged: boolean;
};

export type ConnectionStatus = "backoff" | "connected" | "connecting" | "disconnected" | "error";

export type GuildInfo = {
  icon?: string;
  id: string;
  name: string;
};

export type LogEntry = {
  action?: "backoff" | "config" | "connected" | "connecting" | "disconnected" | "error" | "system";
  level: "debug" | "error" | "info" | "warn";
  message: string;
  serverId?: string;
  serverName?: string;
  time: Date;
};

export type NavigationState = {
  currentView: NavigationView;
  selectedServerId: null | string;
};

// Navigation state for sidebar views
export type NavigationView = "activity" | "dashboard" | "server";

export type ServerEntry = {
  channel_id: string;
  channel_name?: string;
  connect_on_start: boolean;
  guild_icon?: string;
  guild_id: string;
  guild_name?: string;
  id: string;
  priority: number;
};

// Server groups for organization
export type ServerGroup = {
  collapsed: boolean;
  id: string;
  name: string;
  serverIds: string[];
};

// Stats tracking
export type Stats = {
  connectionAttempts: number;
  serverUptimes: Map<string, number>;
  sessionStart: Date | null;
  successfulConnections: number;
  totalUptime: number;
};

export type Status = "dnd" | "idle" | "online";

export type VoiceChannelInfo = {
  id: string;
  name: string;
  position: number;
};

export type WebSocketMessage = {
  code?: string;
  config?: Configuration;
  level?: string;
  message?: string;
  server_id?: string;
  status?: ConnectionStatus;
  type: "config_changed" | "error" | "log" | "status";
};
