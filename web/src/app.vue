<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";

import type { ConnectionStatus, ServerEntry, Status } from "@/types";

import AppLayout from "@/components/layout/AppLayout.vue";
import LoginForm from "@/components/LoginForm.vue";
import ServerForm from "@/components/ServerForm.vue";
import TosModal from "@/components/TosModal.vue";
import { useAuth } from "@/composables/useAuth";
import { useConfig } from "@/composables/useConfig";
import { useServers } from "@/composables/useServers";
import { useWebSocket } from "@/composables/useWebSocket";

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
const { authenticated, authRequired, checkAuth, logout } = useAuth();

const showServerForm = ref(false);
const editingServer = ref<null | ServerEntry>(null);
const initialLoading = ref(true);

const needsLogin = computed(() => authRequired.value && !authenticated.value);

// Create a reactive map for server statuses
const serverStatusMap = computed(() => {
  const map = new Map<string, ConnectionStatus>();
  config.value.servers.forEach((server) => {
    map.set(server.id, getServerStatus(server.id));
  });
  return map;
});

onMounted(async () => {
  await checkAuth();

  if (authenticated.value || !authRequired.value) {
    await loadConfig();
    updateServerNamesFromConfig(config.value);

    if (config.value.tos_acknowledged) {
      connect();
      setOnConfigChanged((newConfig) => {
        setConfig(newConfig);
        updateServerNamesFromConfig(newConfig);
      });
    }
  }

  initialLoading.value = false;
});

async function handleLogout() {
  await logout();
}

// Watch for successful login to load config
watch(authenticated, async (isAuthenticated) => {
  if (isAuthenticated && !initialLoading.value) {
    await loadConfig();
    updateServerNamesFromConfig(config.value);

    if (config.value.tos_acknowledged) {
      connect();
      setOnConfigChanged((newConfig) => {
        setConfig(newConfig);
        updateServerNamesFromConfig(newConfig);
      });
    }
  }
});

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
</script>

<template>
  <!-- Loading State -->
  <div v-if="initialLoading" class="bg-background flex h-screen items-center justify-center">
    <div class="flex flex-col items-center gap-3">
      <div class="border-muted border-t-foreground h-8 w-8 animate-spin rounded-full border-2" />
      <p class="text-muted-foreground text-sm">Loading...</p>
    </div>
  </div>

  <!-- Login Form -->
  <LoginForm v-else-if="needsLogin" />

  <!-- Main App (after auth) -->
  <template v-else>
    <!-- TOS Modal -->
    <TosModal :open="!config.tos_acknowledged" @acknowledge="handleAcknowledgeTos" />

    <!-- Main App Layout -->
    <AppLayout
      v-if="config.tos_acknowledged"
      :config="config"
      :server-statuses="serverStatusMap"
      :logs="filteredLogs"
      :log-filter="logFilter"
      :ws-status="wsStatus"
      :action-loading="actionLoading"
      @add-server="handleAddServer"
      @edit-server="handleEditServer"
      @delete-server="handleDeleteServer"
      @join-server="handleJoin"
      @rejoin-server="handleRejoin"
      @exit-server="handleExit"
      @update-status="handleStatusChange"
      @clear-logs="clearLogs"
      @update-log-filter="
        (f: string) => setLogFilter(f as 'all' | 'info' | 'warn' | 'error' | 'debug')
      "
      @logout="handleLogout"
    />

    <!-- Server Form Dialog -->
    <ServerForm v-model:open="showServerForm" :server="editingServer" @save="handleSaveServer" />
  </template>
</template>
