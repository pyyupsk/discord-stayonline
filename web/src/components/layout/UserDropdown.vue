<script setup lang="ts">
import { useColorMode } from "@vueuse/core";
import { Check, CircleDot, LogOut, Moon, Palette, Sun } from "lucide-vue-next";
import { computed, onMounted, ref } from "vue";

import type { Status, UserInfo } from "@/types";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";

const props = defineProps<{
  class?: string;
  status: Status;
}>();

const emit = defineEmits<{
  logout: [];
  statusChange: [status: Status];
}>();

const mode = useColorMode();

const user = ref<null | UserInfo>(null);
const loading = ref(true);

const displayName = computed(() => {
  if (!user.value) return "";
  return user.value.global_name || user.value.username;
});

const avatarUrl = computed(() => {
  if (!user.value?.avatar) return undefined;
  return `https://cdn.discordapp.com/avatars/${user.value.id}/${user.value.avatar}.png?size=64`;
});

const initials = computed(() => {
  if (!user.value) return "?";
  return (user.value.global_name || user.value.username).slice(0, 2).toUpperCase();
});

async function fetchUser() {
  try {
    const response = await fetch("/api/discord/user");
    if (response.ok) {
      user.value = await response.json();
    }
  } catch {
    // Ignore errors, user just won't be displayed
  } finally {
    loading.value = false;
  }
}

onMounted(fetchUser);

function getStatusColor(status: Status) {
  switch (status) {
    case "dnd":
      return "bg-destructive";
    case "idle":
      return "bg-yellow-500";
    case "online":
      return "bg-green-500";
  }
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger class="focus:outline-none" :class="props.class">
      <div class="hover:bg-accent flex items-center gap-2 rounded-lg px-2 py-1.5">
        <template v-if="loading">
          <Skeleton class="h-8 w-8 rounded-full" />
          <Skeleton class="h-4 w-20" />
        </template>
        <template v-else>
          <div class="relative">
            <Avatar class="h-8 w-8">
              <AvatarImage v-if="avatarUrl" :src="avatarUrl" :alt="displayName" />
              <AvatarFallback>{{ initials }}</AvatarFallback>
            </Avatar>
            <span
              class="border-background absolute -right-0.5 -bottom-0.5 size-3 rounded-full border-2"
              :class="getStatusColor(props.status)"
            />
          </div>
          <span class="text-sm font-medium">{{ displayName }}</span>
        </template>
      </div>
    </DropdownMenuTrigger>

    <DropdownMenuContent align="end" class="w-56">
      <DropdownMenuLabel v-if="user" class="font-normal">
        <div class="flex flex-col space-y-1">
          <p class="text-sm font-medium">{{ displayName }}</p>
          <p class="text-muted-foreground text-xs">@{{ user.username }}</p>
        </div>
      </DropdownMenuLabel>
      <DropdownMenuSeparator v-if="user" />

      <DropdownMenuGroup>
        <!-- Status Submenu -->
        <DropdownMenuSub>
          <DropdownMenuSubTrigger>
            <CircleDot class="mr-2 h-4 w-4" />
            <span>Status</span>
          </DropdownMenuSubTrigger>
          <DropdownMenuPortal>
            <DropdownMenuSubContent>
              <DropdownMenuRadioGroup
                :model-value="props.status"
                @update:model-value="emit('statusChange', $event as Status)"
              >
                <DropdownMenuRadioItem value="online">
                  Online
                  <span class="h-2.5 w-2.5 rounded-full bg-green-500" />
                </DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="idle">
                  Idle
                  <span class="h-2.5 w-2.5 rounded-full bg-yellow-500" />
                </DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="dnd">
                  Do Not Disturb
                  <span class="bg-destructive h-2.5 w-2.5 rounded-full" />
                </DropdownMenuRadioItem>
              </DropdownMenuRadioGroup>
            </DropdownMenuSubContent>
          </DropdownMenuPortal>
        </DropdownMenuSub>

        <!-- Theme Submenu -->
        <DropdownMenuSub>
          <DropdownMenuSubTrigger>
            <Palette class="mr-2 h-4 w-4" />
            <span>Theme</span>
          </DropdownMenuSubTrigger>
          <DropdownMenuPortal>
            <DropdownMenuSubContent>
              <DropdownMenuItem @click="mode = 'light'">
                <Sun class="h-4 w-4" />
                Light
                <Check v-if="mode === 'light'" class="ml-auto h-4 w-4" />
              </DropdownMenuItem>
              <DropdownMenuItem @click="mode = 'dark'">
                <Moon class="h-4 w-4" />
                Dark
                <Check v-if="mode === 'dark'" class="ml-auto h-4 w-4" />
              </DropdownMenuItem>
              <DropdownMenuItem @click="mode = 'auto'">
                <Palette class="h-4 w-4" />
                System
                <Check v-if="mode === 'auto'" class="ml-auto h-4 w-4" />
              </DropdownMenuItem>
            </DropdownMenuSubContent>
          </DropdownMenuPortal>
        </DropdownMenuSub>
      </DropdownMenuGroup>

      <DropdownMenuSeparator />

      <DropdownMenuItem class="text-destructive" @click="emit('logout')">
        <LogOut />
        Logout
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
