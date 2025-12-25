<script setup lang="ts">
import { ref } from "vue";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/composables/useAuth";
import { KeyRound, LogIn } from "lucide-vue-next";

const { login, loading, error } = useAuth();

const apiKey = ref("");

async function handleSubmit() {
  if (!apiKey.value.trim()) return;
  await login(apiKey.value.trim());
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center p-4">
    <Card class="w-full max-w-md">
      <CardHeader class="text-center">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
          <KeyRound class="h-6 w-6 text-primary" />
        </div>
        <CardTitle class="text-xl">Discord Stay Online</CardTitle>
        <p class="text-sm text-muted-foreground">
          Enter your API key to access the dashboard
        </p>
      </CardHeader>
      <CardContent>
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <div class="space-y-2">
            <Label for="api-key">API Key</Label>
            <Input
              id="api-key"
              v-model="apiKey"
              type="password"
              placeholder="Enter your API key"
              autocomplete="current-password"
              required
            />
          </div>

          <p v-if="error" class="text-sm text-destructive">
            {{ error }}
          </p>

          <Button type="submit" class="w-full" :disabled="loading || !apiKey.trim()">
            <LogIn class="mr-2 h-4 w-4" />
            {{ loading ? "Logging in..." : "Login" }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
