<script setup lang="ts">
import { Activity, LayoutDashboard, Plus, Power, PowerOff } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, ServerEntry } from "@/types";

import ServerIcon from "@/components/servers/ServerIcon.vue";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useNavigation } from "@/composables/useNavigation";

const props = defineProps<{
  actionLoading: Map<string, boolean>;
  servers: ServerEntry[];
  serverStatuses: Map<string, ConnectionStatus>;
}>();

const emit = defineEmits<{
  addServer: [];
  exitServer: [id: string];
  joinServer: [id: string];
}>();

const {
  isActivityView,
  isDashboard,
  navigateToActivity,
  navigateToDashboard,
  navigateToServer,
  selectedServerId,
} = useNavigation();

function connectAll() {
  props.servers.forEach((server) => {
    const status = props.serverStatuses.get(server.id);
    if (status !== "connected" && status !== "connecting") {
      emit("joinServer", server.id);
    }
  });
}

function disconnectAll() {
  props.servers.forEach((server) => {
    const status = props.serverStatuses.get(server.id);
    if (status === "connected" || status === "connecting") {
      emit("exitServer", server.id);
    }
  });
}

const connectedCount = computed(() => {
  let count = 0;
  props.servers.forEach((server) => {
    if (props.serverStatuses.get(server.id) === "connected") {
      count++;
    }
  });
  return count;
});
</script>

<template>
  <TooltipProvider :delay-duration="100">
    <aside
      class="discord-sidebar border-border/50 bg-sidebar flex h-full w-[72px] flex-col items-center border-r py-3"
    >
      <!-- Dashboard Button -->
      <Tooltip>
        <TooltipTrigger as-child>
          <button
            class="discord-nav-icon mb-2"
            :class="{ active: isDashboard }"
            @click="navigateToDashboard"
          >
            <LayoutDashboard class="h-5 w-5" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="right" :side-offset="10">
          <p>Dashboard</p>
        </TooltipContent>
      </Tooltip>

      <Separator class="bg-border/50 mx-auto mb-2 w-8" />

      <!-- Server List -->
      <div class="sidebar-scrollbar flex flex-1 flex-col items-center gap-2 overflow-y-auto px-3">
        <TransitionGroup name="server-list">
          <ServerIcon
            v-for="server in servers"
            :key="server.id"
            :server="server"
            :status="serverStatuses.get(server.id) || 'disconnected'"
            :is-selected="selectedServerId === server.id"
            :is-loading="actionLoading.get(server.id) || false"
            @click="navigateToServer(server.id)"
          />
        </TransitionGroup>

        <!-- Add Server Button -->
        <Tooltip>
          <TooltipTrigger as-child>
            <button class="discord-nav-icon add-server" @click="emit('addServer')">
              <Plus class="h-5 w-5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="right" :side-offset="10">
            <p>Add Server</p>
          </TooltipContent>
        </Tooltip>
      </div>

      <Separator class="bg-border/50 mx-auto mt-2 mb-2 w-8" />

      <!-- Quick Actions -->
      <div class="flex flex-col items-center gap-2">
        <!-- Activity Log -->
        <Tooltip>
          <TooltipTrigger as-child>
            <button
              class="discord-nav-icon"
              :class="{ active: isActivityView }"
              @click="navigateToActivity"
            >
              <Activity class="h-5 w-5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="right" :side-offset="10">
            <p>Activity Log</p>
          </TooltipContent>
        </Tooltip>

        <!-- Connect All -->
        <Tooltip>
          <TooltipTrigger as-child>
            <button
              class="discord-nav-icon success"
              :disabled="connectedCount === servers.length"
              @click="connectAll"
            >
              <Power class="h-5 w-5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="right" :side-offset="10">
            <p>Connect All</p>
          </TooltipContent>
        </Tooltip>

        <!-- Disconnect All -->
        <Tooltip>
          <TooltipTrigger as-child>
            <button
              class="discord-nav-icon destructive"
              :disabled="connectedCount === 0"
              @click="disconnectAll"
            >
              <PowerOff class="h-5 w-5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="right" :side-offset="10">
            <p>Disconnect All</p>
          </TooltipContent>
        </Tooltip>
      </div>
    </aside>
  </TooltipProvider>
</template>

<style scoped>
.discord-nav-icon {
  display: flex;
  height: 48px;
  width: 48px;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  border-radius: 24px;
  background: var(--muted);
  color: var(--muted-foreground);
  transition: all 0.15s ease;
}

.discord-nav-icon:hover {
  border-radius: 16px;
  background: var(--primary);
  color: var(--primary-foreground);
}

.discord-nav-icon.active {
  border-radius: 16px;
  background: var(--primary);
  color: var(--primary-foreground);
}

.discord-nav-icon.add-server {
  background: transparent;
  color: var(--success);
}

.discord-nav-icon.add-server:hover {
  background: var(--success);
  color: var(--success-foreground);
}

.discord-nav-icon.success:hover:not(:disabled) {
  background: var(--success);
  color: var(--success-foreground);
}

.discord-nav-icon.destructive:hover:not(:disabled) {
  background: var(--destructive);
  color: var(--destructive-foreground);
}

.discord-nav-icon:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.sidebar-scrollbar {
  scrollbar-width: thin;
  scrollbar-color: var(--muted) transparent;
}

.sidebar-scrollbar::-webkit-scrollbar {
  width: 4px;
}

.sidebar-scrollbar::-webkit-scrollbar-thumb {
  background: var(--muted);
  border-radius: 2px;
}

.server-list-enter-active,
.server-list-leave-active {
  transition: all 0.2s ease;
}

.server-list-enter-from,
.server-list-leave-to {
  opacity: 0;
  transform: scale(0.8);
}
</style>
