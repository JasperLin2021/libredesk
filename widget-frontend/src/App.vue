<script setup>
import { ref, watch, onMounted } from 'vue'
import { MessageCircle, X, Loader2, Server, Key } from 'lucide-vue-next'
import { useWidget } from './composables/useWidget.js'

const STORAGE_KEY = 'libredesk_service_config'

const {
  isReady,
  isVisible,
  isAuthenticated,
  checking,
  error,
  config,
  initWidget,
  show,
  hide,
  verifyBixiaoAuth,
  destroy,
} = useWidget()

// ----- States -----
// 'config' → need baseURL + inboxID
// 'loading' → widget script loading
// 'ready' → widget ready, waiting for auth
// 'error' → widget failed to load
const appState = ref('config')

// ----- Config form -----
const inputBaseURL = ref('')
const inputInboxID = ref('')
const configError = ref('')

// ----- Bixiao auth modal -----
const showBixiaoModal = ref(false)
const bixiaoToken = ref('')
const bixiaoDevice = ref('')
const bixiaoVersionSeq = ref('')
const bixiaoInbox = ref('')
const bixiaoError = ref('')

// ----- Helpers -----
function loadStoredConfig() {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) return null
  try {
    const p = JSON.parse(raw)
    if (p.baseURL && p.inboxID) return p
  } catch { /* ignore */ }
  return null
}

function saveConfig() {
  const baseURL = inputBaseURL.value.trim().replace(/\/+$/, '')
  const inboxID = inputInboxID.value.trim()

  if (!baseURL) { configError.value = '请输入后端地址'; return }
  if (!inboxID) { configError.value = '请输入 Inbox ID'; return }
  configError.value = ''

  localStorage.setItem(STORAGE_KEY, JSON.stringify({ baseURL, inboxID }))
  initWidget(baseURL, inboxID)
  appState.value = 'loading'
}

// ----- Service Button -----
function handleServiceClick() {
  if (appState.value !== 'ready') return
  if (isAuthenticated.value) {
    isVisible.value ? hide() : show()
  } else {
    bixiaoToken.value = 'ut:9e116f4a-8236-450d-a776-a17500dbc3e9'
    bixiaoDevice.value = 'Android'
    bixiaoVersionSeq.value = '202607015'
    bixiaoInbox.value = config.inboxID
    bixiaoError.value = ''
    showBixiaoModal.value = true
  }
}

// ----- Bixiao Verification -----
async function handleBixiaoVerify() {
  const token = bixiaoToken.value.trim()
  const device = bixiaoDevice.value.trim()
  const versionSeq = bixiaoVersionSeq.value.trim()
  const inbox = bixiaoInbox.value.trim()

  if (!token) { bixiaoError.value = '请输入 Token'; return }
  if (!device) { bixiaoError.value = '请输入 Device'; return }
  if (!versionSeq) { bixiaoError.value = '请输入 VersionSeq'; return }
  if (!inbox) { bixiaoError.value = '请输入 Inbox ID'; return }

  try {
    await verifyBixiaoAuth({ token, device, versionSeq, inbox })
    showBixiaoModal.value = false
    // Small delay to let the widget process setUser before showing
    setTimeout(() => show(), 400)
  } catch (err) {
    bixiaoError.value = err.message || '非法用户'
  }
}

// ----- Reset -----
function handleReset() {
  destroy()
  localStorage.removeItem(STORAGE_KEY)
  appState.value = 'config'
}

// ----- Lifecycle -----
watch(isReady, (val) => { if (val) appState.value = 'ready' })
watch(error, (val) => { if (val) appState.value = 'error' })

onMounted(() => {
  const stored = loadStoredConfig()
  if (stored) {
    initWidget(stored.baseURL, stored.inboxID)
    appState.value = 'loading'
  } else {
    appState.value = 'config'
  }
})
</script>

<template>
  <div class="min-h-screen bg-white flex flex-col">

    <!-- ======== Config Panel ======== -->
    <div v-if="appState === 'config'"
         class="flex-1 flex items-center justify-center px-4">
      <div class="w-full max-w-md">
        <div class="text-center mb-8">
          <div class="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-indigo-50 mb-4">
            <MessageCircle class="w-7 h-7 text-indigo-600" />
          </div>
          <h1 class="text-xl font-bold text-slate-900">在线客服配置</h1>
          <p class="mt-1 text-sm text-slate-500">请输入 LibreDesk 后端信息以开始使用</p>
        </div>

        <form @submit.prevent="saveConfig"
              class="space-y-4 bg-slate-50 rounded-2xl p-6 border border-slate-200">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5 flex items-center gap-1.5">
              <Server class="w-4 h-4" /> 后端地址
            </label>
            <input
              v-model="inputBaseURL"
              type="text"
              placeholder="https://your-server.com"
              class="w-full px-4 py-2.5 rounded-xl border border-slate-300 bg-white text-sm
                     placeholder:text-slate-400 focus:outline-none focus:ring-2
                     focus:ring-indigo-500/30 focus:border-indigo-500 transition"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5 flex items-center gap-1.5">
              <Key class="w-4 h-4" /> Inbox ID
            </label>
            <input
              v-model="inputInboxID"
              type="text"
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              class="w-full px-4 py-2.5 rounded-xl border border-slate-300 bg-white text-sm
                     placeholder:text-slate-400 focus:outline-none focus:ring-2
                     focus:ring-indigo-500/30 focus:border-indigo-500 transition font-mono"
            />
          </div>

          <p v-if="configError" class="text-sm text-red-500">{{ configError }}</p>

          <button type="submit"
                  class="w-full py-2.5 rounded-xl bg-indigo-600 text-white text-sm font-semibold
                         hover:bg-indigo-700 active:scale-[0.98] transition-all">
            保存并连接
          </button>
        </form>
      </div>
    </div>

    <!-- ======== Main Page (loading / ready / error) ======== -->
    <template v-else>
      <!-- Empty space -->
      <div class="flex-1 flex items-center justify-center">
        <!-- Loading -->
        <div v-if="appState === 'loading'" class="text-center">
          <Loader2 class="w-8 h-8 text-indigo-500 animate-spin mx-auto" />
          <p class="mt-4 text-sm text-slate-500">正在加载客服组件...</p>
        </div>

        <!-- Error -->
        <div v-if="appState === 'error'" class="text-center px-6">
          <div class="w-14 h-14 rounded-2xl bg-red-50 flex items-center justify-center mx-auto mb-4">
            <X class="w-7 h-7 text-red-500" />
          </div>
          <p class="text-slate-700 font-medium">加载失败</p>
          <p class="mt-1 text-sm text-slate-500 max-w-sm">{{ error }}</p>
          <button @click="handleReset"
                  class="mt-5 px-5 py-2 rounded-xl bg-slate-100 text-sm text-slate-700
                         hover:bg-slate-200 transition">
            重新配置
          </button>
        </div>

        <!-- Ready: blank page, just the button below -->
      </div>

      <!-- ======== "我的客服" Floating Button ======== -->
      <div class="fixed bottom-8 left-1/2 -translate-x-1/2 z-40">
        <button
          @click="handleServiceClick"
          :disabled="appState !== 'ready'"
          class="flex items-center gap-2.5 px-6 py-3.5 rounded-2xl shadow-lg shadow-indigo-500/25
                 bg-indigo-600 text-white font-semibold text-base
                 hover:bg-indigo-700 active:scale-[0.97] transition-all duration-200
                 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <MessageCircle class="w-5 h-5" />
          <span>{{ isAuthenticated ? '继续咨询' : '我的客服' }}</span>
        </button>
      </div>
    </template>

    <!-- ======== Bixiao Auth Modal ======== -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showBixiaoModal"
             class="fixed inset-0 z-50 flex items-center justify-center px-4"
             @click.self="showBixiaoModal = false">
          <!-- Backdrop -->
          <div class="absolute inset-0 bg-black/40 backdrop-blur-sm"></div>

          <!-- Modal -->
          <div class="relative w-full max-w-sm bg-white rounded-2xl shadow-2xl p-6">
            <button @click="showBixiaoModal = false"
                    class="absolute top-4 right-4 w-8 h-8 rounded-lg flex items-center justify-center
                           text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition">
              <X class="w-4 h-4" />
            </button>

            <div class="text-center mb-5">
              <div class="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-indigo-50 mb-3">
                <MessageCircle class="w-6 h-6 text-indigo-600" />
              </div>
              <h2 class="text-lg font-bold text-slate-900">身份验证</h2>
              <p class="mt-1 text-sm text-slate-500">请输入认证信息以开始咨询</p>
            </div>

            <form @submit.prevent="handleBixiaoVerify" class="space-y-3">
              <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Token</label>
                <input
                  v-model="bixiaoToken"
                  type="text"
                  placeholder="请输入 Token"
                  class="w-full px-4 py-2.5 rounded-xl border border-slate-300 bg-white text-sm
                         placeholder:text-slate-400 focus:outline-none focus:ring-2
                         focus:ring-indigo-500/30 focus:border-indigo-500 transition"
                />
              </div>

              <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Device</label>
                <input
                  v-model="bixiaoDevice"
                  type="text"
                  placeholder="请输入 Device"
                  class="w-full px-4 py-2.5 rounded-xl border border-slate-300 bg-white text-sm
                         placeholder:text-slate-400 focus:outline-none focus:ring-2
                         focus:ring-indigo-500/30 focus:border-indigo-500 transition"
                />
              </div>

              <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">VersionSeq</label>
                <input
                  v-model="bixiaoVersionSeq"
                  type="text"
                  placeholder="请输入 VersionSeq"
                  class="w-full px-4 py-2.5 rounded-xl border border-slate-300 bg-white text-sm
                         placeholder:text-slate-400 focus:outline-none focus:ring-2
                         focus:ring-indigo-500/30 focus:border-indigo-500 transition"
                />
              </div>

              <div>
                <label class="block text-sm font-medium text-slate-700 mb-1">Inbox ID</label>
                <input
                  v-model="bixiaoInbox"
                  type="text"
                  placeholder="Inbox ID（预填）"
                  class="w-full px-4 py-2.5 rounded-xl border border-slate-300 bg-slate-50 text-sm
                         placeholder:text-slate-400 focus:outline-none focus:ring-2
                         focus:ring-indigo-500/30 focus:border-indigo-500 transition font-mono"
                />
              </div>

              <p v-if="bixiaoError" class="text-sm text-red-500 flex items-center gap-1">
                <X class="w-4 h-4 flex-shrink-0" /> {{ bixiaoError }}
              </p>

              <button type="submit"
                      :disabled="checking"
                      class="w-full py-2.5 rounded-xl bg-indigo-600 text-white text-sm font-semibold
                             hover:bg-indigo-700 active:scale-[0.98] transition-all
                             disabled:opacity-60 disabled:cursor-not-allowed flex items-center justify-center gap-2">
                <Loader2 v-if="checking" class="w-4 h-4 animate-spin" />
                <span>{{ checking ? '验证中...' : '验证并开始咨询' }}</span>
              </button>
            </form>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.modal-enter-active { transition: all 0.25s ease-out; }
.modal-leave-active { transition: all 0.15s ease-in; }
.modal-enter-from,
.modal-leave-to { opacity: 0; }
.modal-enter-from .relative { transform: scale(0.95) translateY(10px); }
.modal-leave-to .relative { transform: scale(0.95) translateY(10px); }
</style>
