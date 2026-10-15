<template>
  <div class="flex h-full w-full flex-col overflow-hidden rounded-lg border border-border bg-background">
    <!-- Header: nickname top-left, close top-right -->
    <header class="flex items-start justify-between gap-2 border-b border-border px-4 py-3">
      <div class="min-w-0">
        <h3 class="truncate text-base font-semibold text-foreground">
          {{ nickname }}
        </h3>
        <p v-if="conversation" class="truncate text-xs text-muted-foreground">
          #{{ conversation.reference_number }}
          <template v-if="conversation.status"> · {{ conversation.status }}</template>
        </p>
      </div>
      <button
        type="button"
        class="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        :aria-label="$t('history.detail.close')"
        @click="$emit('close')"
      >
        <X class="h-4 w-4" />
      </button>
    </header>

    <Spinner v-if="loading" class="m-auto" />

    <p v-else-if="error" role="alert" class="m-auto text-center text-sm text-destructive">
      {{ error }}
    </p>

    <!-- Body: chat transcript (left) + contact info (right) -->
    <div v-else class="flex min-h-0 flex-1">
      <div ref="listRef" class="flex-1 space-y-3 overflow-y-auto p-4">
        <p
          v-if="!messages.length"
          class="pt-8 text-center text-sm text-muted-foreground"
        >
          {{ $t('history.detail.noMessages') }}
        </p>
        <div
          v-for="msg in messages"
          :key="msg.uuid"
          :id="`history-msg-${msg.uuid}`"
          class="rounded-md transition-colors"
          :class="msg.uuid === resolvedHighlightUuid ? 'history-msg-highlight ring-2 ring-primary/60' : ''"
        >
          <!-- Timestamp divider -->
          <div class="flex justify-center">
            <span class="text-[11px] text-muted-foreground">{{ formatTime(msg.created_at) }}</span>
          </div>

          <!-- Closing line: the "conversation closed" message ends the record. -->
          <div v-if="msg.uuid === untilUuid" class="flex justify-center px-2">
            <span
              class="rounded-full border border-border bg-muted px-3 py-1 text-center text-[11px] text-muted-foreground"
            >
              {{ closingText(msg) }}
            </span>
          </div>

          <div v-else class="flex" :class="msg.sender_type === 'contact' ? 'justify-start' : 'justify-end'">
            <div class="max-w-[85%]">
              <div class="mb-1 text-[11px] text-muted-foreground">
                {{ senderName(msg) }}
              </div>
              <div
                class="native-html rounded-lg px-3 py-2 text-sm"
                :class="
                  msg.sender_type === 'contact'
                    ? 'bg-muted text-foreground'
                    : 'bg-primary text-primary-foreground'
                "
              >
                <div class="whitespace-pre-wrap">{{ plainContent(msg) }}</div>
                <!-- Quick-reply topic/question cards (主题与问题), rendered inside the same bubble. -->
                <div
                  v-if="msg.meta?.type === 'bot_quick_reply' && Array.isArray(msg.meta?.items) && msg.meta.items.length"
                  class="mt-2 flex flex-col items-start gap-2"
                >
                  <span
                    v-for="(item, idx) in msg.meta.items"
                    :key="idx"
                    class="w-full rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-600"
                  >
                    {{ item.label }}
                  </span>
                  <span
                    v-if="msg.meta.transfer_keyword"
                    class="inline-flex items-center gap-1.5 rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-600"
                  >
                    <Headset class="h-4 w-4" />
                    {{ msg.meta.transfer_keyword }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Contact info -->
      <aside class="w-56 shrink-0 space-y-4 overflow-y-auto border-l border-border bg-muted/30 p-4">
        <div class="flex flex-col items-center gap-2 pt-2">
          <Avatar class="h-16 w-16">
            <AvatarImage :src="contact?.avatar_url || ''" class="object-cover" />
            <AvatarFallback>
              {{ initials(contact) }}
            </AvatarFallback>
          </Avatar>
          <p class="text-center text-sm font-medium text-foreground">
            {{ contactName }}
          </p>
        </div>

        <dl class="space-y-3 text-sm">
          <div>
            <dt class="text-xs text-muted-foreground">{{ $t('globals.terms.email') }}</dt>
            <dd class="truncate text-foreground" :title="contact?.email || ''">
              {{ contact?.email || $t('conversation.sidebar.notAvailable') }}
            </dd>
          </div>
          <div>
            <dt class="text-xs text-muted-foreground">{{ $t('globals.terms.phoneNumber') }}</dt>
            <dd class="truncate text-foreground">
              {{ phoneNumber || $t('conversation.sidebar.notAvailable') }}
            </dd>
          </div>
          <div>
            <dt class="text-xs text-muted-foreground">{{ $t('contact.userId') }}</dt>
            <dd class="truncate text-foreground" :title="userId || ''">
              {{ userId || $t('conversation.sidebar.notAvailable') }}
            </dd>
          </div>
          <div v-if="lastAgentLabel">
            <dt class="text-xs text-muted-foreground">{{ $t('history.detail.lastAgent') }}</dt>
            <dd class="truncate text-foreground" :title="lastAgentLabel">
              {{ lastAgentLabel }}
            </dd>
          </div>
          <div>
            <dt class="text-xs text-muted-foreground">{{ $t('history.detail.pickedUpAt') }}</dt>
            <dd class="text-foreground">
              {{ pickedUpAtLabel }}
            </dd>
          </div>
          <div>
            <dt class="text-xs text-muted-foreground">{{ $t('history.detail.closedAt') }}</dt>
            <dd class="text-foreground">
              {{ closedAtLabel }}
            </dd>
          </div>
          <div>
            <dt class="text-xs text-muted-foreground">{{ $t('history.detail.closedBy') }}</dt>
            <dd class="truncate text-foreground" :title="closedByLabel">
              {{ closedByLabel }}
            </dd>
          </div>
          <div>
            <dt class="text-xs text-muted-foreground">{{ $t('globals.terms.createdAt') }}</dt>
            <dd class="text-foreground">
              {{ conversation ? formatTime(conversation.created_at) : '-' }}
            </dd>
          </div>
        </dl>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { X, Headset } from 'lucide-vue-next'
import { parseISO, format } from 'date-fns'
import api from '@main/api'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Spinner } from '@shared-ui/components/ui/spinner'

const { t } = useI18n()

// How many pages (of PAGE_SIZE messages each) we'll walk through when looking
// for the highlight target. Caps the worst-case load when a search match
// happens to be very old in a long conversation.
const MAX_PAGES = 20
const PAGE_SIZE = 50
// Cap for the transcript of a service history record (a close event), where we
// page back until the whole transcript up to the closing message is loaded.
const RECORD_MAX_PAGES = 10

const props = defineProps({
  conversationUuid: {
    type: String,
    required: true
  },
  // When set, the matching message is scrolled into view and highlighted.
  highlightUuid: {
    type: String,
    default: ''
  },
  // Raw search-hit item. Used as a fallback to locate the message by content
  // when the search API response does not carry a message uuid (older
  // backend builds).
  matchItem: {
    type: Object,
    default: null
  },
  // Optional metadata coming from the service history ("会话记录") list row.
  // The conversation payload only carries the assignee id, so the pick up time
  // and the last agent name are passed in from the list to avoid an extra call.
  pickedUpAt: {
    type: String,
    default: ''
  },
  lastAgent: {
    type: String,
    default: ''
  },
  // uuid of the "conversation closed" activity message of the service history
  // record. It bounds the transcript: only the messages up to and including it
  // are requested, so the closing message is always the last entry.
  untilUuid: {
    type: String,
    default: ''
  },
  closedAt: {
    type: String,
    default: ''
  },
  closedBy: {
    type: String,
    default: ''
  }
})
defineEmits(['close'])

const conversation = ref(null)
const messages = ref([])
const loading = ref(false)
const error = ref(null)
const listRef = ref(null)

const contact = computed(() => conversation.value?.contact || null)

const contactName = computed(() => {
  const c = contact.value || {}
  return `${c.first_name || ''} ${c.last_name || ''}`.trim() || '-'
})

const nickname = computed(() => {
  const subject = (conversation.value?.subject || '').trim()
  return subject || contactName.value
})

const phoneNumber = computed(() => {
  const c = contact.value || {}
  if (!c.phone_number) return ''
  const code = c.phone_number_country_code ? `+${c.phone_number_country_code} ` : ''
  return `${code}${c.phone_number}`
})

const userId = computed(() => {
  const c = contact.value || {}
  const ext = typeof c.external_user_id === 'string' ? c.external_user_id.trim() : ''
  if (ext) return ext
  return c.id ? String(c.id) : ''
})

const initials = (c) => {
  const name = `${c?.first_name || ''}${c?.last_name || ''}`.trim()
  return name ? name.substring(0, 2).toUpperCase() : '?'
}

const senderName = (msg) => {
  const author = msg.author || {}
  if (msg.sender_type === 'contact') return contactName.value
  return `${author.first_name || ''} ${author.last_name || ''}`.trim() || msg.sender_type
}

const plainContent = (msg) => {
  const text = msg.content || msg.text_content || ''
  // Strip basic HTML tags so email messages stay readable in the preview.
  return String(text)
    .replace(/<style[\s\S]*?<\/style>/gi, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/\s{2,}/g, ' ')
    .trim()
}

const formatTime = (value) => {
  if (!value) return ''
  try {
    return format(parseISO(value), 'MMM d, HH:mm')
  } catch (err) {
    return value
  }
}

const formatDateTime = (value) => {
  if (!value) return ''
  try {
    return format(parseISO(value), 'yyyy-MM-dd HH:mm:ss')
  } catch (err) {
    return value
  }
}

// Prefer the value handed over by the list row (instant paint) and fall back to
// the freshly loaded conversation payload.
const pickedUpAtLabel = computed(() =>
  formatDateTime(props.pickedUpAt || conversation.value?.first_reply_at || '') || '-'
)

const lastAgentLabel = computed(() => (props.lastAgent || '').trim())

// Close event metadata of the record, handed over by the service history row.
const closedAtLabel = computed(() => formatDateTime(props.closedAt || '') || '-')

const closedByLabel = computed(
  () => (props.closedBy || '').trim() || t('conversation.sidebar.notAvailable')
)

// Localized text of the closing message, e.g. "Alice 将状态更改为 已关闭".
const closingText = (msg) => plainContent(msg) || t('history.detail.conversationClosed')

const normalizeText = (s) => String(s || '').replace(/\s+/g, ' ').trim()

// Resolve which loaded message should be highlighted. Prefers the exact
// message uuid from the search hit; falls back to matching the hit's text
// content against loaded messages when the uuid is missing.
const resolvedHighlightUuid = computed(() => {
  if (props.highlightUuid && messages.value.some((m) => m.uuid === props.highlightUuid)) {
    return props.highlightUuid
  }
  const needle = normalizeText(props.matchItem?.text_content)
  if (needle) {
    const hit = messages.value.find((m) => {
      const hay = normalizeText(plainContent(m))
      return hay.includes(needle) || needle.includes(hay)
    })
    if (hit) return hit.uuid
  }
  return ''
})

// Smoothly bring the highlighted message into the viewport center. We compute
// the scroll position ourselves (scrollIntoView is unreliable inside the
// nested flex/overflow layout) and clamp to the actual scroll range – when
// the target is past the last page the unclamped value gets silently dropped
// by the browser, which is exactly why the message stayed off-screen.
const scrollToHighlight = (attempt = 0) => {
  if (!resolvedHighlightUuid.value) return
  const list = listRef.value

  // List not in the DOM yet (still rendering the Spinner branch because
  // `loading` is true) – wait and retry. This is the key fix: previously the
  // function bailed out silently here and the scroll never happened.
  if (!list) {
    if (attempt < 10) {
      setTimeout(() => scrollToHighlight(attempt + 1), 80)
    }
    return
  }

  const sel = `#history-msg-${CSS.escape(resolvedHighlightUuid.value)}`
  const el = list.querySelector(sel) || document.querySelector(sel)

  // Element not in the DOM yet (e.g. still rendering) – wait and retry.
  if (!el) {
    if (attempt < 10) {
      setTimeout(() => scrollToHighlight(attempt + 1), 80)
    }
    return
  }

  // Force a layout flush so all rects/scroll metrics reflect the freshly
  // rendered message bubbles.
  void list.offsetHeight

  const listRect = list.getBoundingClientRect()
  const elRect = el.getBoundingClientRect()
  // Distance from the top of the scroll viewport to the element's top edge.
  const elOffsetInList = elRect.top - listRect.top + list.scrollTop
  // Position that would center the element vertically in the list.
  const targetTop = elOffsetInList - (listRect.height - elRect.height) / 2
  // Clamp to the legal scroll range – the bug we're fixing: previously this
  // was missing, so targetTop > maxScroll made scrollTo a no-op and the
  // message stayed off-screen.
  const maxScroll = Math.max(0, list.scrollHeight - list.clientHeight)
  const finalTop = Math.max(0, Math.min(maxScroll, targetTop))

  if (import.meta.env.DEV) {
    // Diagnostic – uncomment while debugging.
    console.log('[history] scrollToHighlight', {
      attempt,
      elOffsetInList,
      listClientHeight: listRect.height,
      elHeight: elRect.height,
      scrollHeight: list.scrollHeight,
      clientHeight: list.clientHeight,
      maxScroll,
      currentScrollTop: list.scrollTop,
      finalTop
    })
  }

  if (Math.abs(list.scrollTop - finalTop) > 1) {
    list.scrollTo({ top: finalTop, behavior: attempt === 0 ? 'smooth' : 'auto' })
    // Belt-and-suspenders: directly assign scrollTop so even browsers that
    // ignore the object form of scrollTo still move the viewport.
    list.scrollTop = finalTop
  }

  // Extra safety net: ask the browser to scroll the element into view.
  if (attempt === 0 && typeof el.scrollIntoView === 'function') {
    try {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    } catch (e) {
      /* noop */
    }
  }

  // Re-apply after a short delay to absorb late layout shifts (fonts,
  // avatars, message bubble wrap recalculation, etc.).
  const delays = [200, 500, 1000]
  if (attempt < delays.length) {
    setTimeout(() => scrollToHighlight(attempt + 1), delays[attempt])
  }
}

const loadMessages = async () => {
  // Walk newest → oldest page by page. As soon as the target message shows up
  // we also pull the next page for context, then stop. If there is no target
  // (user opened a conversation from the "conversations" tab) we only load
  // the first page.
  const collected = []
  let total = Infinity
  const targetUuid = props.highlightUuid || ''
  const needle = normalizeText(props.matchItem?.text_content)
  const hasTarget = !!targetUuid || !!needle
  // A service history record ("会话记录") is bounded by its closing message and
  // shows the whole transcript that belongs to that close event.
  const until = props.untilUuid || ''
  const maxPages = until ? RECORD_MAX_PAGES : MAX_PAGES
  const pageHasHit = (results) =>
    results.some((m) => {
      if (targetUuid && m.uuid === targetUuid) return true
      if (!targetUuid && needle) {
        const hay = normalizeText(plainContent(m))
        return hay.includes(needle) || needle.includes(hay)
      }
      return false
    })

  for (let page = 1; page <= maxPages; page++) {
    const resp = await api.getConversationMessages(props.conversationUuid, {
      page,
      page_size: PAGE_SIZE,
      until: until || undefined
    })
    const payload = resp?.data?.data || {}
    const pageResults = payload.results || []
    if (page === 1 && typeof payload.total === 'number') {
      total = payload.total
    }
    collected.push(...pageResults)

    if (hasTarget) {
      if (pageHasHit(pageResults)) {
        // Pull the page after for context so the hit isn't isolated at the end.
        if (page < maxPages) {
          const next = await api.getConversationMessages(props.conversationUuid, {
            page: page + 1,
            page_size: PAGE_SIZE,
            until: until || undefined
          })
          collected.push(...((next?.data?.data?.results) || []))
        }
        break
      }
    } else if (!until) {
      // Plain conversation preview: only the newest page is needed.
      break
    }

    if (pageResults.length < PAGE_SIZE) break
    if (collected.length >= total) break
  }
  return collected
}

const fetchData = async () => {
  loading.value = true
  error.value = null
  conversation.value = null
  messages.value = []
  try {
    const convResp = await api.getConversation(props.conversationUuid)
    conversation.value = convResp?.data?.data || null
    const raw = await loadMessages()
    // Activity messages stay out of the transcript, with one exception: the
    // "conversation closed" message of the record, which is rendered as the
    // closing line and must therefore be the last entry.
    const closing = props.untilUuid
      ? raw.find((m) => m.uuid === props.untilUuid)
      : null
    const transcript = raw.filter((m) => m.type !== 'activity').reverse()
    if (closing) {
      transcript.push(closing)
    } else if (props.untilUuid) {
      // Closing message not loaded (very long transcript or stale uuid): still
      // mark the end of the record with a generic closing line.
      transcript.push({
        uuid: props.untilUuid,
        type: 'activity',
        content: '',
        created_at: props.closedAt
      })
    }
    messages.value = transcript
  } catch (err) {
    error.value = err?.response?.data?.message || err?.message || 'Failed to load conversation'
  } finally {
    // CRITICAL: scroll only after `loading` is false. While loading is true
    // the template renders <Spinner/> and the listRef isn't in the DOM, so
    // any earlier scrollToHighlight() call would no-op. Waiting for the
    // post-update tick guarantees the message list (and our target id) are
    // mounted before we try to scroll to them.
    loading.value = false
    await nextTick()
    if (resolvedHighlightUuid.value) {
      scrollToHighlight()
    } else if (listRef.value) {
      listRef.value.scrollTo({ top: listRef.value.scrollHeight })
    }
  }
}

// Identifies the current target: conversation uuid + the close record (if any) +
// the preferred message (uuid when available, otherwise the search hit's text).
const targetKey = computed(
  () =>
    `${props.conversationUuid}|${props.untilUuid || ''}|${props.highlightUuid || props.matchItem?.text_content || ''}`
)

watch(
  () => targetKey.value,
  (next, prev = '') => {
    const uuid = props.conversationUuid
    if (!uuid) return
    const prevUuid = prev.split('|')[0] || ''
    if (uuid !== prevUuid) {
      fetchData()
      return
    }
    // Same conversation, different target message. If the target is already
    // in memory just scroll; otherwise re-load (and paginate) to find it.
    if (resolvedHighlightUuid.value) {
      nextTick(scrollToHighlight)
    } else {
      fetchData()
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.history-msg-highlight {
  background-color: hsl(var(--primary) / 0.08);
  scroll-margin-top: 1rem;
  scroll-margin-bottom: 1rem;
}
</style>
