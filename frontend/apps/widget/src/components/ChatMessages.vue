<template>
  <div class="flex flex-col relative flex-1 min-h-0">
    <!-- Loading conversation overlay -->
    <div v-if="isLoadingConversation" class="absolute inset-0 bg-background z-10" role="status">
      <Spinner size="md" :text="$t('globals.terms.loading')" absolute />
    </div>
    <div
      class="flex-1 min-h-0 overflow-y-auto [overflow-anchor:none] scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50"
      ref="messagesContainer"
      @scroll="handleScroll"
    >
      <div ref="contentEl" class="p-4 flex flex-col gap-4">
        <!-- Chat Intro -->
        <ChatIntro v-if="!props.showPreChatForm" :introText="config.chat_introduction" />

        <!-- Notice -->
        <NoticeBanner
          v-if="config.notice_banner.enabled === true && !props.showPreChatForm"
          :noticeText="config.notice_banner.text"
        />

        <!-- Messages -->
        <TransitionGroup
          tag="div"
          enter-active-class="animate-slide-in"
          class="flex flex-col gap-4"
        >
          <div
            v-for="message in chatStore.getCurrentConversationMessages"
            :key="message.uuid"
            :class="['flex flex-col gap-1', isUserMessage(message) ? 'items-end' : 'items-start']"
          >
            <!-- Auto-reply header: avatar + bot name side-by-side (above the bubble).
                 Queue-info bubbles still render the header because the bubble itself belongs
                 to the auto-reply flow. CSAT bubbles use their own UI and skip this. -->
            <div v-if="isAutoReply(message) && !isCsat(message)" class="flex items-center gap-2">
              <Avatar class="size-8 flex-shrink-0">
                <AvatarImage :src="getMessageAvatarUrl(message)" />
                <AvatarFallback>{{ getMessageAvatarFallback(message) }}</AvatarFallback>
              </Avatar>
              <span class="text-sm font-medium text-foreground">
                {{ config.bot_name || message.author.first_name }}
              </span>
            </div>

            <!-- Avatar + bubble row. For auto-replies the bubble is indented to align with the
                 bot name (avatar is rendered in the header row above instead). The avatar always
                 aligns with the top of the bubble, matching the user message layout. -->
            <div
              :class="[
                'flex gap-2',
                isUserMessage(message) ? 'flex-row-reverse items-start' : 'flex-row items-start',
                isAutoReply(message) && !isCsat(message) ? 'ml-10' : ''
              ]"
            >
              <!-- Avatar column (skipped for auto-replies — header row carries the avatar —
                   and for CSAT which has its own dedicated UI). -->
              <Avatar v-if="!isAutoReply(message) && !isCsat(message)" class="size-8 flex-shrink-0">
                <AvatarImage :src="getMessageAvatarUrl(message)" />
                <AvatarFallback>{{ getMessageAvatarFallback(message) }}</AvatarFallback>
              </Avatar>

              <!-- Message content column (no min-w-0 so inner bubbles can shrink-to-fit correctly). -->
              <div
                :class="[
                  'flex flex-col max-w-full',
                  isUserMessage(message) ? 'items-end' : 'items-start'
                ]"
              >
                <!-- Quick Reply Card (bot) -->
                <QuickReplyCard
                  v-if="message.meta?.type === 'bot_quick_reply'"
                  :message="message"
                  @error="handleQuickReplyError"
                />

                <!-- Queue Info Bubble (transfer to human queue position) -->
                <div
                  v-else-if="message.meta?.type === 'queue_info'"
                  class="flex flex-col items-start max-w-[85%]"
                >
                  <div
                    class="px-4 py-3 rounded-2xl text-sm leading-5 break-words whitespace-pre-wrap bg-muted text-foreground"
                  >
                    {{ message.content }}
                  </div>
                  <div
                    v-if="typeof message.meta?.count === 'number'"
                    class="mt-2 flex items-center gap-2 text-xs px-3 py-2 rounded-lg bg-amber-50 border border-amber-200 text-amber-800"
                    role="status"
                  >
                    <Clock class="w-3.5 h-3.5 shrink-0" />
                    <span>{{ $t('widget.queuePosition', { count: message.meta.count }) }}</span>
                  </div>
                </div>

                <!-- CSAT Message Bubble -->
                <CSATMessageBubble
                  v-else-if="message.meta?.is_csat"
                  :message="message"
                  @submitted="handleCSATSubmitted"
                />

                <!-- Regular Message Bubble -->
                <div
                  v-else
                  class="box-content max-w-[85%] whitespace-pre-wrap break-words px-4 py-3 rounded-2xl text-sm leading-5 w-fit transition-all duration-200"
                  :class="[
                    message.author.type === 'contact' || message.author.type === 'visitor'
                      ? [
                          'text-primary-foreground',
                          message.status === 'sending' || message.status === 'uploading'
                            ? 'bg-primary/60'
                            : message.status === 'failed'
                              ? 'bg-destructive/60'
                              : 'bg-primary'
                        ]
                      : 'bg-muted text-foreground',
                    {
                      'show-quoted-text': isQuotedTextVisible(message.uuid),
                      'hide-quoted-text': !isQuotedTextVisible(message.uuid)
                    }
                  ]"
                >
                  <!-- Message content -->
                  <span v-if="message.content_type === 'text'">{{ message.content }}</span>
                  <Letter
                    v-else
                    :html="message.content"
                    :allowedSchemas="['cid', 'https', 'http', 'mailto']"
                    :allowed-css-properties="extendedCssProperties"
                    class="native-html"
                  />
                  <div
                    v-if="containsQuoteMarkers(message.content)"
                    @click="toggleQuote(message.uuid)"
                    @keydown.enter.prevent="toggleQuote(message.uuid)"
                    @keydown.space.prevent="toggleQuote(message.uuid)"
                    tabindex="0"
                    role="button"
                    :aria-expanded="isQuotedTextVisible(message.uuid)"
                    :class="[
                      'text-xs cursor-pointer px-2 py-1 w-max rounded-md transition-all mt-1',
                      message.author.type === 'contact' || message.author.type === 'visitor'
                        ? 'text-primary-foreground/70 hover:bg-primary-foreground/10 hover:text-primary-foreground'
                        : 'text-muted-foreground hover:bg-muted hover:text-primary'
                    ]"
                  >
                    {{
                      isQuotedTextVisible(message.uuid)
                        ? t('conversation.hideQuotedText')
                        : t('conversation.showQuotedText')
                    }}
                  </div>
                  <!-- Show attachments if available -->
                  <MessageAttachment class="mt-1" :attachments="message.attachments" />
                </div>
              </div>
            </div>

            <!-- Message metadata (caption). Hidden for queue-info and CSAT bubbles. -->
            <div
              v-if="!isQueueInfo(message) && !isCsat(message)"
              :class="[
                'text-[10px] text-muted-foreground flex items-center gap-2',
                isAutoReply(message) ? 'h-4 ml-10' : ''
              ]"
            >
              <!-- Auto-replies: only show time below the bubble (name is rendered in the header row). -->
              <span v-if="isAutoReply(message)">
                {{ getMessageTime(message.created_at) }}
              </span>

              <!-- Agent name and time for human agent messages -->
              <span v-else-if="message.author.type === 'agent'">
                {{ message.author.first_name }} {{ message.author.last_name }}
                •
                {{ getMessageTime(message.created_at) }}
              </span>

              <!-- Delivery status for user messages -->
              <span v-else-if="isUserMessage(message)" class="flex items-center gap-1">
                <span
                  v-if="message.status === 'sending' || message.status === 'uploading'"
                  class="flex items-center gap-1"
                >
                  <div
                    class="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin"
                  ></div>
                  <span v-if="message.status === 'sending'">
                    {{ $t('globals.messages.sending') }}
                  </span>
                  <span v-if="message.status === 'uploading'">
                    {{ $t('globals.messages.uploading') }}
                  </span>
                </span>
                <span v-else>
                  {{ getMessageTime(message.created_at) }}
                </span>
              </span>
            </div>
          </div>
        </TransitionGroup>

        <!-- Typing Indicator -->
        <div v-if="isTyping" class="flex flex-col gap-1 items-start">
          <div class="flex items-center gap-2">
            <Avatar class="size-8 flex-shrink-0">
              <AvatarImage :src="inboxAvatarUrl" />
              <AvatarFallback>{{ inboxAvatarFallback }}</AvatarFallback>
            </Avatar>
            <span class="text-sm font-medium text-foreground">
              {{ config.bot_name || $t('conversation.support') }}
            </span>
          </div>
          <div
            class="ml-10 w-fit max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 bg-muted text-foreground"
          >
            <TypingIndicator />
          </div>
        </div>
      </div>
    </div>

    <!-- Sticky queue position footer: stays visible below the messages until
         the conversation is assigned to a human agent. -->
    <div
      v-if="showQueueInfoFooter"
      role="status"
      class="border-t border-amber-200 bg-amber-50 px-4 py-2.5 flex items-center justify-center gap-2 text-xs text-amber-800"
    >
      <Clock class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
      <span>{{ $t('widget.queuePosition', { count: queueInfoCount }) }}</span>
    </div>

    <!-- Sticky scroll to bottom button -->
    <ScrollToBottomButton
      :is-at-bottom="!hasUserScrolled"
      :unread-count="unreadMessages"
      @scroll-to-bottom="handleScrollToBottom"
    />
  </div>
</template>

<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { format } from 'date-fns'
import { Clock } from 'lucide-vue-next'
import { useDocumentVisibility, useDebounceFn } from '@vueuse/core'
import { useWidgetStore } from '../store/widget.js'
import { useChatStore } from '../store/chat.js'
import { useUserStore } from '../store/user.js'
import { useI18n } from 'vue-i18n'
import { Letter } from 'vue-letter'
import { allowedCssProperties } from 'lettersanitizer'
import ScrollToBottomButton from '@shared-ui/components/ScrollToBottomButton'
import ChatIntro from './ChatIntro.vue'
import NoticeBanner from './NoticeBanner.vue'
import MessageAttachment from './MessageAttachment.vue'
import CSATMessageBubble from './CSATMessageBubble.vue'
import QuickReplyCard from './QuickReplyCard.vue'
import { TypingIndicator } from '@shared-ui/components/TypingIndicator'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { containsQuoteMarkers } from '@shared-ui/utils/quotedContent.js'
import { useStickyScroll } from '@shared-ui/composables'

const extendedCssProperties = [...allowedCssProperties, 'transform', 'transform-origin']

const props = defineProps({
  showPreChatForm: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['error'])

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const messagesContainer = ref(null)
const contentEl = ref(null)
const unreadMessages = ref(0)
const currentConversationUUID = ref('')
const quotedTextState = ref({})
const { t } = useI18n()

const { hasUserScrolled, scrollToBottom, handleScroll } = useStickyScroll(
  messagesContainer,
  contentEl,
  {
    onArriveBottom: () => {
      unreadMessages.value = 0
    }
  }
)

const config = computed(() => widgetStore.config)
const isTyping = computed(() => chatStore.isTyping)
const isLoadingConversation = computed(() => chatStore.isLoadingConversation)

// Persistent queue footer visibility. Shown while the current conversation is
// still open, not yet assigned to any agent/team and the server has persisted a
// numeric queue count. Once the conversation gets assigned, the broadcast sets
// meta.queue_info to null (and assignee becomes non-empty), hiding the footer.
const showQueueInfoFooter = computed(() => {
  const conversation = chatStore.currentConversation
  if (!conversation?.uuid) return false
  if (conversation.status !== 'Open') return false
  if (conversation.assignee) return false
  const count = conversation.meta?.queue_info?.count
  return typeof count === 'number'
})

const queueInfoCount = computed(() => {
  return chatStore.currentConversation?.meta?.queue_info?.count ?? 0
})

const userStore = useUserStore()

// Author-type helpers for the message row layout and avatars.
const isUserMessage = (message) => {
  return message.author?.type === 'contact' || message.author?.type === 'visitor'
}

// Auto-replies (system user "System" / AI assistant) show the inbox avatar and use
// the configurable bot_name in the message caption.
const isAutoReply = (message) => {
  return (
    message.author?.type === 'ai_assistant' ||
    (message.author?.type === 'agent' && message.author?.first_name === 'System')
  )
}

// Queue info and CSAT keep their original special layout without avatars.
// Queue-info bubbles still belong to the auto-reply flow (transfer, etc.) so the header row
// (avatar + bot name) is rendered — but the avatar inside the bubble row is skipped because
// the bubble already has a built-in icon. CSAT bubbles are end-of-conversation UI and are
// handled by a dedicated component, so they skip the per-message avatar / meta row entirely.
const isQueueInfo = (message) => message.meta?.type === 'queue_info'
const isCsat = (message) => message.meta?.is_csat === true

const getMessageAvatarUrl = (message) => {
  if (isUserMessage(message)) {
    return message.author?.avatar_url || ''
  }
  // Auto-replies (system user / AI assistant) use the inbox avatar.
  if (isAutoReply(message)) {
    return widgetStore.config.avatar_url || widgetStore.config.launcher?.logo_url || ''
  }
  // Real human agent.
  return message.author?.avatar_url || ''
}

const getMessageAvatarFallback = (message) => {
  // When a real avatar URL is present, don't render a letter fallback while the
  // image is still loading — it briefly flashes "V" on hard refresh before the
  // avatar image finishes downloading.
  if (getMessageAvatarUrl(message)) return ''
  if (isUserMessage(message)) {
    const name = message.author?.first_name || userStore.firstName || 'V'
    return name.charAt(0).toUpperCase()
  }
  if (isAutoReply(message)) {
    const name = widgetStore.config.bot_name || widgetStore.config.brand_name || 'L'
    return name.charAt(0).toUpperCase()
  }
  return (message.author?.first_name || 'A').charAt(0).toUpperCase()
}

// Inbox avatar used by the typing indicator (consistent with system/auto replies).
const inboxAvatarUrl = computed(
  () => widgetStore.config.avatar_url || widgetStore.config.launcher?.logo_url || ''
)
const inboxAvatarFallback = computed(() =>
  (widgetStore.config.bot_name || widgetStore.config.brand_name || 'L').charAt(0).toUpperCase()
)

// getMessageTime returns the absolute clock time in 24-hour format (e.g. "11:15") for display
// under chat message bubbles. Older than ~1 day adds the date prefix for disambiguation.
const getMessageTime = (timestamp) => {
  const date = new Date(timestamp)
  const oneDayMs = 24 * 60 * 60 * 1000
  if (Date.now() - date.getTime() >= oneDayMs) {
    return format(date, 'd MMM, HH:mm')
  }
  return format(date, 'HH:mm')
}

const isQuotedTextVisible = (messageUuid) => {
  return quotedTextState.value[messageUuid] || false
}

const toggleQuote = (messageUuid) => {
  quotedTextState.value[messageUuid] = !quotedTextState.value[messageUuid]
}

// handleCSATSubmitted updates the local message state when CSAT feedback is submitted.
const handleCSATSubmitted = ({ message_uuid, rating, feedback }) => {
  const currentMessage = chatStore.getCurrentConversationMessages.find(
    (m) => m.uuid === message_uuid
  )
  const updatedMeta = {
    ...currentMessage.meta,
    csat_submitted: true,
    is_csat: true
  }

  // Add submitted rating and feedback to meta if provided
  if (rating > 0) {
    updatedMeta.submitted_rating = rating
  }
  if (feedback && feedback.trim()) {
    updatedMeta.submitted_feedback = feedback.trim()
  }

  chatStore.replaceMessage(chatStore.currentConversation.uuid, message_uuid, {
    ...currentMessage,
    meta: updatedMeta
  })
}

const handleScrollToBottom = () => {
  hasUserScrolled.value = false
  scrollToBottom()
}

const handleQuickReplyError = (message) => {
  emit('error', message)
}

// Debounced version for tab-switch and widget-open triggers only.
// New message and conversation switch call the store function directly.
const debouncedUpdateLastSeen = useDebounceFn(() => {
  // Make sure widget is open and there's a convo loaded.
  if (widgetStore.isOpen && !document.hidden && chatStore.currentConversation?.uuid) {
    chatStore.updateCurrentConversationLastSeen()
  }
}, 2000)

const visibility = useDocumentVisibility()
watch(visibility, (state) => {
  if (state === 'visible' && widgetStore.isOpen && chatStore.currentConversation?.uuid) {
    debouncedUpdateLastSeen()
  }
})

// Conversation switch - reset scroll state and update last seen.
watch(
  () => chatStore.currentConversation?.uuid,
  (newUUID) => {
    if (!newUUID || currentConversationUUID.value === newUUID) return
    currentConversationUUID.value = newUUID
    unreadMessages.value = 0
    hasUserScrolled.value = false
    nextTick(scrollToBottom)
    if (widgetStore.isOpen && !chatStore.isLoadingConversation) {
      chatStore.updateCurrentConversationLastSeen()
    }
  },
  { immediate: true }
)

// New message arrival - update last seen for agent messages, increment unread if user scrolled up, force-stick for own messages.
watch(
  () => chatStore.getCurrentConversationMessages.length,
  (newLen, oldLen) => {
    if (oldLen === 0 && newLen > 0) {
      hasUserScrolled.value = false
      nextTick(scrollToBottom)
      return
    }
    if (!oldLen || !widgetStore.isOpen) return
    if (newLen <= oldLen) return
    const messages = chatStore.getCurrentConversationMessages
    const newMessage = messages[messages.length - 1]
    const isOwnMessage =
      newMessage.author?.type === 'contact' || newMessage.author?.type === 'visitor'

    if (!isOwnMessage && !document.hidden) {
      chatStore.updateCurrentConversationLastSeen()
    }

    if (isOwnMessage) {
      hasUserScrolled.value = false
    } else if (hasUserScrolled.value) {
      unreadMessages.value++
    }
  }
)

// Widget opening - direct_to_conversation case where messages load while widget is hidden.
watch(
  () => widgetStore.isOpen,
  (isOpen) => {
    if (isOpen && chatStore.currentConversation?.uuid) {
      chatStore.updateCurrentConversationLastSeen()
      if (!hasUserScrolled.value) scrollToBottom()
    }
  }
)
</script>
