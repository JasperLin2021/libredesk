<template>
  <div class="flex h-full w-full gap-6 p-6">
    <div class="flex flex-1 flex-col gap-5 overflow-hidden">
      <header class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold text-foreground">
            {{ $t('history.conversationRecords') }}
          </h1>
          <p class="mt-1 text-sm text-muted-foreground">
            {{ $t('history.list.total', { count: total }) }}
            <span class="text-xs">· {{ $t('history.list.recordHint') }}</span>
          </p>
        </div>

        <Button
          variant="outline"
          size="sm"
          class="shrink-0 gap-2"
          :class="filtersOpen ? 'border-primary text-primary' : ''"
          :aria-controls="filtersOpen ? 'history-conversation-filter-panel' : undefined"
          :aria-expanded="filtersOpen"
          @click="toggleFilters"
        >
          <Filter class="h-4 w-4" />
          {{ $t('history.filter.toggle') }}
        </Button>
      </header>

      <div ref="rowRef" class="flex min-h-0 flex-1 gap-0 overflow-hidden">
        <div class="flex min-w-0 flex-1 flex-col overflow-hidden pr-3" :aria-busy="loading">
          <div v-if="loading" class="flex flex-1 items-center justify-center">
            <Spinner />
          </div>

          <div v-else-if="error && !rows.length" class="m-auto space-y-4 text-center">
            <p role="alert" class="text-lg text-destructive">{{ error }}</p>
            <Button @click="fetchRows({ page: 1 })">{{ $t('globals.terms.tryAgain') }}</Button>
          </div>

          <p v-else-if="!rows.length" class="m-auto text-center text-muted-foreground">
            {{ $t('history.list.empty') }}
          </p>

          <template v-else>
            <HistoryConversationTable
              :rows="rows"
              :selected-closure-uuid="selectedClosureUuid"
              @select="openConversation"
            />
            <div v-if="hasMore" class="flex justify-center pt-3">
              <Button variant="outline" size="sm" :is-loading="loadingMore" @click="loadMore">
                {{ $t('globals.terms.loadMore') }}
              </Button>
            </div>
          </template>
        </div>

        <!-- Draggable splitter: only rendered while the right column is open. -->
        <div
          v-if="selectedClosureUuid || filtersOpen"
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
          <span class="block h-8 w-0.5 rounded-full bg-border transition-colors" />
        </div>

        <div
          v-if="selectedRow"
          :key="selectedRow.closure_uuid"
          :style="{ width: rightWidth + 'px' }"
          class="shrink-0"
        >
          <!-- The record's transcript is bounded by its close message. -->
          <HistoryConversationDetail
            :conversation-uuid="selectedRow.uuid"
            :until-uuid="selectedRow.closure_uuid || ''"
            :picked-up-at="selectedRow.first_reply_at || ''"
            :last-agent="selectedRow.assigned_user_name || ''"
            :closed-at="selectedRow.closed_at || ''"
            :closed-by="selectedRow.closed_by_name || ''"
            @close="closeConversation"
          />
        </div>

        <div
          v-else-if="filtersOpen"
          id="history-conversation-filter-panel"
          :style="{ width: rightWidth + 'px' }"
          class="shrink-0"
        >
          <HistoryConversationFilters v-model="filters" @apply="onFiltersApplied" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Filter } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import HistoryConversationDetail from '@/features/history/HistoryConversationDetail.vue'
import HistoryConversationFilters from '@/features/history/HistoryConversationFilters.vue'
import HistoryConversationTable from '@/features/history/HistoryConversationTable.vue'
import { buildConversationsParams } from '@/features/history/searchParams'

const PAGE_SIZE = 30

const RIGHT_WIDTH_MIN = 360
const RIGHT_WIDTH_MAX = 960
const RIGHT_WIDTH_DEFAULT = 560

const emitter = useEmitter()
const { t } = useI18n()

const filters = ref({
  sort: 'newest',
  // Every record is a close event, so the range/ordering default to the close time.
  timeDimension: 'closed',
  period: 'this_week',
  start: '',
  end: '',
  visitorId: ''
})
const activeFilters = ref({ ...filters.value })

const filtersOpen = ref(false)
const rows = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const error = ref(null)
// uuid of the closure message of the currently previewed record.
const selectedClosureUuid = ref('')
const selectedRow = ref(null)
const rightWidth = ref(RIGHT_WIDTH_DEFAULT)

// Guards against out of order responses when the filters change quickly.
let requestId = 0

const hasMore = computed(() => rows.value.length < total.value)

function showError (message) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: message })
}

async function fetchRows ({ page: targetPage = 1, append = false } = {}) {
  const currentRequest = ++requestId
  if (append) loadingMore.value = true
  else loading.value = true
  error.value = null

  try {
    const params = buildConversationsParams({
      filters: activeFilters.value,
      page: targetPage,
      pageSize: PAGE_SIZE
    })
    const resp = await api.getHistoryConversations(params)
    if (currentRequest !== requestId) return

    const payload = resp?.data?.data || {}
    const results = payload.results || []
    rows.value = append ? [...rows.value, ...results] : results
    total.value = typeof payload.total === 'number' ? payload.total : results.length
    page.value = targetPage
  } catch (err) {
    if (currentRequest !== requestId) return
    const message =
      err?.response?.data?.message || err?.message || t('history.list.loadError')
    error.value = message
    showError(message)
  } finally {
    if (currentRequest === requestId) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

function loadMore () {
  if (loading.value || loadingMore.value) return
  fetchRows({ page: page.value + 1, append: true })
}

function onFiltersApplied (next) {
  activeFilters.value = { ...next }
  // A new filter set invalidates the current preview.
  closeConversation()
  fetchRows({ page: 1 })
}

// Open the inline preview panel instead of navigating to the conversation page.
// Each row is a close event, so its closure message uuid identifies the record.
function openConversation (row) {
  if (!row?.uuid || !row?.closure_uuid) return
  filtersOpen.value = false
  selectedRow.value = row
  selectedClosureUuid.value = row.closure_uuid
}

function closeConversation () {
  selectedClosureUuid.value = ''
  selectedRow.value = null
}

function toggleFilters () {
  // The right column shows either the preview panel or the filter panel.
  closeConversation()
  filtersOpen.value = !filtersOpen.value
}

// ── Resizable splitter ──────────────────────────────────────────────
// The record list keeps at least this much room, so the chat panel can grow
// without swallowing the list entirely.
const LIST_MIN_WIDTH = 360

// The row holding the list, the splitter and the right column.
const rowRef = ref(null)

/**
 * Maximum width the right column can reach right now, given the current row
 * width. There is no fixed cap (it used to be 960px) — the only ceiling is
 * the available row width minus LIST_MIN_WIDTH, so the user can always drag
 * the chat panel wider.
 */
const maxRightWidth = computed(() => {
  const rowWidth = rowRef.value?.getBoundingClientRect().width || 0
  if (rowWidth <= 0) return RIGHT_WIDTH_MAX
  return Math.max(RIGHT_WIDTH_MIN, rowWidth - LIST_MIN_WIDTH)
})

/**
 * Clamps a requested width of the right column to the valid range, so the
 * chat panel never gets smaller than the usable minimum and never squeezes
 * the record list below LIST_MIN_WIDTH.
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
  // the chat panel, dragging it to the RIGHT narrows it.
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

// Keyboard equivalent of the mouse drag so the panel is resizable without a pointer.
const nudgeSplitter = (delta) => {
  rightWidth.value = clampRightWidth(rightWidth.value + delta)
}

// Keep the right column within bounds when the viewport changes.
const onWindowResize = () => {
  rightWidth.value = clampRightWidth(rightWidth.value)
}
// ────────────────────────────────────────────────────────────────────

onMounted(() => {
  // Start at a fraction of the available width so the panel can actually be
  // dragged wider: starting at the maximum (the old behaviour) left no room to
  // grow, so dragging right looked like it did nothing or shrank the panel.
  const rowWidth = rowRef.value?.getBoundingClientRect().width || window.innerWidth
  rightWidth.value = clampRightWidth(rowWidth * 0.45)
  window.addEventListener('resize', onWindowResize)
  fetchRows({ page: 1 })
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
