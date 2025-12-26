import { defineStore } from "pinia";
import { ref } from "vue";

import type { ConnectionStatus } from "@/types";

export const useServersStore = defineStore("servers", () => {
  const actionLoading = ref<Map<string, boolean>>(new Map());

  async function executeAction(
    serverId: string,
    action: "exit" | "join" | "rejoin",
  ): Promise<{
    error?: string;
    newStatus?: ConnectionStatus;
    success: boolean;
  }> {
    actionLoading.value.set(serverId, true);

    try {
      const response = await fetch(`/api/servers/${serverId}/action`, {
        body: JSON.stringify({ action }),
        headers: { "Content-Type": "application/json" },
        method: "POST",
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          error: data.message || `Action '${action}' failed`,
          success: false,
        };
      }

      return {
        newStatus: data.new_status,
        success: true,
      };
    } catch (err) {
      return {
        error: err instanceof Error ? err.message : "Unknown error",
        success: false,
      };
    } finally {
      actionLoading.value.set(serverId, false);
    }
  }

  async function joinServer(serverId: string) {
    return executeAction(serverId, "join");
  }

  async function rejoinServer(serverId: string) {
    return executeAction(serverId, "rejoin");
  }

  async function exitServer(serverId: string) {
    return executeAction(serverId, "exit");
  }

  function isLoading(serverId: string): boolean {
    return actionLoading.value.get(serverId) || false;
  }

  return {
    actionLoading,
    exitServer,
    isLoading,
    joinServer,
    rejoinServer,
  };
});
