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

      <div ref="rowRef" class="flex min-h-0 flex-1 gap-0 overflow-hidden">
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
          class="history-splitter flex w-1.5 shrink-0 cursor-col-resize items-center justify-center focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          role="separator"
          tabindex="0"
          aria-orientation="vertical"
          :aria-label="$t('history.filter.resize')"
          :aria-valuemin="RIGHT_WIDTH_MIN"
          :aria-valuemax="maxRightWidth"
          :aria-valuenow="rightWidth"
          @mousedown="onSplitterPointerDown"
          @keydown.left.prevent="nudgeSplitter(24)"
          @keydown.right.prevent="nudgeSplitter(-24)"
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
// The results column keeps at least this much room, so the right column can
// grow without swallowing it entirely.
const LIST_MIN_WIDTH = 360

// The row holding the results, the splitter and the right column.
const rowRef = ref(null)

/**
 * Maximum width the right column can reach right now, given the current row
 * width. There is no fixed cap (it used to be 960px) — the only ceiling is
 * the available row width minus LIST_MIN_WIDTH, so the user can always drag
 * the right column wider.
 */
const maxRightWidth = computed(() => {
  const rowWidth = rowRef.value?.getBoundingClientRect().width || 0
  if (rowWidth <= 0) return RIGHT_WIDTH_MAX
  return Math.max(RIGHT_WIDTH_MIN, rowWidth - LIST_MIN_WIDTH)
})

/**
 * Clamps a requested width of the right column to the valid range, so the
 * column never gets smaller than the usable minimum and never squeezes the
 * results column below LIST_MIN_WIDTH.
 */
function clampRightWidth (width) {
  return Math.min(maxRightWidth.value, Math.max(RIGHT_WIDTH_MIN, Math.round(width)))
}

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
  // The splitter is the LEFT edge of the right column, so the column width is
  // measured from the right edge of the row. Solving width = rowRight - clientX
  // keeps the splitter exactly under the cursor: dragging it to the LEFT widens
  // the right column, dragging it to the RIGHT narrows it.
  const rowRight = rowRef.value?.getBoundingClientRect().right
  rightWidth.value = rowRight
    ? clampRightWidth(rowRight - e.clientX)
    : clampRightWidth(dragStartWidth - (e.clientX - dragStartX))
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

// Keyboard equivalent of the mouse drag so the column is resizable without a pointer.
const nudgeSplitter = (delta) => {
  rightWidth.value = clampRightWidth(rightWidth.value + delta)
}

// Keep the right column within bounds when the viewport changes.
const onWindowResize = () => {
  rightWidth.value = clampRightWidth(rightWidth.value)
}

onMounted(() => {
  // Start at a fraction of the available width so the column can actually be
  // dragged wider instead of starting pinned at its maximum.
  const rowWidth = rowRef.value?.getBoundingClientRect().width || window.innerWidth
  rightWidth.value = clampRightWidth(rowWidth * 0.45)
  window.addEventListener('resize', onWindowResize)
})

onBeforeUnmount(() => {
  onSplitterPointerUp()
  window.removeEventListener('resize', onWindowResize)
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
