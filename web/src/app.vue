<script setup lang="ts">
import { LogOut, Plus, Wifi, WifiOff } from "lucide-vue-next";
import { computed, onMounted, ref, watch } from "vue";

import type { ServerEntry, Status } from "@/types";

import ActivityLog from "@/components/ActivityLog.vue";
import GlobalStatus from "@/components/GlobalStatus.vue";
import LoginForm from "@/components/LoginForm.vue";
import ServerCard from "@/components/ServerCard.vue";
import ServerForm from "@/components/ServerForm.vue";
import TosModal from "@/components/TosModal.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
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
  wsStatus,
} = useWebSocket();
const { exitServer, isLoading, joinServer, rejoinServer } = useServers();
const { authenticated, authRequired, checkAuth, loading: authLoading, logout } = useAuth();

const showServerForm = ref(false);
const editingServer = ref<null | ServerEntry>(null);
const initialLoading = ref(true);

const isConnected = computed(() => wsStatus.value === "connected");
const needsLogin = computed(() => authRequired.value && !authenticated.value);

onMounted(async () => {
  await checkAuth();

  if (authenticated.value || !authRequired.value) {
    await loadConfig();

    if (config.value.tos_acknowledged) {
      connect();
      setOnConfigChanged((newConfig) => {
        setConfig(newConfig);
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

    if (config.value.tos_acknowledged) {
      connect();
      setOnConfigChanged((newConfig) => {
        setConfig(newConfig);
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

async function handleDeleteServer(server: ServerEntry) {
  if (!confirm("Are you sure you want to delete this server?")) return;

  const success = await deleteServer(server.id);
  if (success) {
    addLog("info", "Server deleted");
  }
}

function handleEditServer(server: ServerEntry) {
  editingServer.value = server;
  showServerForm.value = true;
}

async function handleExit(server: ServerEntry) {
  if (!confirm("Exit will close the connection. Continue?")) return;

  const result = await exitServer(server.id);
  if (!result.success) {
    addLog("error", result.error || "Exit failed");
  }
}

async function handleJoin(server: ServerEntry) {
  const result = await joinServer(server.id);
  if (!result.success) {
    addLog("error", result.error || "Join failed");
  }
}

async function handleRejoin(server: ServerEntry) {
  if (!confirm("Rejoin will close the current connection. Continue?")) return;

  const result = await rejoinServer(server.id);
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

async function handleStatusChange(status: Status) {
  const success = await updateStatus(status);
  if (success) {
    addLog("info", `Status changed to ${status}`);
  }
}
</script>

<template>
  <!-- Loading State -->
  <div v-if="initialLoading" class="flex min-h-screen items-center justify-center">
    <p class="text-muted-foreground">Loading...</p>
  </div>

  <!-- Login Form -->
  <LoginForm v-else-if="needsLogin" />

  <!-- Main App (after auth) -->
  <template v-else>
    <!-- TOS Modal -->
    <TosModal :open="!config.tos_acknowledged" @acknowledge="handleAcknowledgeTos" />

    <!-- Main App -->
    <div v-if="config.tos_acknowledged" class="bg-background min-h-screen">
      <!-- Header -->
      <header class="border-b">
        <div class="container mx-auto flex items-center justify-between px-4 py-4">
          <h1 class="text-xl font-semibold">Discord Stay Online</h1>
          <div class="flex items-center gap-3">
            <Badge :variant="isConnected ? 'default' : 'secondary'">
              <component :is="isConnected ? Wifi : WifiOff" />
              {{ isConnected ? "Connected" : "Disconnected" }}
            </Badge>
            <Button
              v-if="authRequired"
              variant="ghost"
              size="icon"
              :disabled="authLoading"
              title="Logout"
              @click="handleLogout"
            >
              <LogOut />
            </Button>
          </div>
        </div>
      </header>

      <!-- Main Content -->
      <main class="container mx-auto space-y-6 px-4 py-6">
        <!-- Global Status -->
        <section class="flex items-center justify-between">
          <GlobalStatus :status="config.status" @change="handleStatusChange" />
        </section>

        <Separator />

        <!-- Server List -->
        <section class="space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-medium">Server Connections</h2>
            <Button size="sm" :disabled="config.servers.length >= 15" @click="handleAddServer">
              <Plus />
              Add Server
            </Button>
          </div>

          <div
            v-if="config.servers.length === 0"
            class="rounded-lg border border-dashed p-8 text-center"
          >
            <p class="text-muted-foreground">
              No servers configured. Click "Add Server" to get started.
            </p>
          </div>

          <div v-else class="space-y-3">
            <ServerCard
              v-for="server in config.servers"
              :key="server.id"
              :server="server"
              :status="getServerStatus(server.id)"
              :loading="isLoading(server.id)"
              @join="handleJoin(server)"
              @rejoin="handleRejoin(server)"
              @exit="handleExit(server)"
              @edit="handleEditServer(server)"
              @delete="handleDeleteServer(server)"
            />
          </div>
        </section>

        <Separator />

        <!-- Activity Log -->
        <section>
          <ActivityLog
            :logs="filteredLogs"
            :filter="logFilter"
            @clear="clearLogs"
            @update:filter="setLogFilter"
          />
        </section>
      </main>
    </div>

    <!-- Server Form Dialog -->
    <ServerForm v-model:open="showServerForm" :server="editingServer" @save="handleSaveServer" />
  </template>
</template>
