<template>
  <div
    class="libredesk-widget-app text-foreground bg-background"
    :class="{ dark: widgetStore.config.dark_mode, mobile: widgetStore.isMobileFullScreen }"
    :style="customColorStyle"
    @click.once="initAudioContext"
    @touchstart.once="initAudioContext"
  >
    <div class="widget-container">
      <MainLayout />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, watch, getCurrentInstance } from 'vue'
import { useWidgetStore } from './store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from './store/user.js'
import { initWidgetWS, closeWidgetWebSocket, sendPageVisit } from './websocket.js'
import api, { setApiSessionToken, setVisitorToken, initVisitorToken, getVisitorToken, saveSession, registerStores } from '@widget/api/index.js'
import { useUnreadCount } from './composables/useUnreadCount.js'
import { initAudioContext } from '@shared-ui/composables/useNotificationSound.js'
import { hexToHSL, getContrastingHSL } from '@shared-ui/utils/color.js'
import { resolveInit } from './state.js'
import MainLayout from '@widget/layouts/MainLayout.vue'

// Helper to read cookies (mirrors widget.js getCookie)
function getCookie(name) {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '=([^;]*)'))
  return match ? decodeURIComponent(match[1]) : null
}

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()

// Register stores for the global 401 response interceptor.
registerStores({ userStore, chatStore, widgetStore })

// Initialize unread count tracking and sending to parent window.
useUnreadCount()

const widgetConfig = getCurrentInstance().appContext.config.globalProperties.$widgetConfig
if (widgetConfig) {
  widgetStore.updateConfig(widgetConfig)
}

const customColorStyle = computed(() => {
  const style = {}
  const colors = widgetStore.config.colors
  if (colors?.primary) {
    style['--primary'] = hexToHSL(colors.primary)
    style['--primary-foreground'] = getContrastingHSL(colors.primary)
  }
  return style
})

onMounted(() => {
  setupParentMessageListeners()
  window.parent.postMessage({ type: 'VUE_APP_READY' }, '*')
})

const signalWidgetLoaded = () => {
  window.parent.postMessage({ type: 'WIDGET_LOADED' }, '*')
}

// Helper: init conversation via visitor_token (used by both fresh visitor path and session-expiry fallback).
const initViaVisitorToken = async (isReturningVisitor) => {
  try {
    const initPayload = isReturningVisitor ? { skip_welcome: true } : {}
    const visitorToken = getVisitorToken()
    if (visitorToken) {
      initPayload.visitor_token = visitorToken
    }
    const resp = await api.initChatConversation(initPayload)
    const { conversation, session_token, visitor_token, user, messages, business_hours_id, working_hours_utc_offset } = resp.data.data
    conversation.business_hours_id = business_hours_id
    conversation.working_hours_utc_offset = working_hours_utc_offset

    // Save visitor token from backend (the authoritative token stored in DB).
    // This ensures the cookie always matches the DB, enabling future recovery.
    if (visitor_token) {
      setVisitorToken(visitor_token)
    }

    if (!userStore.userSessionToken && session_token) {
      saveSession(session_token, user, userStore, false)
    }

    chatStore.addConversationToList(conversation)
    chatStore.setCurrentConversation(conversation)
    chatStore.replaceMessages(messages || [])
  } catch (err) {
    console.error('Failed to init visitor conversation:', err)
  }
}

const fetchInitialConversations = async (isReturningVisitor = false) => {
  // For visitor users (no session token), call initChatConversation to reuse/reopen existing conversation.
  // This handles both cases: with visitor token (reuse) and without (create new).
  if (!userStore.userSessionToken) {
    await initViaVisitorToken(isReturningVisitor)
    return
  }

  // Path B: has session token — try authenticated fetch first.
  // Kick off auth/me in parallel with the conversation list fetch so user
  // metadata (including avatar_url) is available as early as possible. Without
  // this, a message sent right after a hard refresh would briefly show the
  // fallback letter instead of the real avatar.
  const mePromise = api
    .getAuthMe()
    .then((resp) => resp?.data?.data || null)
    .catch(() => null)
  const success = await chatStore.fetchConversations()
  if (success) {
    const me = await mePromise
    if (me) userStore.setUserMeta(me)
    if (chatStore.hasConversations) {
      try {
        await chatStore.loadConversation(chatStore.getConversations[0].uuid)
      } catch { /* non-blocking */ }
    }
    if (widgetStore.config?.direct_to_conversation) {
      widgetStore.navigateToChat()
    }
    return
  }

  // Path B failed (likely 401 = expired session). Fallback to visitor_token recovery.
  // handleSessionExpired() already cleared the expired session token and chat state.
  setApiSessionToken('')
  await initViaVisitorToken(true)
}

// Listen for messages from parent window (widget.js)
const setupParentMessageListeners = () => {
  window.addEventListener('message', async (event) => {
    if (event.data.type == 'WIDGET_CLOSED') {
      widgetStore.setOpen(false)
    } else if (event.data.type === 'WIDGET_OPENED') {
      widgetStore.setOpen(true)
    } else if (event.data.type === 'SET_MOBILE_STATE') {
      widgetStore.setMobileFullScreen(event.data.isMobile)
    } else if (event.data.type === 'WIDGET_EXPANDED') {
      widgetStore.setExpanded(event.data.isExpanded)
    } else if (event.data.type === 'SESSION_DATA') {
      if (event.data.visitorToken) {
        initVisitorToken(event.data.visitorToken)
      }
      const sessionToken = event.data.sessionToken
      const isNewSession = event.data.isNewSession
      // Check if this is a returning visitor BEFORE setting the session token.
      // A session cookie means the visitor was here before.
      const inboxId = new URLSearchParams(window.location.search).get('inbox_id')
      const sessionCookieName = `libredesk-session-${inboxId}`
      const isReturningVisitor = !!(sessionToken && getCookie(sessionCookieName))
      try {
        if (sessionToken) {
          userStore.setSessionToken(sessionToken)
          setApiSessionToken(sessionToken)
        }

        if (isNewSession && sessionToken) {
          // First-time Bixiao auth: clear anonymous state and call initChatConversation
          // to trigger backend's create/reuse/reopen conversation logic.
          if (userStore.isVisitor) {
            userStore.clearSessionToken()
          }
          userStore.setSessionToken(sessionToken)
          setApiSessionToken(sessionToken)
          const resp = await api.initChatConversation({})
          const { conversation, messages, business_hours_id, working_hours_utc_offset } = resp.data.data
          conversation.business_hours_id = business_hours_id
          conversation.working_hours_utc_offset = working_hours_utc_offset
          chatStore.addConversationToList(conversation)
          chatStore.setCurrentConversation(conversation)
          chatStore.replaceMessages(messages || [])
          widgetStore.navigateToChat()
        } else {
          await fetchInitialConversations(isReturningVisitor)
        }
      } finally {
        resolveInit()
        signalWidgetLoaded()
      }
    } else if (event.data.type === 'SET_JWT_TOKEN') {
      if (event.data.visitorToken) {
        initVisitorToken(event.data.visitorToken)
      }
      if (event.data.jwt) {
        try {
          const resp = await api.exchangeJWTForSession(event.data.jwt)
          const { session_token, user } = resp.data.data
          saveSession(session_token, user, userStore)
          chatStore.conversations = null
          const initResp = await api.initChatConversation({})
          const { conversation, messages, business_hours_id, working_hours_utc_offset } = initResp.data.data
          conversation.business_hours_id = business_hours_id
          conversation.working_hours_utc_offset = working_hours_utc_offset
          chatStore.addConversationToList(conversation)
          chatStore.setCurrentConversation(conversation)
          chatStore.replaceMessages(messages || [])
          resolveInit()
        } catch (err) {
          console.error('Failed to exchange JWT for session:', err)
          resolveInit()
        } finally {
          signalWidgetLoaded()
        }
      }
    } else if (event.data.type === 'CLEAR_SESSION') {
      userStore.clearSessionToken()
    } else if (event.data.type === 'PAGE_VISIT') {
      sendPageVisit(event.data.url, event.data.title)
    }
  })
}

const initializeWebSocket = () => {
  const token = userStore.userSessionToken
  if (token) {
    const urlParams = new URLSearchParams(window.location.search)
    const inboxId = urlParams.get('inbox_id')
    if (inboxId) {
      initWidgetWS(token, inboxId)
    } else {
      console.error('Cannot initialize WebSocket: missing `inbox_id`')
    }
  } else {
    closeWidgetWebSocket()
  }
}

watch(
  () => userStore.userSessionToken,
  (newToken) => {
    if (newToken) {
      initializeWebSocket()
    } else {
      closeWidgetWebSocket()
    }
  }
)
</script>

<style scoped>
.libredesk-widget-app {
  width: 100vw;
  height: 100dvh;
  overflow: hidden;
}

.widget-container {
  width: 100%;
  height: 100%;
}

/* iOS Safari auto-zooms on focus when font-size < 16px. Force 16px on mobile to prevent it. */
.mobile :deep(input),
.mobile :deep(textarea),
.mobile :deep(select) {
  font-size: 16px;
}
</style>
