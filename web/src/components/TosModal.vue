<script setup lang="ts">
import { ref } from "vue";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { TriangleAlert } from "lucide-vue-next";

const props = defineProps<{
  open: boolean;
}>();

const emit = defineEmits<{
  acknowledge: [];
}>();

const loading = ref(false);

async function handleAcknowledge() {
  loading.value = true;
  emit("acknowledge");
}
</script>

<template>
  <Dialog :open="props.open">
    <DialogContent
      class="max-w-lg"
      :hideCloseButton="true"
      @escapeKeyDown.prevent
      @pointerDownOutside.prevent
      @interactOutside.prevent
    >
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2 text-destructive">
          <TriangleAlert class="h-5 w-5" />
          Terms of Service Warning
        </DialogTitle>
        <DialogDescription>
          Please read and acknowledge before proceeding
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-4">
        <div
          class="rounded-lg border border-destructive/50 bg-destructive/10 p-4"
        >
          <p class="mb-3 font-semibold text-destructive">
            IMPORTANT: READ BEFORE PROCEEDING
          </p>
          <p class="mb-3 text-sm text-muted-foreground">
            This tool uses Discord user tokens to maintain presence status.
            Using user tokens with automated tools
            <strong class="text-foreground"
              >may violate Discord's Terms of Service</strong
            >
            and could result in:
          </p>
          <ul
            class="mb-3 list-inside list-disc space-y-1 text-sm text-muted-foreground"
          >
            <li>Account suspension</li>
            <li>Account termination</li>
            <li>Loss of access to Discord services</li>
          </ul>
        </div>

        <div class="text-sm text-muted-foreground">
          <p class="mb-2">By clicking the button below, you acknowledge:</p>
          <ul class="list-inside list-disc space-y-1">
            <li>You understand the risks involved with using user tokens</li>
            <li>
              You accept full responsibility for any consequences to your
              Discord account
            </li>
            <li>
              The authors are not responsible for any actions taken against your
              account
            </li>
          </ul>
        </div>
      </div>

      <Button
        variant="destructive"
        class="w-full"
        :disabled="loading"
        @click="handleAcknowledge"
      >
        {{ loading ? "Acknowledging..." : "I Understand and Accept the Risks" }}
      </Button>
    </DialogContent>
  </Dialog>
</template>
