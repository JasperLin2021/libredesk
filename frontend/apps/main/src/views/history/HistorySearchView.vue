<template>
  <div class="flex h-full w-full gap-6 p-6">
    <div class="flex flex-1 flex-col gap-5 overflow-hidden">
      <header>
        <h1 class="text-2xl font-semibold text-foreground">
          {{ $t('globals.terms.search') }}
        </h1>
      </header>

      <HistorySearchBar
        v-model="searchQuery"
        :history="historyRecords"
        :filters-open="filtersOpen"
        @search="onSearch"
        @toggle-filters="toggleFilters"
      />

      <div class="flex min-h-0 flex-1 gap-0 overflow-hidden">
        <div class="flex min-w-0 flex-1 flex-col overflow-y-auto pr-3">
          <SearchHistoryList
            v-if="!searchPerformed"
            :records="historyRecords"
            @select="applyHistoryQuery"
            @remove="removeFromHistory"
            @clear="clearHistory"
          />

          <div v-else-if="loading" class="flex flex-1 items-center justify-center">
            <Spinner />
          </div>

          <div v-else-if="error" class="mt-8 space-y-4 text-center">
            <p class="text-lg text-destructive">{{ error }}</p>
            <Button @click="runSearch">{{ $t('globals.terms.tryAgain') }}</Button>
          </div>

          <p
            v-else-if="searchPerformed && totalResults === 0"
            class="mt-8 text-center text-muted-foreground"
          >
            {{
              $t('search.noResultsForQuery', { query: searchQuery })
            }}
          </p>

          <SearchResults
            v-else
            :results="results"
            interactive
            class="h-full"
            @select="openConversation"
          />
        </div>

        <!-- Draggable splitter: only shown when the right column is open. -->
        <div
          v-if="selectedConversationUuid || filtersOpen"
          class="history-splitter flex w-1.5 shrink-0 cursor-col-resize items-center justify-center"
          role="separator"
          :aria-valuenow="rightWidth"
          aria-orientation="vertical"
          @mousedown="onSplitterPointerDown"
        >
          <span class="block h-8 w-0.5 rounded-full bg-border transition-colors group-hover:bg-primary" />
        </div>

        <div
          v-if="selectedConversationUuid"
          :key="selectedConversationUuid"
          :style="{ width: rightWidth + 'px' }"
          class="shrink-0"
        >
          <HistoryConversationDetail
            :conversation-uuid="selectedConversationUuid"
            :highlight-uuid="selectedMessageUuid"
            :match-item="selectedMessageItem"
            @close="closeConversation"
          />
        </div>

        <div
          v-else-if="filtersOpen"
          :style="{ width: rightWidth + 'px' }"
          class="shrink-0"
        >
          <HistorySearchFilters
            v-model="filters"
            @apply="onFiltersApplied"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import SearchResults from '@/features/search/SearchResults.vue'
import HistorySearchBar from '@/features/history/HistorySearchBar.vue'
import HistorySearchFilters from '@/features/history/HistorySearchFilters.vue'
import HistoryConversationDetail from '@/features/history/HistoryConversationDetail.vue'
import SearchHistoryList from '@/features/history/SearchHistoryList.vue'
import { useSearchHistory } from '@/features/history/useSearchHistory'
import { buildSearchParams, ALL_AGENTS } from '@/features/history/searchParams'

const MIN_SEARCH_LENGTH = 3

const RIGHT_WIDTH_MIN = 360
const RIGHT_WIDTH_MAX = 960
const RIGHT_WIDTH_DEFAULT = 560

const emitter = useEmitter()
const { t } = useI18n()

const { records: historyRecords, add: addToHistory, remove: removeFromHistory, clear: clearHistory } =
  useSearchHistory()

const searchQuery = ref('')
const filters = ref({
  sort: 'newest',
  period: '',
  start: '',
  end: '',
  inboxIds: [],
  agentId: ALL_AGENTS
})
const filtersOpen = ref(false)
const selectedConversationUuid = ref(null)
const selectedMessageUuid = ref('')
const selectedMessageItem = ref(null)
const rightWidth = ref(RIGHT_WIDTH_DEFAULT)

const results = ref({ conversations: [], messages: [] })
const loading = ref(false)
const error = ref(null)
const searchPerformed = ref(false)
const activeFilters = ref(filters.value)

let searchRequestId = 0

const totalResults = computed(
  () => results.value.conversations.length + results.value.messages.length
)

function showError (message) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: message })
}

function onSearch (value) {
  const q = (value ?? searchQuery.value).trim()
  if (q.length < MIN_SEARCH_LENGTH) {
    searchRequestId++
    results.value = { conversations: [], messages: [] }
    searchPerformed.value = false
    loading.value = false
    error.value = null
    showMinLengthHint()
    return
  }
  if (q.length >= MIN_SEARCH_LENGTH) addToHistory(q)
  runSearch()
}

function showMinLengthHint () {
  showError(t('search.minQueryLength', { length: MIN_SEARCH_LENGTH }))
}

function onFiltersApplied (next) {
  activeFilters.value = { ...next }
  // Always re-run the search with the new filters – even when no search has
  // been performed yet, otherwise clicking "apply" appears to do nothing.
  const q = searchQuery.value.trim()
  if (q.length < MIN_SEARCH_LENGTH) {
    showMinLengthHint()
    return
  }
  runSearch()
}

function applyHistoryQuery (q) {
  searchQuery.value = q
  onSearch(q)
}

// Open the inline preview panel instead of navigating to the conversation page.
function openConversation (payload) {
  const { conversationUuid, messageUuid, item } = payload || {}
  if (!conversationUuid) return
  selectedConversationUuid.value = conversationUuid
  // Re-key the detail component so highlight scroll resets when the target
  // message changes within the same conversation.
  selectedMessageUuid.value = messageUuid || ''
  // Keep the raw search hit around so the detail panel can fall back to
  // content matching when the API response lacks a message uuid.
  selectedMessageItem.value = item || null
}

function closeConversation () {
  selectedConversationUuid.value = null
  selectedMessageUuid.value = ''
  selectedMessageItem.value = null
}

function toggleFilters () {
  // The right column shows either the preview panel or the filter panel.
  selectedConversationUuid.value = null
  selectedMessageUuid.value = ''
  selectedMessageItem.value = null
  filtersOpen.value = !filtersOpen.value
}

// ── Resizable splitter ──────────────────────────────────────────────
let dragging = false
let dragStartX = 0
let dragStartWidth = RIGHT_WIDTH_DEFAULT

const onSplitterPointerDown = (e) => {
  dragging = true
  dragStartX = e.clientX
  dragStartWidth = rightWidth.value
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', onSplitterPointerMove)
  window.addEventListener('mouseup', onSplitterPointerUp)
}

const onSplitterPointerMove = (e) => {
  if (!dragging) return
  // Splitter sits on the right edge of the right column; dragging right grows it.
  const delta = e.clientX - dragStartX
  const next = Math.min(
    RIGHT_WIDTH_MAX,
    Math.max(RIGHT_WIDTH_MIN, dragStartWidth + delta)
  )
  rightWidth.value = next
}

const onSplitterPointerUp = () => {
  dragging = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  window.removeEventListener('mousemove', onSplitterPointerMove)
  window.removeEventListener('mouseup', onSplitterPointerUp)
}
// ────────────────────────────────────────────────────────────────────

function runSearch () {
  const q = searchQuery.value.trim()
  if (q.length < MIN_SEARCH_LENGTH) {
    showMinLengthHint()
    return
  }
  loading.value = true
  error.value = null
  searchPerformed.value = true

  const params = buildSearchParams({ query: q, filters: activeFilters.value })
  const requestId = ++searchRequestId
  Promise.all([
    api.searchConversations(params).then((r) => r?.data?.data ?? []),
    api.searchMessages(params).then((r) => r?.data?.data ?? [])
  ])
    .then(([conversations, messages]) => {
      if (requestId !== searchRequestId) return
      results.value = { conversations, messages }
    })
    .catch((err) => {
      if (requestId !== searchRequestId) return
      const message =
        err?.response?.data?.message || err?.message || 'Search failed'
      error.value = message
      showError(message)
    })
    .finally(() => {
      if (requestId === searchRequestId) loading.value = false
    })
}

// Search runs only on an explicit action (enter / search button / apply
// filters / recent-search click). The watcher just resets the result area
// when the query becomes too short to search for.
watch(searchQuery, (next) => {
  if (next.trim().length >= MIN_SEARCH_LENGTH) return
  searchRequestId++
  results.value = { conversations: [], messages: [] }
  searchPerformed.value = false
  loading.value = false
  error.value = null
})

onMounted(() => {
  // Clamp the initial width on smaller viewports so the splitter stays usable.
  if (typeof window !== 'undefined') {
    const max = Math.floor(window.innerWidth * 0.7)
    rightWidth.value = Math.min(RIGHT_WIDTH_MAX, Math.max(RIGHT_WIDTH_MIN, max))
  }
})

onBeforeUnmount(() => {
  onSplitterPointerUp()
})
</script>

<style scoped>
.history-splitter:hover {
  background-color: hsl(var(--accent) / 0.5);
}
.history-splitter:hover span {
  background-color: hsl(var(--primary));
}
</style>
