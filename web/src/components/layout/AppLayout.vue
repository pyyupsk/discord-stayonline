<script setup lang="ts">
import type { Configuration, ConnectionStatus, LogEntry, ServerEntry } from "@/types";

import MainContent from "./MainContent.vue";
import Sidebar from "./Sidebar.vue";

defineProps<{
  actionLoading: Map<string, boolean>;
  config: Configuration;
  logFilter: string;
  logs: LogEntry[];
  serverStatuses: Map<string, ConnectionStatus>;
  wsStatus: string;
}>();

const emit = defineEmits<{
  addServer: [];
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
</script>

<template>
  <div class="bg-background flex h-screen w-full overflow-hidden">
    <!-- Sidebar -->
    <Sidebar
      :servers="config.servers"
      :server-statuses="serverStatuses"
      :action-loading="actionLoading"
      @add-server="emit('addServer')"
      @join-server="(id) => emit('joinServer', id)"
      @exit-server="(id) => emit('exitServer', id)"
    />

    <!-- Main Content -->
    <MainContent
      :config="config"
      :server-statuses="serverStatuses"
      :logs="logs"
      :log-filter="logFilter"
      :ws-status="wsStatus"
      :action-loading="actionLoading"
      @edit-server="(server) => emit('editServer', server)"
      @delete-server="(id) => emit('deleteServer', id)"
      @join-server="(id) => emit('joinServer', id)"
      @rejoin-server="(id) => emit('rejoinServer', id)"
      @exit-server="(id) => emit('exitServer', id)"
      @update-status="(status) => emit('updateStatus', status)"
      @clear-logs="emit('clearLogs')"
      @update-log-filter="(filter) => emit('updateLogFilter', filter)"
      @logout="emit('logout')"
    />
  </div>
</template>
