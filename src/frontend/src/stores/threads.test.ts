import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useThreadsStore, type Thread } from './threads'

// Mock axios at module level
vi.mock('axios', () => ({
  default: {
    get: vi.fn(),
    patch: vi.fn(),
    post: vi.fn(),
  },
}))

import axios from 'axios'
const axiosMock = axios as { get: ReturnType<typeof vi.fn>; patch: ReturnType<typeof vi.fn>; post: ReturnType<typeof vi.fn> }

const makeThread = (overrides: Partial<Thread> = {}): Thread => ({
  id: 'thread-1',
  conversation_id: 'conv-1',
  subject: 'Test Subject',
  from: 'sender@example.com',
  received_at: '2026-01-01T12:00:00Z',
  last_message_at: '2026-01-01T12:00:00Z',
  message_count: 3,
  summary: '',
  priority: '',
  action_required: false,
  action_detail: '',
  status: 'unread',
  notes: '',
  summary_status: 'no_key',
  ...overrides,
})

describe('useThreadsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchThreads', () => {
    it('populates threads and total on success', async () => {
      const threads = [makeThread(), makeThread({ id: 'thread-2', subject: 'Another' })]
      axiosMock.get.mockResolvedValueOnce({ data: { threads, total: 2 } })

      const store = useThreadsStore()
      await store.fetchThreads()

      expect(store.threads).toHaveLength(2)
      expect(store.total).toBe(2)
      expect(store.loading).toBe(false)
    })

    it('sets loading=true during fetch, false after', async () => {
      let resolveGet!: (v: unknown) => void
      axiosMock.get.mockReturnValueOnce(new Promise((res) => { resolveGet = res }))

      const store = useThreadsStore()
      const fetchPromise = store.fetchThreads()
      expect(store.loading).toBe(true)

      resolveGet({ data: { threads: [], total: 0 } })
      await fetchPromise
      expect(store.loading).toBe(false)
    })

    it('sets loading=false even when fetch throws', async () => {
      axiosMock.get.mockRejectedValueOnce(new Error('network error'))

      const store = useThreadsStore()
      await store.fetchThreads().catch(() => {})
      expect(store.loading).toBe(false)
    })

    it('passes filter params to the API', async () => {
      axiosMock.get.mockResolvedValueOnce({ data: { threads: [], total: 0 } })

      const store = useThreadsStore()
      store.filter.status = 'unread'
      store.filter.priority = 'high'
      await store.fetchThreads()

      expect(axiosMock.get).toHaveBeenCalledWith('/api/threads', {
        params: expect.objectContaining({ status: 'unread', priority: 'high' }),
      })
    })
  })

  describe('updateThread', () => {
    it('patches the API and updates local thread', async () => {
      axiosMock.patch.mockResolvedValueOnce({})

      const store = useThreadsStore()
      store.threads.push(makeThread({ id: 'thread-1', status: 'unread' }))

      await store.updateThread('thread-1', { status: 'read' })

      expect(axiosMock.patch).toHaveBeenCalledWith('/api/threads/thread-1', { status: 'read' })
      expect(store.threads[0].status).toBe('read')
    })

    it('updates notes on local thread', async () => {
      axiosMock.patch.mockResolvedValueOnce({})

      const store = useThreadsStore()
      store.threads.push(makeThread({ id: 'thread-1', notes: '' }))

      await store.updateThread('thread-1', { notes: 'Important' })

      expect(store.threads[0].notes).toBe('Important')
    })

    it('does not mutate other threads', async () => {
      axiosMock.patch.mockResolvedValueOnce({})

      const store = useThreadsStore()
      store.threads.push(makeThread({ id: 'thread-1', status: 'unread' }))
      store.threads.push(makeThread({ id: 'thread-2', status: 'unread' }))

      await store.updateThread('thread-1', { status: 'done' })

      expect(store.threads[1].status).toBe('unread')
    })
  })

  describe('triggerResummary', () => {
    it('posts to resummary endpoint and sets local status=pending', async () => {
      axiosMock.post.mockResolvedValueOnce({})

      const store = useThreadsStore()
      store.threads.push(makeThread({ id: 'thread-1', summary_status: 'done' }))

      await store.triggerResummary('thread-1')

      expect(axiosMock.post).toHaveBeenCalledWith('/api/threads/thread-1/resummary')
      expect(store.threads[0].summary_status).toBe('pending')
    })

    it('does not change threads if id not found', async () => {
      axiosMock.post.mockResolvedValueOnce({})

      const store = useThreadsStore()
      store.threads.push(makeThread({ id: 'thread-1', summary_status: 'done' }))

      await store.triggerResummary('thread-999')

      expect(store.threads[0].summary_status).toBe('done')
    })
  })

  describe('triggerSync', () => {
    it('posts to /api/mail/sync', async () => {
      axiosMock.post.mockResolvedValueOnce({})

      const store = useThreadsStore()
      await store.triggerSync()

      expect(axiosMock.post).toHaveBeenCalledWith('/api/mail/sync')
    })
  })

  describe('filter defaults', () => {
    it('initializes with page=1 and limit=50', () => {
      const store = useThreadsStore()
      expect(store.filter.page).toBe(1)
      expect(store.filter.limit).toBe(50)
    })
  })
})
