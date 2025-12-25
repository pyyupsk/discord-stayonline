import { ref } from "vue";
import type { Configuration, ServerEntry, Status } from "@/types";

const config = ref<Configuration>({
  servers: [],
  status: "online",
  tos_acknowledged: false,
});

const loading = ref(false);
const error = ref<string | null>(null);

function generateId(): string {
  return Array.from({ length: 8 }, () =>
    Math.floor(Math.random() * 16).toString(16),
  ).join("");
}

async function fetchServerNames(servers: ServerEntry[]): Promise<void> {
  if (servers.length === 0) return;

  try {
    const response = await fetch("/api/discord/bulk-info", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(
        servers.map((s) => ({
          guild_id: s.guild_id,
          channel_id: s.channel_id,
        })),
      ),
    });

    if (!response.ok) return;

    const results: Array<{
      guild_id: string;
      guild_name: string;
      channel_id: string;
      channel_name: string;
    }> = await response.json();

    // Merge names into servers
    for (const result of results) {
      const server = servers.find(
        (s) =>
          s.guild_id === result.guild_id && s.channel_id === result.channel_id,
      );
      if (server) {
        server.guild_name = result.guild_name || undefined;
        server.channel_name = result.channel_name || undefined;
      }
    }
  } catch {
    // Silently fail - names are optional
  }
}

export function useConfig() {
  async function loadConfig() {
    loading.value = true;
    error.value = null;

    try {
      const response = await fetch("/api/config");
      if (!response.ok) {
        throw new Error("Failed to load configuration");
      }
      config.value = await response.json();

      // Fetch server/channel names from Discord API
      await fetchServerNames(config.value.servers);
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Unknown error";
    } finally {
      loading.value = false;
    }
  }

  async function saveConfig(servers: ServerEntry[], status?: Status) {
    loading.value = true;
    error.value = null;

    try {
      const response = await fetch("/api/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          servers,
          status: status ?? config.value.status,
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.message || "Failed to save configuration");
      }

      const result = await response.json();
      if (result.servers) {
        config.value.servers = result.servers;
        // Fetch server/channel names from Discord API
        await fetchServerNames(config.value.servers);
      }
      return true;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Unknown error";
      return false;
    } finally {
      loading.value = false;
    }
  }

  async function updateStatus(status: Status) {
    const success = await saveConfig(config.value.servers, status);
    if (success) {
      config.value.status = status;
    }
    return success;
  }

  async function acknowledgeTos() {
    loading.value = true;
    error.value = null;

    try {
      const response = await fetch("/api/acknowledge-tos", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ acknowledged: true }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.message || "Failed to acknowledge TOS");
      }

      config.value.tos_acknowledged = true;
      return true;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Unknown error";
      return false;
    } finally {
      loading.value = false;
    }
  }

  async function addServer(server: Omit<ServerEntry, "id">) {
    const newServer: ServerEntry = {
      ...server,
      id: generateId(),
    };

    const servers = [...config.value.servers, newServer];
    return saveConfig(servers);
  }

  async function updateServer(id: string, updates: Partial<ServerEntry>) {
    const servers = config.value.servers.map((s) =>
      s.id === id ? { ...s, ...updates } : s,
    );
    return saveConfig(servers);
  }

  async function deleteServer(id: string) {
    const servers = config.value.servers.filter((s) => s.id !== id);
    return saveConfig(servers);
  }

  function setConfig(newConfig: Configuration) {
    config.value = newConfig;
  }

  return {
    config,
    loading,
    error,
    loadConfig,
    saveConfig,
    updateStatus,
    acknowledgeTos,
    addServer,
    updateServer,
    deleteServer,
    setConfig,
  };
}
