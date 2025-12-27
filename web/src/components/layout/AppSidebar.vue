<script setup lang="ts">
import { Activity, LayoutDashboard, Plus, Power, PowerOff, Server } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, ServerEntry } from "@/types";

import { Button } from "@/components/ui/button";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from "@/components/ui/sidebar";
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

function getStatusColor(status: ConnectionStatus) {
  switch (status) {
    case "backoff":
    case "connecting":
      return "bg-warning";
    case "connected":
      return "bg-success";
    case "error":
      return "bg-destructive";
    default:
      return "bg-muted-foreground";
  }
}
</script>

<template>
  <Sidebar collapsible="icon">
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg" as-child>
            <RouterLink to="/">
              <div
                class="bg-primary text-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg"
              >
                <Server class="size-4" />
              </div>
              <div class="grid flex-1 text-left text-sm leading-tight">
                <span class="truncate font-semibold">Discord Stay Online</span>
                <span class="text-muted-foreground truncate text-xs">
                  {{ connectedCount }}/{{ servers.length }} connected
                </span>
              </div>
            </RouterLink>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>

    <SidebarContent>
      <!-- Navigation -->
      <SidebarGroup>
        <SidebarGroupLabel>Navigation</SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton :is-active="isDashboard" @click="navigateToDashboard">
                <LayoutDashboard />
                <span>Dashboard</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton :is-active="isActivityView" @click="navigateToActivity">
                <Activity />
                <span>Activity</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>

      <SidebarSeparator />

      <!-- Servers -->
      <SidebarGroup>
        <SidebarGroupLabel>
          Servers
          <Button variant="ghost" size="icon" class="ml-auto size-5" @click="emit('addServer')">
            <Plus class="size-3" />
          </Button>
        </SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem v-for="server in servers" :key="server.id">
              <SidebarMenuButton
                :is-active="selectedServerId === server.id"
                @click="navigateToServer(server.id)"
              >
                <div class="relative">
                  <div
                    class="bg-muted text-muted-foreground flex size-5 items-center justify-center rounded text-xs font-medium"
                  >
                    {{ (server.guild_name || server.guild_id).slice(0, 2).toUpperCase() }}
                  </div>
                  <span
                    class="border-background absolute -right-0.5 -bottom-0.5 size-2 rounded-full border"
                    :class="getStatusColor(serverStatuses.get(server.id) || 'disconnected')"
                  />
                </div>
                <span class="truncate">
                  {{ server.guild_name || `Server ${server.guild_id.slice(-4)}` }}
                </span>
              </SidebarMenuButton>
              <SidebarMenuBadge v-if="actionLoading.get(server.id)">
                <span class="bg-warning size-2 animate-pulse rounded-full" />
              </SidebarMenuBadge>
            </SidebarMenuItem>

            <SidebarMenuItem v-if="servers.length === 0">
              <SidebarMenuButton disabled>
                <span class="text-muted-foreground text-xs">No servers configured</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            :disabled="connectedCount === servers.length || servers.length === 0"
            @click="connectAll"
          >
            <Power class="text-success" />
            <span>Connect All</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
        <SidebarMenuItem>
          <SidebarMenuButton :disabled="connectedCount === 0" @click="disconnectAll">
            <PowerOff class="text-destructive" />
            <span>Disconnect All</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>
  </Sidebar>
</template>
