<template>
  <div class="dashboard">
    <header class="top-bar">
      <h1>Mail Tracker</h1>
      <div class="actions">
        <button @click="sync" :disabled="syncing" class="btn-sync">
          {{ syncing ? 'Syncing…' : 'Sync' }}
        </button>
        <span class="user-name">{{ auth.user?.display_name }}</span>
        <a href="/settings" class="btn-settings">Settings</a>
        <button @click="handleLogout" class="btn-logout">Logout</button>
      </div>
    </header>

    <div v-if="auth.user?.needs_reauth" class="banner banner-error">
      Session expired. <a href="/auth/login">Sign in again</a>
    </div>

    <div v-if="noAIKey" class="banner banner-warn">
      No AI key configured. Summaries are disabled.
      <a href="/settings">Configure in Settings</a>
    </div>

    <!-- Filter bar -->
    <div class="filter-bar">
      <input v-model="store.filter.search" placeholder="Search…" @input="debouncedFetch" />
      <select v-model="store.filter.status" @change="store.fetchThreads()">
        <option value="">All status</option>
        <option value="unread">Unread</option>
        <option value="read">Read</option>
        <option value="done">Done</option>
        <option value="archived">Archived</option>
      </select>
      <select v-model="store.filter.priority" @change="store.fetchThreads()">
        <option value="">All priority</option>
        <option value="high">High</option>
        <option value="medium">Medium</option>
        <option value="low">Low</option>
      </select>
    </div>

    <!-- Thread list -->
    <div v-if="store.loading" class="loading">Loading…</div>
    <div v-else-if="store.threads.length === 0" class="empty">No threads found.</div>
    <table v-else class="thread-table">
      <thead>
        <tr>
          <th>Subject</th>
          <th>From</th>
          <th>Messages</th>
          <th>Received</th>
          <th v-if="hasTier2">Summary</th>
          <th v-if="hasTier2">Priority</th>
          <th v-if="hasTier2">Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="t in store.threads" :key="t.id" class="thread-row">
          <td>{{ t.subject }}</td>
          <td>{{ t.from }}</td>
          <td>{{ t.message_count }}</td>
          <td>{{ formatDate(t.received_at) }}</td>
          <td v-if="hasTier2" class="summary-cell">{{ t.summary || t.summary_status }}</td>
          <td v-if="hasTier2">
            <span :class="`badge badge-${t.priority}`">{{ t.priority }}</span>
          </td>
          <td v-if="hasTier2">
            <select :value="t.status" @change="(e) => store.updateThread(t.id, { status: (e.target as HTMLSelectElement).value })">
              <option value="unread">Unread</option>
              <option value="read">Read</option>
              <option value="done">Done</option>
              <option value="archived">Archived</option>
            </select>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useThreadsStore } from '@/stores/threads'

const auth = useAuthStore()
const store = useThreadsStore()
const router = useRouter()
const syncing = ref(false)

const hasTier2 = computed(() =>
  store.threads.some((t) => t.summary_status === 'done'),
)
const noAIKey = computed(() =>
  store.threads.length > 0 && store.threads.every((t) => t.summary_status === 'no_key'),
)

let debounceTimer: ReturnType<typeof setTimeout>
function debouncedFetch() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => store.fetchThreads(), 400)
}

async function sync() {
  syncing.value = true
  try {
    await store.triggerSync()
  } finally {
    syncing.value = false
  }
}

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString()
}

// SSE: re-fetch when sync completes
let es: EventSource | null = null

onMounted(() => {
  store.fetchThreads()
  es = new EventSource('/events', { withCredentials: true })
  es.addEventListener('sync_complete', () => store.fetchThreads())
})

onUnmounted(() => {
  es?.close()
})
</script>

<style scoped>
.dashboard { max-width: 1200px; margin: 0 auto; padding: 16px; }
.top-bar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.top-bar h1 { font-size: 20px; margin: 0; }
.actions { display: flex; gap: 12px; align-items: center; }
.banner { padding: 10px 16px; border-radius: 4px; margin-bottom: 12px; font-size: 14px; }
.banner-error { background: #fdecea; color: #d32f2f; }
.banner-warn { background: #fff8e1; color: #f57c00; }
.filter-bar { display: flex; gap: 8px; margin-bottom: 12px; }
.filter-bar input, .filter-bar select { padding: 6px 10px; border: 1px solid #ddd; border-radius: 4px; }
.thread-table { width: 100%; border-collapse: collapse; font-size: 14px; }
.thread-table th, .thread-table td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #eee; }
.thread-table th { background: #f5f5f5; font-weight: 600; }
.summary-cell { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.badge { padding: 2px 8px; border-radius: 12px; font-size: 12px; text-transform: capitalize; }
.badge-high { background: #fdecea; color: #d32f2f; }
.badge-medium { background: #fff8e1; color: #f57c00; }
.badge-low { background: #e8f5e9; color: #388e3c; }
.loading, .empty { padding: 40px; text-align: center; color: #888; }
.btn-sync, .btn-logout { padding: 6px 14px; border: 1px solid #ddd; border-radius: 4px; cursor: pointer; background: white; }
.btn-sync:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-settings { color: #0078d4; text-decoration: none; }
</style>
