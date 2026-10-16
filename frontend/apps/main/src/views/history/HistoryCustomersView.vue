<template>
  <div class="flex h-full w-full gap-6 p-6">
    <div class="flex flex-1 flex-col gap-5 overflow-hidden">
      <header class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold text-foreground">
            {{ $t('history.customer.title') }}
          </h1>
          <p class="mt-1 text-sm text-muted-foreground">
            {{ $t('history.customer.total', { count: total }) }}
          </p>
        </div>

        <Button
          variant="outline"
          size="sm"
          class="shrink-0 gap-2"
          :class="filtersOpen ? 'border-primary text-primary' : ''"
          :aria-controls="filtersOpen ? 'history-customer-filter-panel' : undefined"
          :aria-expanded="filtersOpen"
          @click="toggleFilters"
        >
          <Filter class="h-4 w-4" />
          {{ $t('history.filter.toggle') }}
        </Button>
      </header>

      <!-- Tabs: every customer vs. the ones currently blacklisted. -->
      <div class="flex items-center gap-2">
        <div class="flex items-center gap-1 rounded-lg border border-border bg-muted/40 p-1">
          <button
            type="button"
            class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors duration-150"
            :class="
              tab === 'all'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            "
            :aria-pressed="tab === 'all'"
            @click="switchTab('all')"
          >
            {{ $t('history.customer.tab.all') }}
          </button>
          <button
            type="button"
            class="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors duration-150"
            :class="
              tab === 'blocked'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            "
            :aria-pressed="tab === 'blocked'"
            @click="switchTab('blocked')"
          >
            {{ $t('history.customer.tab.blocked') }}
            <Badge variant="secondary" class="px-1.5 tabular-nums">
              {{ blockedCount }}
            </Badge>
          </button>
        </div>
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
            {{ $t('history.customer.empty') }}
          </p>

          <template v-else>
            <CustomerBlacklistTable
              :rows="rows"
              :columns="tab"
              :callback-contact-id="callbackContactId"
              @updated="onBlacklistUpdated"
              @callback="onCallback"
            />
            <div v-if="hasMore" class="flex justify-center pt-3">
              <Button variant="outline" size="sm" :is-loading="loadingMore" @click="loadMore">
                {{ $t('globals.terms.loadMore') }}
              </Button>
            </div>
          </template>
        </div>

        <div
          v-if="filtersOpen"
          id="history-customer-filter-panel"
          class="w-80 shrink-0 overflow-hidden"
        >
          <BlacklistFilters v-model="filters" @apply="onFiltersApplied" @close="filtersOpen = false" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Filter } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Spinner } from '@shared-ui/components/ui/spinner'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import CustomerBlacklistTable from '@/features/history/CustomerBlacklistTable.vue'
import BlacklistFilters from '@/features/history/BlacklistFilters.vue'
import {
  buildBlacklistCustomersParams,
  defaultBlacklistFilters
} from '@/features/history/blacklistParams'

const PAGE_SIZE = 30

const emitter = useEmitter()
const router = useRouter()
const { t } = useI18n()

const filters = ref(defaultBlacklistFilters())
const activeFilters = ref(defaultBlacklistFilters())
const filtersOpen = ref(false)
const rows = ref([])
const total = ref(0)
const blockedCount = ref(0)
const page = ref(1)
const tab = ref('all')
const loading = ref(false)
const loadingMore = ref(false)
const error = ref(null)
// Contact whose "回呼" (callback) request is in flight.
const callbackContactId = ref(0)

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
    const params = buildBlacklistCustomersParams({
      filters: activeFilters.value,
      tab: tab.value,
      page: targetPage,
      pageSize: PAGE_SIZE
    })
    const resp = await api.getBlacklistCustomers(params)
    if (currentRequest !== requestId) return

    const payload = resp?.data?.data || {}
    const results = payload.results || []
    rows.value = append ? [...rows.value, ...results] : results
    total.value = typeof payload.total === 'number' ? payload.total : results.length
    blockedCount.value = typeof payload.blocked_count === 'number' ? payload.blocked_count : 0
    page.value = targetPage
  } catch (err) {
    if (currentRequest !== requestId) return
    const message = err?.response?.data?.message || err?.message || t('history.customer.loadError')
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

function switchTab (next) {
  if (tab.value === next) return
  tab.value = next
  fetchRows({ page: 1 })
}

function onFiltersApplied (next) {
  activeFilters.value = { ...defaultBlacklistFilters(), ...next }
  fetchRows({ page: 1 })
}

function toggleFilters () {
  filtersOpen.value = !filtersOpen.value
}

// A row action changed the blacklist state: refresh so the row (and the badge)
// reflects it.
function onBlacklistUpdated () {
  fetchRows({ page: 1 })
}

// "回呼" (callback): proactively open a conversation with the customer and
// navigate to it so the agent can write the opening reply.
async function onCallback (row) {
  if (!row?.contact_id || callbackContactId.value) return
  callbackContactId.value = row.contact_id
  try {
    const resp = await api.createContactConversation(row.contact_id)
    const uuid = resp?.data?.data?.uuid
    if (!uuid) throw new Error('missing conversation uuid')
    router.push({ name: 'inbox-conversation', params: { uuid, type: 'assigned' } })
  } catch (err) {
    showError(handleHTTPError(err).message)
  } finally {
    callbackContactId.value = 0
  }
}

onMounted(() => {
  fetchRows({ page: 1 })
})
</script>
