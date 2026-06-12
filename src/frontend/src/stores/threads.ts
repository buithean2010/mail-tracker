import { ref } from 'vue'
import { defineStore } from 'pinia'
import axios from 'axios'

export interface Thread {
  id: string
  conversation_id: string
  subject: string
  from: string
  received_at: string
  last_message_at: string
  message_count: number
  // Tier 2 fields (present when summary_status = done)
  summary: string
  priority: string // high | medium | low
  action_required: boolean
  action_detail: string
  status: string // unread | read | done | archived
  notes: string
  summary_status: string // pending | processing | done | failed | no_key
}

export interface ThreadFilter {
  status?: string
  priority?: string
  from?: string
  search?: string
  date_from?: string
  date_to?: string
  page?: number
  limit?: number
}

export const useThreadsStore = defineStore('threads', () => {
  const threads = ref<Thread[]>([])
  const total = ref(0)
  const loading = ref(false)
  const filter = ref<ThreadFilter>({ page: 1, limit: 50 })

  async function fetchThreads() {
    loading.value = true
    try {
      const res = await axios.get<{ threads: Thread[]; total: number }>('/api/threads', {
        params: filter.value,
      })
      threads.value = res.data.threads
      total.value = res.data.total
    } finally {
      loading.value = false
    }
  }

  async function updateThread(id: string, data: Partial<Pick<Thread, 'status' | 'notes'>>) {
    await axios.patch(`/api/threads/${id}`, data)
    const t = threads.value.find((t) => t.id === id)
    if (t) Object.assign(t, data)
  }

  async function triggerResummary(id: string) {
    await axios.post(`/api/threads/${id}/resummary`)
    const t = threads.value.find((t) => t.id === id)
    if (t) t.summary_status = 'pending'
  }

  async function triggerSync() {
    await axios.post('/api/mail/sync')
  }

  return { threads, total, loading, filter, fetchThreads, updateThread, triggerResummary, triggerSync }
})
