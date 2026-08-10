<script setup>
import { ref, reactive } from 'vue'
import { Settings, Globe, Key, Zap, ArrowRight } from 'lucide-vue-next'

const props = defineProps({
  defaultBaseURL: { type: String, default: 'http://localhost:9000' },
  defaultInboxID: { type: String, default: '' },
})

const emit = defineEmits(['saved'])

const form = reactive({
  baseURL: props.defaultBaseURL || 'http://localhost:9000',
  inboxID: props.defaultInboxID || '',
})

const errors = reactive({
  baseURL: '',
  inboxID: '',
})

function validateURL(url) {
  if (!url) return '请输入后端地址'
  const pattern = /^https?:\/\/.+/
  if (!pattern.test(url)) return '请输入有效的 URL（以 http:// 或 https:// 开头）'
  return ''
}

function validateInboxID(id) {
  if (!id) return '请输入 Inbox ID'
  if (!/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i.test(id)) {
    return '请输入有效的 UUID 格式（例如: 550e8400-e29b-41d4-a716-446655440000）'
  }
  return ''
}

function handleSubmit() {
  errors.baseURL = validateURL(form.baseURL.trim())
  errors.inboxID = validateInboxID(form.inboxID.trim())

  if (errors.baseURL || errors.inboxID) return

  emit('saved', {
    baseURL: form.baseURL.replace(/\/+$/, ''),
    inboxID: form.inboxID.trim(),
  })
}
</script>

<template>
  <div class="fixed inset-0 z-[100] flex items-center justify-center p-4">
    <!-- Backdrop -->
    <div class="absolute inset-0 bg-slate-900/60 backdrop-blur-sm"></div>

    <!-- Panel -->
    <div class="relative w-full max-w-lg glass-dark p-8 animate-[fadeIn_0.4s_ease-out]">
      <!-- Header -->
      <div class="flex items-center gap-3 mb-8">
        <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-primary to-accent flex items-center justify-center">
          <Settings class="w-5 h-5 text-white" />
        </div>
        <div>
          <h2 class="text-xl font-bold text-white">连接 LibreDesk</h2>
          <p class="text-sm text-slate-400 mt-0.5">输入后端地址和 Inbox ID 以启动在线客服</p>
        </div>
      </div>

      <!-- Form -->
      <form @submit.prevent="handleSubmit" class="space-y-6">
        <!-- Base URL -->
        <div>
          <label class="flex items-center gap-2 text-sm font-medium text-slate-300 mb-2">
            <Globe class="w-4 h-4" />
            后端地址 (baseURL)
          </label>
          <input
            v-model="form.baseURL"
            type="text"
            placeholder="http://localhost:9000"
            class="w-full px-4 py-3 bg-slate-800/60 border border-slate-700/60 rounded-xl text-white placeholder-slate-500
                   focus:outline-none focus:border-primary-light focus:ring-2 focus:ring-primary/20 transition-all duration-200"
            :class="{ 'border-red-500/60 focus:border-red-400 focus:ring-red-500/20': errors.baseURL }"
          />
          <p v-if="errors.baseURL" class="text-red-400 text-xs mt-1.5 flex items-center gap-1">
            <span class="w-1 h-1 rounded-full bg-red-400"></span>
            {{ errors.baseURL }}
          </p>
        </div>

        <!-- Inbox ID -->
        <div>
          <label class="flex items-center gap-2 text-sm font-medium text-slate-300 mb-2">
            <Key class="w-4 h-4" />
            Inbox ID
          </label>
          <input
            v-model="form.inboxID"
            type="text"
            placeholder="550e8400-e29b-41d4-a716-446655440000"
            class="w-full px-4 py-3 bg-slate-800/60 border border-slate-700/60 rounded-xl text-white placeholder-slate-500
                   focus:outline-none focus:border-primary-light focus:ring-2 focus:ring-primary/20 transition-all duration-200 font-mono text-sm"
            :class="{ 'border-red-500/60 focus:border-red-400 focus:ring-red-500/20': errors.inboxID }"
          />
          <p v-if="errors.inboxID" class="text-red-400 text-xs mt-1.5 flex items-center gap-1">
            <span class="w-1 h-1 rounded-full bg-red-400"></span>
            {{ errors.inboxID }}
          </p>
        </div>

        <!-- Submit -->
        <button
          type="submit"
          class="w-full flex items-center justify-center gap-2 px-6 py-3.5 bg-gradient-to-r from-primary to-accent
                 text-white font-semibold rounded-xl hover:from-primary-dark hover:to-accent-dark
                 active:scale-[0.98] transition-all duration-200 shadow-lg shadow-primary/25"
        >
          <Zap class="w-4 h-4" />
          连接并启动 Widget
          <ArrowRight class="w-4 h-4" />
        </button>
      </form>

      <!-- Hint -->
      <div class="mt-6 p-4 bg-slate-800/40 rounded-xl border border-slate-700/30">
        <p class="text-xs text-slate-400 leading-relaxed">
          提示：请先在 LibreDesk 管理后台 <strong class="text-slate-300">Settings &rarr; Inboxes</strong> 中创建一个
          <strong class="text-slate-300">Live Chat</strong> 类型的 Inbox，然后复制其 UUID 作为 Inbox ID。
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes fadeIn {
  from { opacity: 0; transform: scale(0.95) translateY(10px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}
</style>
