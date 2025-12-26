<script setup lang="ts">
import ActivityLogView from "@/components/activity/ActivityLogView.vue";
import DashboardLayout from "@/components/layout/DashboardLayout.vue";
import { useDashboard } from "@/composables/useDashboard";

const { clearLogs, config, filteredLogs, logFilter, setLogFilter } = useDashboard();
</script>

<template>
  <DashboardLayout>
    <ActivityLogView
      :logs="filteredLogs"
      :filter="logFilter"
      :servers="config.servers"
      @clear="clearLogs"
      @update:filter="(f) => setLogFilter(f as 'all' | 'info' | 'warn' | 'error' | 'debug')"
    />
  </DashboardLayout>
</template>
