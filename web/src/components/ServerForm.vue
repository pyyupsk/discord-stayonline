<script setup lang="ts">
import { ref, watch, computed } from "vue";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Loader2 } from "lucide-vue-next";
import type { ServerEntry, GuildInfo, VoiceChannelInfo } from "@/types";

const props = defineProps<{
  open: boolean;
  server?: ServerEntry | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  save: [server: Omit<ServerEntry, "id"> & { id?: string }];
}>();

const guildId = ref("");
const channelId = ref("");
const connectOnStart = ref(true);

const guilds = ref<GuildInfo[]>([]);
const channels = ref<VoiceChannelInfo[]>([]);
const loadingGuilds = ref(false);
const loadingChannels = ref(false);
const errorMessage = ref("");

const selectedGuild = computed(() =>
  guilds.value.find((g) => g.id === guildId.value),
);
const selectedChannel = computed(() =>
  channels.value.find((c) => c.id === channelId.value),
);

async function fetchGuilds() {
  loadingGuilds.value = true;
  errorMessage.value = "";
  try {
    const response = await fetch("/api/discord/guilds");
    if (!response.ok) {
      throw new Error("Failed to fetch guilds");
    }
    guilds.value = await response.json();
  } catch {
    errorMessage.value = "Failed to load servers. Please try again.";
  } finally {
    loadingGuilds.value = false;
  }
}

async function fetchChannels(guildId: string) {
  if (!guildId) {
    channels.value = [];
    return;
  }
  loadingChannels.value = true;
  try {
    const response = await fetch(`/api/discord/guilds/${guildId}/channels`);
    if (!response.ok) {
      throw new Error("Failed to fetch channels");
    }
    channels.value = await response.json();
    // Sort by position
    channels.value.sort((a, b) => a.position - b.position);
  } catch {
    channels.value = [];
  } finally {
    loadingChannels.value = false;
  }
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      await fetchGuilds();
      if (props.server) {
        guildId.value = props.server.guild_id;
        channelId.value = props.server.channel_id;
        connectOnStart.value = props.server.connect_on_start;
        // Fetch channels for the existing guild
        await fetchChannels(props.server.guild_id);
      } else {
        guildId.value = "";
        channelId.value = "";
        connectOnStart.value = true;
        channels.value = [];
      }
    }
  },
);

watch(guildId, async (newGuildId, oldGuildId) => {
  if (newGuildId !== oldGuildId) {
    channelId.value = "";
    await fetchChannels(newGuildId);
  }
});

const isEdit = () => !!props.server;

function handleSubmit() {
  if (!guildId.value || !channelId.value) return;

  emit("save", {
    id: props.server?.id,
    guild_id: guildId.value,
    guild_name: selectedGuild.value?.name,
    channel_id: channelId.value,
    channel_name: selectedChannel.value?.name,
    connect_on_start: connectOnStart.value,
    priority: props.server?.priority ?? 1,
  });
}

function handleClose() {
  emit("update:open", false);
}
</script>

<template>
  <Dialog :open="props.open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-md">
      <DialogHeader>
        <DialogTitle>{{ isEdit() ? "Edit Server" : "Add Server" }}</DialogTitle>
        <DialogDescription>
          {{
            isEdit()
              ? "Update the server connection settings."
              : "Add a new server connection."
          }}
        </DialogDescription>
      </DialogHeader>

      <div
        v-if="errorMessage"
        class="rounded-md bg-destructive/10 p-3 text-sm text-destructive"
      >
        {{ errorMessage }}
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div class="space-y-2">
          <Label for="guild-select">Server</Label>
          <Select v-model="guildId" :disabled="loadingGuilds">
            <SelectTrigger id="guild-select" class="w-full">
              <Loader2 v-if="loadingGuilds" class="animate-spin" />
              <SelectValue v-else placeholder="Select a server" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="guild in guilds"
                :key="guild.id"
                :value="guild.id"
              >
                {{ guild.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="space-y-2">
          <Label for="channel-select">Voice Channel</Label>
          <Select v-model="channelId" :disabled="!guildId || loadingChannels">
            <SelectTrigger id="channel-select" class="w-full">
              <Loader2 v-if="loadingChannels" class="animate-spin" />
              <SelectValue
                v-else
                :placeholder="
                  guildId ? 'Select a voice channel' : 'Select a server first'
                "
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="channel in channels"
                :key="channel.id"
                :value="channel.id"
              >
                {{ channel.name }}
              </SelectItem>
            </SelectContent>
          </Select>
          <p
            v-if="guildId && channels.length === 0 && !loadingChannels"
            class="text-xs text-muted-foreground"
          >
            No voice channels available in this server
          </p>
        </div>

        <div class="flex items-center justify-between">
          <Label for="connect-on-start" class="cursor-pointer">
            Connect automatically on startup
          </Label>
          <Switch id="connect-on-start" v-model:checked="connectOnStart" />
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" @click="handleClose">
            Cancel
          </Button>
          <Button type="submit" :disabled="!guildId || !channelId">
            {{ isEdit() ? "Save Changes" : "Add Server" }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
