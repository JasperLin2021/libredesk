<template>
  <div class="flex flex-col items-start max-w-[85%]">
    <!-- Message bubble with quick reply buttons -->
    <div class="px-4 py-3 rounded-2xl text-sm leading-5 break-words bg-muted text-foreground">
      <span v-if="contentWithoutTransfer" class="whitespace-pre-wrap">{{ contentWithoutTransfer }}</span>

      <!-- Quick reply buttons -->
      <div v-if="items.length > 0" class="flex flex-col gap-2 mt-3">
        <button
          v-for="item in items"
          :key="item.value"
          type="button"
          :disabled="isDisabled(item.value)"
          :class="[
            'text-left text-sm font-medium px-3 py-2 rounded-lg border transition-all duration-150',
            isDisabled(item.value)
              ? 'opacity-40 cursor-not-allowed border-muted-foreground/20 bg-muted-foreground/10 text-muted-foreground'
              : 'cursor-pointer border-primary/25 bg-primary/5 text-primary hover:bg-primary/10 hover:border-primary/40'
          ]"
          @click="sendQuickReply(item.value)"
        >
          {{ item.label }}
        </button>
      </div>

      <!-- Transfer to human button -->
      <button
        v-if="transferKeyword"
        type="button"
        :disabled="isDisabled(transferKeyword)"
        :class="[
          'mt-3 w-full text-left text-sm font-medium px-3 py-2 rounded-lg border transition-all duration-150',
          isDisabled(transferKeyword)
            ? 'opacity-40 cursor-not-allowed border-muted-foreground/20 bg-muted-foreground/10 text-muted-foreground'
            : 'cursor-pointer border-secondary/40 bg-secondary/10 text-foreground hover:bg-secondary/20 hover:border-secondary/60'
        ]"
        @click="sendQuickReply(transferKeyword)"
      >
        {{ transferKeyword }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useWidgetStore } from '../store/widget.js'
import { useUserStore } from '../store/user.js'
import { useChatStore } from '../store/chat.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@widget/api/index.js'

const props = defineProps({
  message: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['error'])

const chatStore = useChatStore()
const userStore = useUserStore()
const widgetStore = useWidgetStore()

// Values that have already been clicked (and sent) are disabled.
const clickedValues = ref(new Set())

const items = computed(() => props.message?.meta?.items || [])
const transferKeyword = computed(() => props.message?.meta?.transfer_keyword || '')

// The transfer keyword line is rendered as a dedicated button, so it is
// stripped from the bubble body text.
const contentWithoutTransfer = computed(() => {
  const content = props.message?.content || ''
  const keyword = transferKeyword.value
  if (!keyword) return content
  return content
    .split('\n')
    .filter((line) => line.trim() !== keyword.trim())
    .join('\n')
})

const isDisabled = (value) => clickedValues.value.has(value)

const sendQuickReply = async (value) => {
  if (!value || clickedValues.value.has(value)) return

  // Immediately disable the button for instant feedback.
  const next = new Set(clickedValues.value)
  next.add(value)
  clickedValues.value = next

  const conversationUUID = chatStore.currentConversation?.uuid
  if (!conversationUUID) return

  const tempMessageID = chatStore.addPendingMessage(
    conversationUUID,
    value,
    userStore.isVisitor ? 'visitor' : 'contact',
    userStore.userID
  )

  try {
    const resp = await api.sendChatMessage(conversationUUID, { message: value })
    const data = resp.data.data

    // The response may be { message, bot_messages } when quick reply
    // processing creates automatic bot replies.
    const userMessage = data?.message || data
    const botMessages = data?.bot_messages || []

    if (tempMessageID && userMessage) {
      chatStore.replaceMessage(conversationUUID, tempMessageID, userMessage)
    }
    if (userMessage) {
      chatStore.updateConversationListLastMessage(conversationUUID, userMessage)
    }

    // Append any bot messages created by quick reply matching.
    if (Array.isArray(botMessages)) {
      botMessages.forEach((botMsg) => {
        chatStore.addMessageToConversation(conversationUUID, botMsg)
      })
    }

    emit('error', '')
  } catch (error) {
    if (tempMessageID) {
      chatStore.removeMessage(conversationUUID, tempMessageID)
    }
    emit('error', handleHTTPError(error).message)
  }
}
</script>
