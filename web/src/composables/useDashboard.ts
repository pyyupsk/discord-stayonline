import { computed, onMounted, ref } from "vue";

import type { ConnectionStatus, ServerEntry, Status } from "@/types";

import { useAuth } from "./useAuth";
import { useConfig } from "./useConfig";
import { useServers } from "./useServers";
import { useWebSocket } from "./useWebSocket";

const initialized = ref(false);
const initialLoading = ref(true);

export function useDashboard() {
  const {
    acknowledgeTos,
    addServer,
    config,
    deleteServer,
    loadConfig,
    setConfig,
    updateServer,
    updateStatus,
  } = useConfig();

  const {
    addLog,
    clearLogs,
    connect,
    filteredLogs,
    getServerStatus,
    logFilter,
    setLogFilter,
    setOnConfigChanged,
    updateServerNamesFromConfig,
    wsStatus,
  } = useWebSocket();

  const { actionLoading, exitServer, joinServer, rejoinServer } = useServers();
  const { logout } = useAuth();

  const showServerForm = ref(false);
  const editingServer = ref<null | ServerEntry>(null);

  const serverStatusMap = computed(() => {
    const map = new Map<string, ConnectionStatus>();
    config.value.servers.forEach((server) => {
      map.set(server.id, getServerStatus(server.id));
    });
    return map;
  });

  const connectedCount = computed(() => {
    let count = 0;
    config.value.servers.forEach((server) => {
      if (serverStatusMap.value.get(server.id) === "connected") {
        count++;
      }
    });
    return count;
  });

  async function initialize() {
    if (initialized.value) {
      initialLoading.value = false;
      return;
    }

    await loadConfig();
    updateServerNamesFromConfig(config.value);

    if (config.value.tos_acknowledged) {
      connect();
      setOnConfigChanged((newConfig) => {
        setConfig(newConfig);
        updateServerNamesFromConfig(newConfig);
      });
    }

    initialized.value = true;
    initialLoading.value = false;
  }

  async function handleAcknowledgeTos() {
    const success = await acknowledgeTos();
    if (success) {
      connect();
      setOnConfigChanged((newConfig) => {
        setConfig(newConfig);
      });
      addLog("info", "Terms of Service acknowledged");
    }
  }

  function handleAddServer() {
    editingServer.value = null;
    showServerForm.value = true;
  }

  async function handleDeleteServer(id: string) {
    if (!confirm("Are you sure you want to delete this server?")) return;

    const success = await deleteServer(id);
    if (success) {
      addLog("info", "Server deleted");
    }
  }

  function handleEditServer(server: ServerEntry) {
    editingServer.value = server;
    showServerForm.value = true;
  }

  async function handleExit(id: string) {
    const result = await exitServer(id);
    if (!result.success) {
      addLog("error", result.error || "Exit failed");
    }
  }

  async function handleJoin(id: string) {
    const result = await joinServer(id);
    if (!result.success) {
      addLog("error", result.error || "Join failed");
    }
  }

  async function handleLogout() {
    await logout();
    globalThis.location.href = "/login";
  }

  async function handleRejoin(id: string) {
    const result = await rejoinServer(id);
    if (!result.success) {
      addLog("error", result.error || "Rejoin failed");
    }
  }

  async function handleSaveServer(server: Omit<ServerEntry, "id"> & { id?: string }) {
    let success: boolean;

    if (server.id) {
      success = await updateServer(server.id, server);
      if (success) addLog("info", "Server updated");
    } else {
      success = await addServer(server);
      if (success) addLog("info", "Server added");
    }

    if (success) {
      showServerForm.value = false;
      editingServer.value = null;
    }
  }

  async function handleStatusChange(status: string) {
    const success = await updateStatus(status as Status);
    if (success) {
      addLog("info", `Status changed to ${status}`);
    }
  }

  onMounted(() => {
    initialize();
  });

  return {
    // State
    actionLoading,
    // Actions
    clearLogs,
    config,
    connectedCount,
    editingServer,
    filteredLogs,
    handleAcknowledgeTos,
    handleAddServer,
    handleDeleteServer,
    handleEditServer,

    handleExit,
    handleJoin,
    handleLogout,
    handleRejoin,
    handleSaveServer,
    handleStatusChange,
    initialLoading,
    logFilter,
    serverStatusMap,
    setLogFilter,
    showServerForm,
    wsStatus,
  };
}
