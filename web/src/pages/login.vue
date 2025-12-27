<script setup lang="ts">
import { Eye, EyeOff, Loader2 } from "lucide-vue-next";
import { ref } from "vue";
import { useRouter } from "vue-router";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/composables/useAuth";

const router = useRouter();
const { error, loading, login } = useAuth();

const apiKey = ref("");
const showPassword = ref(false);

async function handleSubmit() {
  if (!apiKey.value.trim()) return;

  const success = await login(apiKey.value.trim());
  if (success) {
    router.push("/");
  }
}
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center p-4">
    <!-- Logo & Title -->
    <div class="mb-8 flex flex-col items-center">
      <img src="/android-chrome-512x512.png" alt="Discord Stay Online" class="mb-4 size-16" />
      <h1 class="text-2xl font-bold tracking-tight">Discord Stay Online</h1>
      <p class="text-muted-foreground mt-1 text-sm">Enter your API key to continue</p>
    </div>

    <!-- Login Form -->
    <div class="w-full max-w-sm">
      <form class="space-y-4" @submit.prevent="handleSubmit">
        <div class="space-y-2">
          <Label for="api-key">API Key</Label>
          <div class="relative">
            <Input
              id="api-key"
              v-model="apiKey"
              :type="showPassword ? 'text' : 'password'"
              placeholder="Enter your API key"
              class="pr-10"
              required
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="absolute top-0 right-0 h-full px-3 hover:bg-transparent"
              @click="showPassword = !showPassword"
            >
              <Eye v-if="!showPassword" class="text-muted-foreground size-4" />
              <EyeOff v-else class="text-muted-foreground size-4" />
            </Button>
          </div>
        </div>

        <p v-if="error" class="text-destructive text-sm">
          {{ error }}
        </p>

        <Button type="submit" class="w-full" :disabled="loading || !apiKey.trim()">
          <Loader2 v-if="loading" class="animate-spin" />
          {{ loading ? "Authenticating..." : "Continue" }}
        </Button>
      </form>

      <p class="text-muted-foreground mt-6 text-center text-xs">
        Set <code class="bg-muted rounded px-1 py-0.5">API_KEY</code> environment variable to enable
        authentication
      </p>
    </div>
  </div>
</template>
