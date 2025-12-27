<script setup lang="ts">
import { TriangleAlert } from "lucide-vue-next";
import { ref } from "vue";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

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
      :hide-close-button="true"
      @escape-key-down.prevent
      @pointer-down-outside.prevent
      @interact-outside.prevent
    >
      <DialogHeader>
        <DialogTitle class="text-destructive flex items-center gap-2">
          <TriangleAlert class="h-5 w-5" />
          Terms of Service Warning
        </DialogTitle>
        <DialogDescription> Please read and acknowledge before proceeding </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-4">
        <div class="border-destructive/50 bg-destructive/10 rounded-lg border p-4">
          <p class="text-destructive mb-3 font-semibold">IMPORTANT: READ BEFORE PROCEEDING</p>
          <p class="text-muted-foreground mb-3 text-sm">
            This tool uses Discord user tokens to maintain presence status. Using user tokens with
            automated tools
            <strong class="text-foreground">may violate Discord's Terms of Service</strong>
            and could result in:
          </p>
          <ul class="text-muted-foreground mb-3 list-inside list-disc space-y-1 text-sm">
            <li>Account suspension</li>
            <li>Account termination</li>
            <li>Loss of access to Discord services</li>
          </ul>
        </div>

        <div class="text-muted-foreground text-sm">
          <p class="mb-2">By clicking the button below, you acknowledge:</p>
          <ul class="list-inside list-disc space-y-1">
            <li>You understand the risks involved with using user tokens</li>
            <li>You accept full responsibility for any consequences to your Discord account</li>
            <li>The authors are not responsible for any actions taken against your account</li>
          </ul>
        </div>
      </div>

      <Button variant="destructive" class="w-full" :disabled="loading" @click="handleAcknowledge">
        {{ loading ? "Acknowledging..." : "I Understand and Accept the Risks" }}
      </Button>
    </DialogContent>
  </Dialog>
</template>
