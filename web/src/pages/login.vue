<script setup lang="ts">
import { KeyRound, LogIn } from "lucide-vue-next";
import { ref } from "vue";
import { useRouter } from "vue-router";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/composables/useAuth";

const router = useRouter();
const { error, loading, login } = useAuth();

const apiKey = ref("");

async function handleSubmit() {
  if (!apiKey.value.trim()) return;

  const success = await login(apiKey.value.trim());
  if (success) {
    router.push("/");
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center p-4">
    <Card class="fade-in border-border/50 bg-card/50 w-full max-w-md backdrop-blur-sm">
      <CardHeader class="space-y-4 text-center">
        <div class="bg-foreground mx-auto flex h-12 w-12 items-center justify-center rounded-lg">
          <KeyRound class="text-background h-5 w-5" />
        </div>
        <div class="space-y-1">
          <CardTitle class="text-lg font-semibold tracking-tight">Discord Stay Online</CardTitle>
          <p class="text-muted-foreground text-sm">Enter your API key to continue</p>
        </div>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="handleSubmit">
          <div class="space-y-2">
            <Label for="api-key" class="text-sm font-medium">API Key</Label>
            <Input
              id="api-key"
              v-model="apiKey"
              type="password"
              placeholder="sk-..."
              autocomplete="current-password"
              class="bg-background/50 focus:bg-background transition-colors"
              required
            />
          </div>

          <p v-if="error" class="text-destructive text-sm">
            {{ error }}
          </p>

          <Button type="submit" class="press-effect w-full" :disabled="loading || !apiKey.trim()">
            <LogIn />
            {{ loading ? "Authenticating..." : "Continue" }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
