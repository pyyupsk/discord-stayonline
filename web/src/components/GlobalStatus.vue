<script setup lang="ts">
import type { Status } from "@/types";

import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const props = defineProps<{
  disabled?: boolean;
  status: Status;
}>();

const emit = defineEmits<{
  change: [status: Status];
}>();

const statusOptions: { label: string; value: Status }[] = [
  { label: "Online", value: "online" },
  { label: "Idle", value: "idle" },
  { label: "Do Not Disturb", value: "dnd" },
];

function handleChange(value: unknown) {
  if (typeof value === "string") {
    emit("change", value as Status);
  }
}
</script>

<template>
  <div class="flex items-center gap-3">
    <Label for="status" class="text-muted-foreground whitespace-nowrap"> Account Status </Label>
    <Select
      :model-value="props.status"
      :disabled="props.disabled"
      @update:model-value="handleChange"
    >
      <SelectTrigger id="status" class="w-[160px]">
        <SelectValue placeholder="Select status" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem v-for="option in statusOptions" :key="option.value" :value="option.value">
          {{ option.label }}
        </SelectItem>
      </SelectContent>
    </Select>
  </div>
</template>
