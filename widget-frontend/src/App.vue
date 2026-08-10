<script setup>
import { ref, onMounted } from 'vue'
import { useWidget } from './composables/useWidget.js'
import ConfigPanel from './components/ConfigPanel.vue'
import DemoPage from './components/DemoPage.vue'

const STORAGE_KEY = 'libredesk_demo_config'

const {
  isReady,
  isVisible,
  unreadCount,
  error,
  config,
  initWidget,
  show,
  hide,
  setUser,
  logout,
  destroy,
} = useWidget()

const hasConfig = ref(false)
const showConfigPanel = ref(false)

function loadConfig() {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored) {
    const parsed = JSON.parse(stored)
    if (parsed.baseURL && parsed.inboxID) {
      return parsed
    }
  }
  return null
}

function handleConfigSaved({ baseURL, inboxID }) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ baseURL, inboxID }))
  hasConfig.value = true
  showConfigPanel.value = false
  initWidget(baseURL, inboxID)
}

function handleOpenConfig() {
  showConfigPanel.value = true
}

function handleReloadWidget() {
  const stored = loadConfig()
  if (stored) {
    destroy()
    setTimeout(() => {
      initWidget(stored.baseURL, stored.inboxID)
    }, 300)
  }
}

function handleClearConfig() {
  destroy()
  localStorage.removeItem(STORAGE_KEY)
  hasConfig.value = false
  showConfigPanel.value = true
}

onMounted(() => {
  const stored = loadConfig()
  if (stored) {
    hasConfig.value = true
    initWidget(stored.baseURL, stored.inboxID)
  } else {
    showConfigPanel.value = true
    hasConfig.value = false
  }
})
</script>

<template>
  <div class="min-h-screen bg-white">
    <!-- Config Panel Modal -->
    <ConfigPanel
      v-if="showConfigPanel"
      :default-base-url="config.baseURL"
      :default-inbox-id="config.inboxID"
      @saved="handleConfigSaved"
    />

    <!-- Main Demo Page -->
    <DemoPage
      v-if="hasConfig"
      :is-ready="isReady"
      :is-visible="isVisible"
      :unread-count="unreadCount"
      :error="error"
      :config="config"
      @show="show"
      @hide="hide"
      @set-user="setUser"
      @logout="logout"
      @destroy="handleClearConfig"
      @reload="handleReloadWidget"
      @open-config="handleOpenConfig"
    />
  </div>
</template>
