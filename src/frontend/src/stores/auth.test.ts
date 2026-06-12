import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore, type User, type FilterSettings } from './auth'

vi.mock('axios', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  },
}))

import axios from 'axios'
const axiosMock = axios as { get: ReturnType<typeof vi.fn>; post: ReturnType<typeof vi.fn>; patch: ReturnType<typeof vi.fn> }

const makeUser = (overrides: Partial<User> = {}): User => ({
  id: 'user-1',
  email: 'me@example.com',
  display_name: 'Alice',
  filter_settings: null,
  needs_reauth: false,
  ...overrides,
})

const makeFilter = (overrides: Partial<FilterSettings> = {}): FilterSettings => ({
  my_email: 'me@example.com',
  my_name: 'Alice',
  my_name_aliases: [],
  pull_conditions: { to_me: true, cc_me: false, mention_email: false, mention_name: false, mention_aliases: false },
  exclude: { senders: [], subject_keywords: [] },
  ...overrides,
})

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchMe', () => {
    it('sets user on success', async () => {
      const user = makeUser()
      axiosMock.get.mockResolvedValueOnce({ data: user })

      const store = useAuthStore()
      await store.fetchMe()

      expect(store.user).toEqual(user)
      expect(store.loading).toBe(false)
    })

    it('sets loading=true during request, false after', async () => {
      let resolve!: (v: unknown) => void
      axiosMock.get.mockReturnValueOnce(new Promise((res) => { resolve = res }))

      const store = useAuthStore()
      const fetchPromise = store.fetchMe()
      expect(store.loading).toBe(true)

      resolve({ data: makeUser() })
      await fetchPromise
      expect(store.loading).toBe(false)
    })

    it('sets loading=false even when fetch fails', async () => {
      axiosMock.get.mockRejectedValueOnce(new Error('401'))

      const store = useAuthStore()
      await store.fetchMe().catch(() => {})
      expect(store.loading).toBe(false)
    })

    it('calls GET /api/users/me', async () => {
      axiosMock.get.mockResolvedValueOnce({ data: makeUser() })

      const store = useAuthStore()
      await store.fetchMe()

      expect(axiosMock.get).toHaveBeenCalledWith('/api/users/me')
    })
  })

  describe('logout', () => {
    it('calls POST /auth/logout and clears user', async () => {
      axiosMock.post.mockResolvedValueOnce({})

      const store = useAuthStore()
      store.user = makeUser()

      await store.logout()

      expect(axiosMock.post).toHaveBeenCalledWith('/auth/logout')
      expect(store.user).toBeNull()
    })
  })

  describe('updateFilter', () => {
    it('patches filter and updates local user.filter_settings', async () => {
      axiosMock.patch.mockResolvedValueOnce({})

      const store = useAuthStore()
      store.user = makeUser({ filter_settings: null })
      const filter = makeFilter()

      await store.updateFilter(filter)

      expect(axiosMock.patch).toHaveBeenCalledWith('/api/users/me/filter', filter)
      expect(store.user?.filter_settings).toEqual(filter)
    })

    it('does not crash if user is null', async () => {
      axiosMock.patch.mockResolvedValueOnce({})

      const store = useAuthStore()
      store.user = null

      await expect(store.updateFilter(makeFilter())).resolves.not.toThrow()
    })
  })

  describe('initial state', () => {
    it('starts with user=null and loading=false', () => {
      const store = useAuthStore()
      expect(store.user).toBeNull()
      expect(store.loading).toBe(false)
    })
  })
})
