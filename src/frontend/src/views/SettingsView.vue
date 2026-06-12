<template>
  <div class="settings">
    <header class="top-bar">
      <a href="/" class="back-link">← Dashboard</a>
      <h1>Settings</h1>
    </header>

    <!-- Filter Settings -->
    <section class="card">
      <h2>Mail Filter</h2>
      <div class="tab-bar">
        <button :class="{ active: tab === 'form' }" @click="tab = 'form'">Form</button>
        <button :class="{ active: tab === 'json' }" @click="tab = 'json'">JSON</button>
      </div>

      <div v-if="tab === 'form'" class="filter-form">
        <label>My email
          <input v-model="filter.my_email" type="email" />
        </label>
        <label>My name
          <input v-model="filter.my_name" />
        </label>

        <fieldset>
          <legend>Watched folders</legend>
          <p class="field-hint">Outlook folders to pull mail from. Leave empty to use Inbox only.</p>
          <div class="tag-list">
            <span v-for="(folder, i) in filter.watched_folders" :key="i" class="tag">
              {{ folder }}
              <button type="button" class="tag-remove" @click="removeFolder(i)">×</button>
            </span>
          </div>
          <div class="tag-input-row">
            <input
              v-model="newFolder"
              placeholder="e.g. Projects/Alpha"
              @keydown.enter.prevent="addFolder"
            />
            <button type="button" class="btn-secondary" @click="addFolder">Add</button>
          </div>
        </fieldset>

        <fieldset>
          <legend>Pull when</legend>
          <label><input v-model="filter.pull_conditions.to_me" type="checkbox" /> To me</label>
          <label><input v-model="filter.pull_conditions.cc_me" type="checkbox" /> CC me</label>
          <label><input v-model="filter.pull_conditions.mention_email" type="checkbox" /> Mentions my email</label>
          <label><input v-model="filter.pull_conditions.mention_name" type="checkbox" /> Mentions my name</label>
        </fieldset>
      </div>

      <div v-else class="json-editor">
        <textarea v-model="filterJson" rows="14" @input="syncJsonToForm" />
        <p v-if="jsonError" class="error-msg">{{ jsonError }}</p>
      </div>

      <button @click="saveFilter" :disabled="saving" class="btn-primary">
        {{ saving ? 'Saving…' : 'Save Filter' }}
      </button>
    </section>

    <!-- AI Config -->
    <section class="card">
      <h2>AI Summary</h2>
      <div class="ai-mode">
        <label><input v-model="aiMode" type="radio" value="byok" /> BYOK — Use my API key</label>
        <label><input v-model="aiMode" type="radio" value="power_automate" /> Copilot via Power Automate (free, M365)</label>
      </div>

      <div v-if="aiMode === 'byok'" class="ai-form">
        <label>Provider
          <select v-model="aiProvider">
            <option value="openai">OpenAI</option>
            <option value="openrouter">OpenRouter</option>
          </select>
        </label>
        <label>API Key
          <input v-model="aiKey" type="password" placeholder="sk-…" />
        </label>
        <label>Model
          <input v-model="aiModel" placeholder="gpt-4o-mini" />
        </label>
      </div>

      <div v-if="aiMode === 'power_automate'" class="ai-form">
        <p class="warn-msg">⚠️ Copilot via Power Automate is experimental (POC). Output may be less stable than BYOK.</p>
        <label>Power Automate Webhook URL
          <input v-model="paWebhook" type="url" placeholder="https://prod-xx.westus.logic.azure.com/…" />
        </label>
      </div>

      <div class="btn-row">
        <button @click="testAI" :disabled="testing" class="btn-secondary">{{ testing ? 'Testing…' : 'Test' }}</button>
        <button @click="saveAI" :disabled="saving" class="btn-primary">{{ saving ? 'Saving…' : 'Save' }}</button>
        <button @click="deleteAI" class="btn-danger">Remove</button>
      </div>
      <p v-if="aiResult" :class="aiResult.ok ? 'success-msg' : 'error-msg'">{{ aiResult.msg }}</p>
    </section>

    <!-- Logout -->
    <section class="card">
      <button @click="handleLogout" class="btn-danger">Logout</button>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { useAuthStore, type FilterSettings } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const tab = ref<'form' | 'json'>('form')
const saving = ref(false)
const testing = ref(false)
const jsonError = ref('')
const aiResult = ref<{ ok: boolean; msg: string } | null>(null)
const newFolder = ref('')

function addFolder() {
  const name = newFolder.value.trim()
  if (!name) return
  if (!filter.value.watched_folders.includes(name)) {
    filter.value.watched_folders.push(name)
  }
  newFolder.value = ''
}

function removeFolder(index: number) {
  filter.value.watched_folders.splice(index, 1)
}

const defaultFilter: FilterSettings = {
  my_email: '',
  my_name: '',
  my_name_aliases: [],
  watched_folders: [],
  pull_conditions: { to_me: true, cc_me: true, mention_email: true, mention_name: true, mention_aliases: true },
  exclude: { senders: [], subject_keywords: [] },
}

const filter = ref<FilterSettings>(
  auth.user?.filter_settings ? { ...defaultFilter, ...auth.user.filter_settings } : { ...defaultFilter },
)
const filterJson = ref(JSON.stringify(filter.value, null, 2))

const aiMode = ref('byok')
const aiProvider = ref('openai')
const aiKey = ref('')
const aiModel = ref('gpt-4o-mini')
const paWebhook = ref('')

watch(filter, (v) => { filterJson.value = JSON.stringify(v, null, 2) }, { deep: true })

function syncJsonToForm() {
  try {
    filter.value = JSON.parse(filterJson.value)
    jsonError.value = ''
  } catch {
    jsonError.value = 'Invalid JSON'
  }
}

async function saveFilter() {
  saving.value = true
  try {
    await auth.updateFilter(filter.value)
  } finally {
    saving.value = false
  }
}

async function testAI() {
  testing.value = true
  aiResult.value = null
  try {
    const body = aiMode.value === 'byok'
      ? { ai_mode: 'byok', provider: aiProvider.value, api_key: aiKey.value, model: aiModel.value }
      : { ai_mode: 'power_automate', pa_webhook_url: paWebhook.value }
    await axios.post('/api/users/me/api-key/test', body)
    aiResult.value = { ok: true, msg: 'Connection successful!' }
  } catch (e: any) {
    aiResult.value = { ok: false, msg: e.response?.data?.error ?? 'Test failed' }
  } finally {
    testing.value = false
  }
}

async function saveAI() {
  saving.value = true
  aiResult.value = null
  try {
    const body = aiMode.value === 'byok'
      ? { ai_mode: 'byok', provider: aiProvider.value, api_key: aiKey.value, model: aiModel.value }
      : { ai_mode: 'power_automate', pa_webhook_url: paWebhook.value }
    await axios.post('/api/users/me/api-key', body)
    aiResult.value = { ok: true, msg: 'Saved.' }
  } catch (e: any) {
    aiResult.value = { ok: false, msg: e.response?.data?.error ?? 'Save failed' }
  } finally {
    saving.value = false
  }
}

async function deleteAI() {
  await axios.delete('/api/users/me/api-key')
  aiResult.value = { ok: true, msg: 'AI config removed.' }
}

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.settings { max-width: 720px; margin: 0 auto; padding: 16px; }
.top-bar { display: flex; align-items: center; gap: 16px; margin-bottom: 20px; }
.top-bar h1 { margin: 0; font-size: 20px; }
.back-link { color: #0078d4; text-decoration: none; }
.card { background: white; border: 1px solid #e0e0e0; border-radius: 8px; padding: 24px; margin-bottom: 16px; }
.card h2 { margin: 0 0 16px; font-size: 18px; }
.tab-bar { display: flex; gap: 0; margin-bottom: 16px; border: 1px solid #ddd; border-radius: 4px; overflow: hidden; width: fit-content; }
.tab-bar button { padding: 6px 16px; border: none; background: white; cursor: pointer; }
.tab-bar button.active { background: #0078d4; color: white; }
label { display: flex; flex-direction: column; gap: 4px; margin-bottom: 12px; font-size: 14px; font-weight: 500; }
label input, label select { padding: 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
fieldset { border: 1px solid #eee; border-radius: 4px; padding: 12px; }
fieldset label { flex-direction: row; align-items: center; gap: 8px; font-weight: normal; margin: 0; }
textarea { width: 100%; box-sizing: border-box; font-family: monospace; font-size: 13px; border: 1px solid #ddd; border-radius: 4px; padding: 8px; }
.ai-mode { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.ai-mode label { flex-direction: row; align-items: center; gap: 8px; font-weight: normal; margin: 0; }
.ai-form { border-top: 1px solid #eee; padding-top: 16px; margin-bottom: 16px; }
.btn-row { display: flex; gap: 8px; }
.btn-primary, .btn-secondary, .btn-danger { padding: 8px 16px; border-radius: 4px; border: none; cursor: pointer; font-size: 14px; }
.btn-primary { background: #0078d4; color: white; }
.btn-secondary { background: white; border: 1px solid #ddd; }
.btn-danger { background: white; border: 1px solid #d32f2f; color: #d32f2f; }
.error-msg { color: #d32f2f; font-size: 13px; }
.warn-msg { color: #f57c00; font-size: 13px; }
.success-msg { color: #388e3c; font-size: 13px; }
.field-hint { margin: 0 0 10px; font-size: 13px; color: #666; font-weight: normal; }
.tag-list { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 10px; min-height: 24px; }
.tag { display: inline-flex; align-items: center; gap: 4px; background: #e3f2fd; border: 1px solid #90caf9; border-radius: 4px; padding: 2px 8px; font-size: 13px; }
.tag-remove { background: none; border: none; cursor: pointer; color: #555; font-size: 15px; line-height: 1; padding: 0; }
.tag-remove:hover { color: #d32f2f; }
.tag-input-row { display: flex; gap: 8px; }
.tag-input-row input { flex: 1; padding: 8px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
</style>
