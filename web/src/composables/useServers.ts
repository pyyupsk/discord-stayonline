import { ref } from "vue";
import type { ConnectionStatus } from "@/types";

const actionLoading = ref<Map<string, boolean>>(new Map());

export function useServers() {
  async function executeAction(
    serverId: string,
    action: "join" | "rejoin" | "exit",
  ): Promise<{
    success: boolean;
    newStatus?: ConnectionStatus;
    error?: string;
  }> {
    actionLoading.value.set(serverId, true);

    try {
      const response = await fetch(`/api/servers/${serverId}/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });

      const data = await response.json();

      if (!response.ok) {
        return {
          success: false,
          error: data.message || `Action '${action}' failed`,
        };
      }

      return {
        success: true,
        newStatus: data.new_status,
      };
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : "Unknown error",
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
    joinServer,
    rejoinServer,
    exitServer,
    isLoading,
  };
}
