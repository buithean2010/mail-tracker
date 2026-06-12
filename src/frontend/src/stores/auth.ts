import { ref } from 'vue'
import { defineStore } from 'pinia'
import axios from 'axios'

export interface FilterSettings {
  my_email: string
  my_name: string
  my_name_aliases: string[]
  watched_folders: string[]
  pull_conditions: {
    to_me: boolean
    cc_me: boolean
    mention_email: boolean
    mention_name: boolean
    mention_aliases: boolean
  }
  exclude: {
    senders: string[]
    subject_keywords: string[]
  }
}

export interface User {
  id: string
  email: string
  display_name: string
  filter_settings: FilterSettings | null
  needs_reauth: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)

  async function fetchMe() {
    loading.value = true
    try {
      const res = await axios.get<User>('/api/users/me')
      user.value = res.data
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    await axios.post('/auth/logout')
    user.value = null
  }

  async function updateFilter(settings: FilterSettings) {
    await axios.patch('/api/users/me/filter', settings)
    if (user.value) {
      user.value.filter_settings = settings
    }
  }

  return { user, loading, fetchMe, logout, updateFilter }
})
