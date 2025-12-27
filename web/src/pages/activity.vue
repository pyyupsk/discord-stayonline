<script setup lang="ts">
import ActivityLogView from "@/components/activity/ActivityLogView.vue";
import AppLayout from "@/components/layout/AppLayout.vue";
import { useDashboard } from "@/composables/useDashboard";

const { clearLogs, config, filteredLogs, logFilter, setLogFilter } = useDashboard();
</script>

<template>
  <AppLayout>
    <ActivityLogView
      :logs="filteredLogs"
      :filter="logFilter"
      :servers="config.servers"
      @clear="clearLogs"
      @update:filter="(f) => setLogFilter(f as 'all' | 'info' | 'warn' | 'error' | 'debug')"
    />
  </AppLayout>
</template>
