<script setup lang="ts">
import { computed } from "vue";

import type { Configuration, ConnectionStatus, LogEntry, ServerEntry } from "@/types";

import ActivityLogView from "@/components/activity/ActivityLogView.vue";
import DashboardView from "@/components/dashboard/DashboardView.vue";
import ServerDetailView from "@/components/servers/ServerDetailView.vue";
import { useNavigation } from "@/composables/useNavigation";

import ContentHeader from "./ContentHeader.vue";

const props = defineProps<{
  actionLoading: Map<string, boolean>;
  config: Configuration;
  logFilter: string;
  logs: LogEntry[];
  serverStatuses: Map<string, ConnectionStatus>;
  wsStatus: string;
}>();

const emit = defineEmits<{
  clearLogs: [];
  deleteServer: [id: string];
  editServer: [server: ServerEntry];
  exitServer: [id: string];
  joinServer: [id: string];
  logout: [];
  rejoinServer: [id: string];
  updateLogFilter: [filter: string];
  updateStatus: [status: string];
}>();

const { currentView, selectedServerId } = useNavigation();

const selectedServer = computed(() => {
  if (!selectedServerId.value) return null;
  return props.config.servers.find((s) => s.id === selectedServerId.value) || null;
});

const selectedServerStatus = computed(() => {
  if (!selectedServerId.value) return "disconnected";
  return props.serverStatuses.get(selectedServerId.value) || "disconnected";
});

const connectedCount = computed(() => {
  let count = 0;
  props.config.servers.forEach((server) => {
    if (props.serverStatuses.get(server.id) === "connected") {
      count++;
    }
  });
  return count;
});
</script>

<template>
  <main class="flex flex-1 flex-col overflow-hidden">
    <!-- Header -->
    <ContentHeader
      :status="config.status"
      :ws-status="wsStatus"
      :connected-count="connectedCount"
      :total-count="config.servers.length"
      @update-status="(status) => emit('updateStatus', status)"
      @logout="emit('logout')"
    />

    <!-- Content Area -->
    <div class="flex-1 overflow-y-auto p-6">
      <Transition name="fade" mode="out-in">
        <!-- Dashboard View -->
        <DashboardView
          v-if="currentView === 'dashboard'"
          :servers="config.servers"
          :server-statuses="serverStatuses"
          :logs="logs"
        />

        <!-- Server Detail View -->
        <ServerDetailView
          v-else-if="currentView === 'server' && selectedServer"
          :server="selectedServer"
          :status="selectedServerStatus"
          :is-loading="actionLoading.get(selectedServer.id) ?? false"
          :logs="logs.filter((l) => l.serverId === selectedServer?.id)"
          @edit="selectedServer && emit('editServer', selectedServer)"
          @delete="selectedServer && emit('deleteServer', selectedServer.id)"
          @join="selectedServer && emit('joinServer', selectedServer.id)"
          @rejoin="selectedServer && emit('rejoinServer', selectedServer.id)"
          @exit="selectedServer && emit('exitServer', selectedServer.id)"
        />

        <!-- Activity Log View -->
        <ActivityLogView
          v-else-if="currentView === 'activity'"
          :logs="logs"
          :filter="logFilter"
          :servers="config.servers"
          @clear="emit('clearLogs')"
          @update:filter="(filter) => emit('updateLogFilter', filter)"
        />
      </Transition>
    </div>
  </main>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
