<script setup lang="ts">
import { ref, watch } from "vue";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import type { ServerEntry } from "@/types";

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

watch(
  () => props.open,
  (open) => {
    if (open && props.server) {
      guildId.value = props.server.guild_id;
      channelId.value = props.server.channel_id;
      connectOnStart.value = props.server.connect_on_start;
    } else if (open) {
      guildId.value = "";
      channelId.value = "";
      connectOnStart.value = true;
    }
  }
);

const isEdit = () => !!props.server;

function handleSubmit() {
  if (!guildId.value || !channelId.value) return;

  emit("save", {
    id: props.server?.id,
    guild_id: guildId.value,
    channel_id: channelId.value,
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
          {{ isEdit() ? "Update the server connection settings." : "Add a new server connection." }}
        </DialogDescription>
      </DialogHeader>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div class="space-y-2">
          <Label for="guild-id">Guild ID (Server ID)</Label>
          <Input
            id="guild-id"
            v-model="guildId"
            placeholder="123456789012345678"
            pattern="[0-9]{17,19}"
            required
          />
          <p class="text-xs text-muted-foreground">
            Right-click server → Copy ID (Enable Developer Mode in Discord settings)
          </p>
        </div>

        <div class="space-y-2">
          <Label for="channel-id">Channel ID (Voice Channel)</Label>
          <Input
            id="channel-id"
            v-model="channelId"
            placeholder="234567890123456789"
            pattern="[0-9]{17,19}"
            required
          />
          <p class="text-xs text-muted-foreground">
            Right-click voice channel → Copy ID
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
          <Button type="submit">
            {{ isEdit() ? "Save Changes" : "Add Server" }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
