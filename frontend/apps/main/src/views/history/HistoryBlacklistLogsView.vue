<template>
  <div class="flex h-full w-full gap-6 p-6">
    <div class="flex flex-1 flex-col gap-5 overflow-hidden">
      <header class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold text-foreground">
            {{ $t('history.logs.title') }}
          </h1>
          <p class="mt-1 text-sm text-muted-foreground">
            {{ $t('history.logs.total', { count: total }) }}
          </p>
        </div>

        <Button
          variant="outline"
          size="sm"
          class="shrink-0 gap-2"
          :class="filtersOpen ? 'border-primary text-primary' : ''"
          :aria-controls="filtersOpen ? 'history-blacklist-log-filter-panel' : undefined"
          :aria-expanded="filtersOpen"
          @click="toggleFilters"
        >
          <Filter class="h-4 w-4" />
          {{ $t('history.filter.toggle') }}
        </Button>
      </header>

      <!-- Single tab, kept for parity with the customer page. -->
      <div class="flex items-center gap-1 rounded-lg border border-border bg-muted/40 p-1">
        <button
          type="button"
          class="rounded-md bg-background px-3 py-1.5 text-sm font-medium text-foreground shadow-sm"
          :aria-pressed="true"
        >
          {{ $t('history.logs.tab.all') }}
        </button>
      </div>

      <div class="flex min-h-0 flex-1 gap-4 overflow-hidden">
        <div class="flex min-w-0 flex-1 flex-col overflow-hidden" :aria-busy="loading">
          <div v-if="loading" class="flex flex-1 items-center justify-center">
            <Spinner />
          </div>

          <div v-else-if="error && !rows.length" class="m-auto space-y-4 text-center">
            <p role="alert" class="text-lg text-destructive">{{ error }}</p>
            <Button @click="fetchRows({ page: 1 })">{{ $t('globals.terms.tryAgain') }}</Button>
          </div>

          <p v-else-if="!rows.length" class="m-auto text-center text-muted-foreground">
            {{ $t('history.logs.empty') }}
          </p>

          <template v-else>
            <BlacklistLogTable :rows="rows" />
            <div v-if="hasMore" class="flex justify-center pt-3">
              <Button variant="outline" size="sm" :is-loading="loadingMore" @click="loadMore">
                {{ $t('globals.terms.loadMore') }}
              </Button>
            </div>
          </template>
        </div>

        <div
          v-if="filtersOpen"
          id="history-blacklist-log-filter-panel"
          class="w-80 shrink-0 overflow-hidden"
        >
          <BlacklistFilters
            v-model="filters"
            show-action
            @apply="onFiltersApplied"
            @close="filtersOpen = false"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Filter } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import BlacklistLogTable from '@/features/history/BlacklistLogTable.vue'
import BlacklistFilters from '@/features/history/BlacklistFilters.vue'
import {
  buildBlacklistLogsParams,
  defaultBlacklistFilters
} from '@/features/history/blacklistParams'

const PAGE_SIZE = 30

const emitter = useEmitter()
const { t } = useI18n()

const filters = ref(defaultBlacklistFilters())
const activeFilters = ref(defaultBlacklistFilters())
const filtersOpen = ref(false)
const rows = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const error = ref(null)

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
    const params = buildBlacklistLogsParams({
      filters: activeFilters.value,
      page: targetPage,
      pageSize: PAGE_SIZE
    })
    const resp = await api.getBlacklistLogs(params)
    if (currentRequest !== requestId) return

    const payload = resp?.data?.data || {}
    const results = payload.results || []
    rows.value = append ? [...rows.value, ...results] : results
    total.value = typeof payload.total === 'number' ? payload.total : results.length
    page.value = targetPage
  } catch (err) {
    if (currentRequest !== requestId) return
    const message = err?.response?.data?.message || err?.message || t('history.logs.loadError')
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
  activeFilters.value = { ...defaultBlacklistFilters(), ...next }
  fetchRows({ page: 1 })
}

function toggleFilters () {
  filtersOpen.value = !filtersOpen.value
}

onMounted(() => {
  fetchRows({ page: 1 })
})
</script>
