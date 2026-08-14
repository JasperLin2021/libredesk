<template>
  <div class="flex flex-col h-full">
    <!-- Chat header -->
    <ChatHeader @goBack="goBack" />

    <!-- Pre-chat form -->
    <PreChatForm
      v-if="showPreChatForm"
      @submit="handlePreChatFormSubmit"
      :exclude-default-fields="!!userStore.userSessionToken"
      :is-submitting="isInitializing"
      class="flex-1 min-h-0"
    />

    <!-- Messages container (when no pre-chat form) -->
    <ChatMessages v-else ref="chatMessages" :showPreChatForm="showPreChatForm" @error="handleError" />

    <!-- Error display -->
    <WidgetError :errorMessage="errorMessage" />

    <!-- Message input (only when pre-chat form is not shown) -->
    <MessageInput v-if="!showPreChatForm && !isConversationClosed" @error="handleError" />

    <!-- Closed conversation notice -->
    <div v-if="isConversationClosed" class="border-t p-4 text-center text-sm text-muted-foreground">
      <p class="mb-3">{{ $t('widget.conversationClosed') }}</p>
      <button
        @click="startNewConversation"
        class="px-4 py-2 text-sm font-medium text-primary-500 bg-primary-50 hover:bg-primary-100 rounded-lg transition-colors"
      >
        {{ $t('widget.startNewConversation') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useWidgetStore } from '../store/widget.js'
import { useUserStore } from '../store/user.js'
import { useChatStore } from '../store/chat.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api, { saveSession } from '@widget/api/index.js'
import WidgetError from '@widget/components/WidgetError.vue'
import ChatHeader from '@widget/components/ChatHeader.vue'
import ChatMessages from '@widget/components/ChatMessages.vue'
import MessageInput from '@widget/components/MessageInput.vue'
import PreChatForm from '@widget/components/PreChatForm.vue'

const widgetStore = useWidgetStore()
const userStore = useUserStore()
const chatStore = useChatStore()
const errorMessage = ref('')
const preChatFormSubmitted = ref(false)
const isInitializing = ref(false)
const config = computed(() => widgetStore.config)

// Determine if pre-chat form should be shown
const showPreChatForm = computed(() => {
  const preChatForm = config.value?.prechat_form

  // Must be enabled and not submitted
  if (!preChatForm?.enabled || preChatFormSubmitted.value) {
    return false
  }

  // Atleast one field must be enabled
  const hasEnabledFields = preChatForm.fields?.some((field) => field.enabled)
  if (!hasEnabledFields) {
    return false
  }

  // If conversation data is already loaded (e.g., from "start new conversation"), skip the form
  if (chatStore.getCurrentConversationMessages?.length > 0) {
    return false
  }

  const isAnonymous = !userStore.userSessionToken
  const isNewConversation = !!userStore.userSessionToken && !chatStore.currentConversation?.uuid
  return isAnonymous || isNewConversation
})

// Check if conversation is closed - always block replies on closed conversations
const isConversationClosed = computed(() => {
  const status = chatStore.currentConversation?.status
  return status === 'Closed'
})

// Start a new conversation when current one is closed
const startNewConversation = async () => {
  // Clear current conversation
  chatStore.setCurrentConversation(null)
  chatStore.clearMessages()

  // Call initChatConversation to reopen closed conversation and send welcome message
  // This ensures the user sees welcome message and preset questions immediately
  try {
    const resp = await api.initChatConversation({})
    const { conversation, session_token, user, messages, business_hours_id, working_hours_utc_offset } = resp.data.data
    conversation.business_hours_id = business_hours_id
    conversation.working_hours_utc_offset = working_hours_utc_offset

    if (!userStore.userSessionToken && session_token) {
      saveSession(session_token, user, userStore, true)
    }

    chatStore.addConversationToList(conversation)
    chatStore.setCurrentConversation(conversation)
    chatStore.replaceMessages(messages || [])
    preChatFormSubmitted.value = true
  } catch (error) {
    console.error('Error initializing conversation:', error)
    errorMessage.value = handleHTTPError(error).message
  }
}

const goBack = () => {
  widgetStore.navigateToMessages()
}

const handleError = (message) => {
  errorMessage.value = message
  if (message) {
    setTimeout(() => {
      errorMessage.value = ''
    }, 5000)
  }
}

// Auto-init conversation on mount to fetch welcome messages
onMounted(async () => {
  // Only auto-init if no current conversation and not already initializing
  if (!chatStore.currentConversation?.uuid && !isInitializing.value) {
    await initConversationForWelcome()
  }
})

// Initialize conversation without message to get welcome reply
const initConversationForWelcome = async () => {
  isInitializing.value = true
  errorMessage.value = ''

  try {
    const payload = {}

    // If user has session token, include it
    if (userStore.userSessionToken) {
      // User is already authenticated, just init without message
    }

    const resp = await api.initChatConversation(payload)
    const { conversation, session_token, user, messages, business_hours_id, working_hours_utc_offset } = resp.data.data
    conversation.business_hours_id = business_hours_id
    conversation.working_hours_utc_offset = working_hours_utc_offset

    console.log('[DEBUG] Welcome init - API response messages:', messages)

    if (!userStore.userSessionToken && session_token) {
      saveSession(session_token, user, userStore, true)
    }

    chatStore.addConversationToList(conversation)
    chatStore.setCurrentConversation(conversation)
    chatStore.replaceMessages(messages)

    preChatFormSubmitted.value = true
  } catch (error) {
    console.error('[DEBUG] Welcome init error:', error)
    errorMessage.value = handleHTTPError(error).message
  } finally {
    isInitializing.value = false
  }
}

// Handle pre-chat form submission - init chat with form data and optional message
const handlePreChatFormSubmit = async ({ formData, message }) => {
  isInitializing.value = true
  errorMessage.value = ''

  try {
    const payload = {}

    if (message) {
      payload.message = message
    }

    if (Object.keys(formData).length > 0) {
      payload.form_data = formData
    }

    const resp = await api.initChatConversation(payload)
    const { conversation, session_token, user, messages, business_hours_id, working_hours_utc_offset } = resp.data.data
    conversation.business_hours_id = business_hours_id
    conversation.working_hours_utc_offset = working_hours_utc_offset

    console.log('[DEBUG] API response messages:', messages)
    console.log('[DEBUG] Messages count:', messages?.length)

    if (!userStore.userSessionToken && session_token) {
      saveSession(session_token, user, userStore, true)
    }

    chatStore.addConversationToList(conversation)
    chatStore.setCurrentConversation(conversation)
    chatStore.replaceMessages(messages)

    console.log('[DEBUG] After replaceMessages, current messages:', chatStore.getCurrentConversationMessages)

    preChatFormSubmitted.value = true
  } catch (error) {
    errorMessage.value = handleHTTPError(error).message
  } finally {
    isInitializing.value = false
  }
}
</script>
