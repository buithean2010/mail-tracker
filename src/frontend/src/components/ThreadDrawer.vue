<template>
  <Teleport to="body">
    <div v-if="thread" class="drawer-overlay" @click.self="$emit('close')">
      <div class="drawer" role="dialog" aria-modal="true">
        <div class="drawer-header">
          <h2>{{ thread.subject }}</h2>
          <button class="btn-close" @click="$emit('close')" aria-label="Close">✕</button>
        </div>

        <div class="drawer-meta">
          <span><strong>From:</strong> {{ thread.from }}</span>
          <span><strong>Received:</strong> {{ formatDate(thread.received_at) }}</span>
          <span><strong>Messages:</strong> {{ thread.message_count }}</span>
          <a href="https://outlook.office.com/mail/" target="_blank" rel="noopener" class="outlook-link">
            Open in Outlook ↗
          </a>
        </div>

        <!-- Tier 2 — AI summary -->
        <div v-if="thread.summary_status === 'done'" class="drawer-summary">
          <div class="summary-badges">
            <span v-if="thread.priority" :class="`badge badge-${thread.priority}`">{{ thread.priority }}</span>
            <span v-if="thread.action_required" class="badge badge-action">Action required</span>
          </div>
          <p class="summary-text">{{ thread.summary }}</p>
          <p v-if="thread.action_detail" class="action-detail">{{ thread.action_detail }}</p>
        </div>
        <div v-else class="summary-status-msg">
          <span v-if="thread.summary_status === 'pending'">Summary pending…</span>
          <span v-else-if="thread.summary_status === 'processing'">Summarizing…</span>
          <span v-else-if="thread.summary_status === 'failed'" class="status-failed">Summary failed.</span>
          <span v-else-if="thread.summary_status === 'no_key'" class="status-nokey">
            No AI key configured. <a href="/settings">Configure in Settings</a>
          </span>
        </div>

        <!-- Notes -->
        <div class="drawer-section">
          <label>Notes
            <textarea v-model="notes" rows="4" placeholder="Add notes…" @blur="saveNotes" />
          </label>
        </div>

        <!-- Actions -->
        <div class="drawer-actions">
          <button @click="resummary" :disabled="resuming" class="btn-secondary">
            {{ resuming ? 'Requesting…' : 'Re-summarize' }}
          </button>
          <select :value="thread.status" @change="updateStatus" class="status-select">
            <option value="unread">Unread</option>
            <option value="read">Read</option>
            <option value="done">Done</option>
            <option value="archived">Archived</option>
          </select>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useThreadsStore, type Thread } from '@/stores/threads'

const props = defineProps<{ thread: Thread | null }>()
defineEmits<{ close: [] }>()

const store = useThreadsStore()
const notes = ref('')
const resuming = ref(false)

watch(() => props.thread, (t) => {
  notes.value = t?.notes ?? ''
})

function formatDate(iso: string) {
  return new Date(iso).toLocaleString()
}

async function saveNotes() {
  if (!props.thread || notes.value === props.thread.notes) return
  await store.updateThread(props.thread.id, { notes: notes.value })
}

async function resummary() {
  if (!props.thread) return
  resuming.value = true
  try {
    await store.triggerResummary(props.thread.id)
  } finally {
    resuming.value = false
  }
}

async function updateStatus(e: Event) {
  if (!props.thread) return
  await store.updateThread(props.thread.id, { status: (e.target as HTMLSelectElement).value })
}
</script>

<style scoped>
.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 100;
  display: flex;
  justify-content: flex-end;
}
.drawer {
  background: white;
  width: 480px;
  max-width: 100%;
  height: 100%;
  overflow-y: auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  box-shadow: -2px 0 12px rgba(0, 0, 0, 0.15);
}
.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}
.drawer-header h2 {
  margin: 0;
  font-size: 17px;
  line-height: 1.4;
  flex: 1;
  word-break: break-word;
}
.btn-close {
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: #666;
  padding: 0;
  flex-shrink: 0;
}
.drawer-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  color: #555;
  padding-bottom: 12px;
  border-bottom: 1px solid #eee;
}
.outlook-link {
  color: #0078d4;
  text-decoration: none;
  margin-top: 4px;
  font-size: 13px;
}
.drawer-summary {
  background: #f8f8f8;
  border-radius: 6px;
  padding: 12px;
}
.summary-badges {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}
.badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  text-transform: capitalize;
}
.badge-high { background: #fdecea; color: #d32f2f; }
.badge-medium { background: #fff8e1; color: #f57c00; }
.badge-low { background: #e8f5e9; color: #388e3c; }
.badge-action { background: #e3f2fd; color: #1565c0; }
.summary-text { margin: 0 0 8px; font-size: 14px; line-height: 1.6; }
.action-detail { margin: 0; font-size: 13px; color: #555; font-style: italic; }
.summary-status-msg { color: #888; font-size: 13px; }
.status-failed { color: #d32f2f; }
.status-nokey a { color: #0078d4; }
.drawer-section label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
}
.drawer-section textarea {
  width: 100%;
  box-sizing: border-box;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
}
.drawer-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.btn-secondary {
  padding: 7px 14px;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: white;
  cursor: pointer;
  font-size: 14px;
}
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
.status-select {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}
</style>
