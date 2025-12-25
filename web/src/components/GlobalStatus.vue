<script setup lang="ts">
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import type { Status } from "@/types";

const props = defineProps<{
  status: Status;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  change: [status: Status];
}>();

const statusOptions: { value: Status; label: string }[] = [
  { value: "online", label: "Online" },
  { value: "idle", label: "Idle" },
  { value: "dnd", label: "Do Not Disturb" },
];

function handleChange(value: unknown) {
  if (typeof value === "string") {
    emit("change", value as Status);
  }
}
</script>

<template>
  <div class="flex items-center gap-3">
    <Label for="status" class="text-muted-foreground whitespace-nowrap">
      Account Status
    </Label>
    <Select :modelValue="props.status" @update:modelValue="handleChange" :disabled="props.disabled">
      <SelectTrigger id="status" class="w-[160px]">
        <SelectValue placeholder="Select status" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem
          v-for="option in statusOptions"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </SelectItem>
      </SelectContent>
    </Select>
  </div>
</template>
