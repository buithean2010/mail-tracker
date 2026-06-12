<template>
  <div class="login-page">
    <div class="login-card">
      <h1>Mail Tracker</h1>
      <p v-if="reauthRequired" class="error-msg">
        Session expired. Please sign in again.
      </p>
      <a href="/auth/login" class="btn-microsoft">
        Sign in with Microsoft
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const reauthRequired = computed(() => route.query.reauth === '1')

onMounted(async () => {
  if (reauthRequired.value) return
  try {
    await auth.fetchMe()
    if (auth.user) router.replace('/')
  } catch {
    // not authenticated — stay on login page
  }
})
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f5f5f5;
}

.login-card {
  background: white;
  border-radius: 8px;
  padding: 40px;
  text-align: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  max-width: 360px;
  width: 100%;
}

h1 {
  margin-bottom: 24px;
  font-size: 24px;
  color: #1a1a1a;
}

.error-msg {
  color: #d32f2f;
  margin-bottom: 16px;
  font-size: 14px;
}

.btn-microsoft {
  display: inline-block;
  background: #0078d4;
  color: white;
  padding: 12px 24px;
  border-radius: 4px;
  text-decoration: none;
  font-size: 15px;
  transition: background 0.2s;
}

.btn-microsoft:hover {
  background: #006cbf;
}
</style>
