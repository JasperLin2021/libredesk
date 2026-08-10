<script setup>
import { ref } from 'vue'
import {
  Eye, EyeOff, Trash2, RotateCcw, LogOut, UserPlus,
  MessageCircle, Wifi, WifiOff, AlertCircle, Settings, X
} from 'lucide-vue-next'

const props = defineProps({
  isReady: Boolean,
  isVisible: Boolean,
  unreadCount: Number,
  error: String,
  config: Object,
})

const emit = defineEmits(['show', 'hide', 'setUser', 'logout', 'destroy', 'reload', 'openConfig'])

const showJWTInput = ref(false)
const jwtValue = ref('')

function handleSetUser() {
  if (jwtValue.value.trim()) {
    emit('setUser', jwtValue.value.trim())
    jwtValue.value = ''
    showJWTInput.value = false
  }
}

function handleLogout() {
  emit('logout')
}

function handleDestroy() {
  emit('destroy')
}

function handleReload() {
  emit('reload')
}
</script>

<template>
  <div class="fixed left-6 bottom-20 z-50 w-80 animate-[slideUp_0.35s_ease-out]">
    <div class="glass-dark overflow-hidden border border-slate-700/30">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-white/[0.06]">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-accent flex items-center justify-center">
            <MessageCircle class="w-4 h-4 text-white" />
          </div>
          <div>
            <h3 class="text-sm font-semibold text-white">Widget 控制台</h3>
            <p class="text-[11px] text-slate-500">LibreDesk API 演示</p>
          </div>
        </div>
      </div>

      <!-- Content -->
      <div class="p-4 space-y-4">
        <!-- Status -->
        <div class="space-y-2">
          <!-- Connection Status -->
          <div class="flex items-center justify-between px-3 py-2.5 bg-slate-800/40 rounded-lg">
            <span class="text-xs text-slate-400">连接状态</span>
            <div class="flex items-center gap-1.5">
              <Wifi v-if="isReady" class="w-3.5 h-3.5 text-green-400" />
              <WifiOff v-else class="w-3.5 h-3.5 text-slate-600" />
              <span class="text-xs font-medium" :class="isReady ? 'text-green-400' : 'text-slate-500'">
                {{ isReady ? '已连接' : '未连接' }}
              </span>
            </div>
          </div>

          <!-- Visibility -->
          <div class="flex items-center justify-between px-3 py-2.5 bg-slate-800/40 rounded-lg">
            <span class="text-xs text-slate-400">聊天窗口</span>
            <div class="flex items-center gap-1.5">
              <Eye v-if="isVisible" class="w-3.5 h-3.5 text-blue-400" />
              <EyeOff v-else class="w-3.5 h-3.5 text-slate-500" />
              <span class="text-xs font-medium" :class="isVisible ? 'text-blue-400' : 'text-slate-500'">
                {{ isVisible ? '已打开' : '已关闭' }}
              </span>
            </div>
          </div>

          <!-- Unread Count -->
          <div class="flex items-center justify-between px-3 py-2.5 bg-slate-800/40 rounded-lg">
            <span class="text-xs text-slate-400">未读消息</span>
            <span
              class="inline-flex items-center justify-center min-w-[22px] h-[22px] px-1.5 text-[11px] font-bold rounded-full"
              :class="unreadCount > 0 ? 'bg-red-500 text-white' : 'bg-slate-700 text-slate-400'"
            >
              {{ unreadCount }}
            </span>
          </div>
        </div>

        <!-- Error -->
        <div v-if="error" class="flex items-start gap-2 px-3 py-2.5 bg-red-500/10 border border-red-500/20 rounded-lg">
          <AlertCircle class="w-3.5 h-3.5 text-red-400 mt-0.5 shrink-0" />
          <p class="text-xs text-red-300 leading-relaxed">{{ error }}</p>
        </div>

        <!-- Config Info -->
        <div class="px-3 py-2 bg-slate-800/30 rounded-lg space-y-1">
          <div class="flex items-center justify-between">
            <span class="text-[10px] text-slate-600 uppercase tracking-wider">Backend</span>
            <span class="text-[11px] text-slate-400 font-mono truncate max-w-[180px]">{{ config.baseURL }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] text-slate-600 uppercase tracking-wider">Inbox</span>
            <span class="text-[11px] text-slate-400 font-mono truncate max-w-[150px]">{{ config.inboxID?.slice(0, 8) }}...</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="space-y-2">
          <p class="text-[10px] text-slate-600 uppercase tracking-wider px-1">Widget API</p>

          <div class="grid grid-cols-2 gap-2">
            <button
              @click="emit('show')"
              :disabled="!isReady"
              class="flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium
                     bg-blue-500/10 border border-blue-500/20 text-blue-400
                     hover:bg-blue-500/20 active:scale-[0.97] transition-all duration-150
                     disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100"
            >
              <Eye class="w-3.5 h-3.5" /> show()
            </button>
            <button
              @click="emit('hide')"
              :disabled="!isReady"
              class="flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium
                     bg-slate-500/10 border border-slate-500/20 text-slate-400
                     hover:bg-slate-500/20 active:scale-[0.97] transition-all duration-150
                     disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100"
            >
              <EyeOff class="w-3.5 h-3.5" /> hide()
            </button>
          </div>

          <div class="grid grid-cols-2 gap-2">
            <button
              @click="showJWTInput = !showJWTInput"
              :disabled="!isReady"
              class="flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium
                     bg-purple-500/10 border border-purple-500/20 text-purple-400
                     hover:bg-purple-500/20 active:scale-[0.97] transition-all duration-150
                     disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100"
            >
              <UserPlus class="w-3.5 h-3.5" /> setUser()
            </button>
            <button
              @click="handleLogout"
              :disabled="!isReady"
              class="flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium
                     bg-amber-500/10 border border-amber-500/20 text-amber-400
                     hover:bg-amber-500/20 active:scale-[0.97] transition-all duration-150
                     disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100"
            >
              <LogOut class="w-3.5 h-3.5" /> logout()
            </button>
          </div>

          <!-- JWT Input -->
          <Transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 max-h-0"
            enter-to-class="opacity-100 max-h-20"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100 max-h-20"
            leave-to-class="opacity-0 max-h-0"
          >
            <div v-if="showJWTInput" class="space-y-2 overflow-hidden">
              <input
                v-model="jwtValue"
                type="text"
                placeholder="粘贴 JWT Token..."
                class="w-full px-3 py-2 bg-slate-800/60 border border-slate-700/50 rounded-lg text-xs text-white
                       placeholder-slate-600 focus:outline-none focus:border-purple-500/40 transition-colors"
              />
              <button
                @click="handleSetUser"
                class="w-full px-3 py-2 bg-purple-500/20 border border-purple-500/30 text-purple-300 text-xs font-medium
                       rounded-lg hover:bg-purple-500/30 active:scale-[0.97] transition-all duration-150"
              >
                确认设置用户
              </button>
            </div>
          </Transition>

          <!-- Lifecycle -->
          <div class="grid grid-cols-3 gap-2 pt-1 border-t border-white/[0.06]">
            <button
              @click="handleReload"
              class="flex items-center justify-center gap-1 px-2 py-2 rounded-lg text-[11px] font-medium
                     bg-emerald-500/10 border border-emerald-500/20 text-emerald-400
                     hover:bg-emerald-500/20 active:scale-[0.97] transition-all duration-150"
            >
              <RotateCcw class="w-3 h-3" /> 重载
            </button>
            <button
              @click="handleDestroy"
              class="flex items-center justify-center gap-1 px-2 py-2 rounded-lg text-[11px] font-medium
                     bg-red-500/10 border border-red-500/20 text-red-400
                     hover:bg-red-500/20 active:scale-[0.97] transition-all duration-150"
            >
              <Trash2 class="w-3 h-3" /> 销毁
            </button>
            <button
              @click="emit('openConfig')"
              class="flex items-center justify-center gap-1 px-2 py-2 rounded-lg text-[11px] font-medium
                     bg-slate-500/10 border border-slate-500/20 text-slate-400
                     hover:bg-slate-500/20 active:scale-[0.97] transition-all duration-150"
            >
              <Settings class="w-3 h-3" /> 配置
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes slideUp {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
