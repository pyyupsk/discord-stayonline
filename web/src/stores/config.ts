import { defineStore } from "pinia";
import { ref } from "vue";

import type { Configuration, ServerEntry, Status } from "@/types";

export const useConfigStore = defineStore("config", () => {
  const config = ref<Configuration>({
    servers: [],
    status: "online",
    tos_acknowledged: false,
  });
  const loading = ref(false);
  const error = ref<null | string>(null);

  async function loadConfig() {
    loading.value = true;
    error.value = null;

    try {
      const response = await fetch("/api/config");
      if (!response.ok) {
        throw new Error("Failed to load configuration");
      }
      config.value = await response.json();

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
        body: JSON.stringify({
          servers,
          status: status ?? config.value.status,
        }),
        headers: { "Content-Type": "application/json" },
        method: "POST",
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.message || "Failed to save configuration");
      }

      const result = await response.json();
      if (result.servers) {
        config.value.servers = result.servers;
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
        body: JSON.stringify({ acknowledged: true }),
        headers: { "Content-Type": "application/json" },
        method: "POST",
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
    const servers = config.value.servers.map((s) => (s.id === id ? { ...s, ...updates } : s));
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
    acknowledgeTos,
    addServer,
    config,
    deleteServer,
    error,
    loadConfig,
    loading,
    saveConfig,
    setConfig,
    updateServer,
    updateStatus,
  };
});

async function fetchServerNames(servers: ServerEntry[]): Promise<void> {
  if (servers.length === 0) return;

  try {
    const response = await fetch("/api/discord/bulk-info", {
      body: JSON.stringify(
        servers.map((s) => ({
          channel_id: s.channel_id,
          guild_id: s.guild_id,
        })),
      ),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    });

    if (!response.ok) return;

    const results: Array<{
      channel_id: string;
      channel_name: string;
      guild_id: string;
      guild_name: string;
    }> = await response.json();

    for (const result of results) {
      const server = servers.find(
        (s) => s.guild_id === result.guild_id && s.channel_id === result.channel_id,
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

function generateId(): string {
  return Array.from({ length: 8 }, () => Math.floor(Math.random() * 16).toString(16)).join("");
}
