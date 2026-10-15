<template>
  <aside
    class="flex h-full w-full min-w-0 shrink-0 flex-col rounded-lg border border-border bg-background"
    :aria-label="$t('history.filter.title')"
  >
    <header class="flex items-center justify-between border-b border-border px-4 py-3">
      <h3 class="text-sm font-semibold text-foreground">
        {{ $t('history.filter.title') }}
      </h3>
      <button
        type="button"
        class="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        :aria-label="collapsed ? $t('history.filter.expand') : $t('history.filter.collapse')"
        :aria-expanded="!collapsed"
        aria-controls="history-conversation-filter-body"
        @click="collapsed = !collapsed"
      >
        <ChevronDown
          class="h-4 w-4 transition-transform"
          :class="collapsed ? '-rotate-90' : 'rotate-0'"
        />
      </button>
    </header>

    <div
      id="history-conversation-filter-body"
      v-show="!collapsed"
      class="flex-1 space-y-6 overflow-y-auto px-4 py-4"
    >
      <!-- Time sort -->
      <fieldset>
        <legend class="mb-2 block text-sm font-medium text-foreground">
          {{ $t('history.filter.sort') }}
        </legend>
        <div class="grid grid-cols-2 overflow-hidden rounded-md border border-border">
          <button
            v-for="option in sortOptions"
            :key="option.value"
            type="button"
            class="px-3 py-2 text-xs font-medium transition-colors duration-150"
            :class="
              draft.sort === option.value
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'bg-background text-muted-foreground hover:bg-accent hover:text-foreground'
            "
            :aria-pressed="draft.sort === option.value"
            @click="draft.sort = option.value"
          >
            {{ $t(option.labelKey) }}
          </button>
        </div>
      </fieldset>

      <!-- Time period: dimension + period + resolved range -->
      <section class="space-y-2">
        <span class="block text-sm font-medium text-foreground">
          {{ $t('history.filter.period') }}
        </span>

        <Select v-model="draft.timeDimension">
          <SelectTrigger :aria-label="$t('history.filter.period')">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="dim in timeDimensions" :key="dim.value" :value="dim.value">
              {{ $t(dim.labelKey) }}
            </SelectItem>
          </SelectContent>
        </Select>

        <Select v-model="draft.period">
          <SelectTrigger :aria-label="$t('history.filter.periodRange')">
            <SelectValue :placeholder="$t('history.filter.periodPlaceholder')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="option in periodOptions" :key="option.value" :value="option.value">
              {{ $t(option.labelKey) }}
            </SelectItem>
          </SelectContent>
        </Select>

        <p v-if="rangeLabel" class="px-1 text-xs text-muted-foreground">
          {{ rangeLabel }}
        </p>

        <!-- Custom range: stacked so the end datetime stays reachable at the
             narrowest panel width. -->
        <div v-if="draft.period === 'custom'" class="space-y-2 pt-1">
          <Input
            v-model="customStart"
            type="datetime-local"
            class="h-9 w-full min-w-0 text-xs"
            :aria-label="$t('history.filter.period')"
          />
          <div class="flex items-center gap-2">
            <span class="h-px flex-1 bg-border" />
            <span class="text-xs text-muted-foreground">{{ $t('history.filter.to') }}</span>
            <span class="h-px flex-1 bg-border" />
          </div>
          <Input
            v-model="customEnd"
            type="datetime-local"
            class="h-9 w-full min-w-0 text-xs"
            :aria-label="$t('history.filter.period')"
          />
        </div>
      </section>

      <!-- Visitor id -->
      <section>
        <label
          for="history-visitor-id"
          class="mb-2 block text-sm font-medium text-foreground"
        >
          {{ $t('history.filter.visitorId') }}
        </label>
        <Input
          id="history-visitor-id"
          v-model="draft.visitorId"
          type="text"
          autocomplete="off"
          class="h-9 w-full min-w-0"
          :placeholder="$t('history.filter.visitorIdPlaceholder')"
          @keyup.enter="apply"
        />
      </section>
    </div>

    <footer
      v-show="!collapsed"
      class="flex items-center justify-between border-t border-border px-4 py-3"
    >
      <button
        type="button"
        class="text-sm text-muted-foreground transition-colors hover:text-foreground"
        @click="reset"
      >
        {{ $t('history.filter.clearAll') }}
      </button>
      <Button size="sm" @click="apply">
        {{ $t('history.filter.query') }}
      </Button>
    </footer>
  </aside>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown } from 'lucide-vue-next'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import {
  HISTORY_PERIOD_OPTIONS,
  HISTORY_TIME_DIMENSIONS,
  SORT_OPTIONS
} from './searchParams'

const props = defineProps({
  modelValue: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['update:modelValue', 'apply'])

const { t } = useI18n()

const sortOptions = SORT_OPTIONS
const periodOptions = HISTORY_PERIOD_OPTIONS
const timeDimensions = HISTORY_TIME_DIMENSIONS

const draft = ref(clone(props.modelValue))
const collapsed = ref(false)
const customStart = ref(toDatetimeLocal(props.modelValue.start))
const customEnd = ref(toDatetimeLocal(props.modelValue.end))

watch(
  () => props.modelValue,
  (next) => {
    draft.value = clone(next)
    customStart.value = toDatetimeLocal(next.start)
    customEnd.value = toDatetimeLocal(next.end)
  },
  { deep: true }
)

const rangeLabel = computed(() => {
  if (draft.value.period === 'custom') {
    return formatRange(customStart.value, customEnd.value)
  }
  if (draft.value.start || draft.value.end) {
    return formatRange(draft.value.start, draft.value.end)
  }
  // Resolve named periods locally so the panel mirrors the range the backend
  // will apply, e.g. "2026/10/11 00:00:00 至 2026/10/17 23:59:59".
  const resolved = resolvePeriodRange(draft.value.period)
  if (!resolved) return ''
  return formatRange(resolved[0], resolved[1])
})

function clone (obj) {
  return {
    sort: obj?.sort ?? 'newest',
    timeDimension: obj?.timeDimension ?? 'first_reply_at',
    period: obj?.period ?? '',
    start: obj?.start ?? '',
    end: obj?.end ?? '',
    visitorId: obj?.visitorId ?? ''
  }
}

function toDatetimeLocal (raw) {
  if (!raw) return ''
  // Accept either "YYYY/MM/DD HH:mm:ss" or ISO; normalise to datetime-local.
  const normalised = String(raw).replace(/\//g, '-').replace(' ', 'T')
  return normalised.slice(0, 16)
}

function startOfDay (d) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate(), 0, 0, 0, 0)
}

function endOfDay (d) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate(), 23, 59, 59, 0)
}

/**
 * Mirrors the backend period resolution (Monday based weeks, local time) so the
 * panel can show the exact range that will be queried.
 */
function resolvePeriodRange (period) {
  const now = new Date()
  switch (period) {
    case 'today':
      return [startOfDay(now), endOfDay(now)]
    case 'yesterday': {
      const y = new Date(now)
      y.setDate(y.getDate() - 1)
      return [startOfDay(y), endOfDay(y)]
    }
    case 'this_week': {
      const weekday = now.getDay() === 0 ? 7 : now.getDay()
      const monday = new Date(now)
      monday.setDate(monday.getDate() - (weekday - 1))
      return [startOfDay(monday), endOfDay(now)]
    }
    case 'this_month':
      return [new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0), endOfDay(now)]
    case 'last_month': {
      const firstOfThisMonth = new Date(now.getFullYear(), now.getMonth(), 1)
      const lastOfPrevMonth = new Date(firstOfThisMonth)
      lastOfPrevMonth.setDate(0)
      return [
        new Date(lastOfPrevMonth.getFullYear(), lastOfPrevMonth.getMonth(), 1, 0, 0, 0, 0),
        endOfDay(lastOfPrevMonth)
      ]
    }
    default:
      return null
  }
}

function formatRange (start, end) {
  const fmt = (v) => {
    if (!v) return ''
    const d = v instanceof Date ? v : new Date(String(v).replace(/-/g, '/').replace('T', ' '))
    if (Number.isNaN(d.getTime())) return String(v)
    const pad = (n) => String(n).padStart(2, '0')
    return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }
  const a = fmt(start)
  const b = fmt(end)
  if (a && b) return `${a} ${t('history.filter.to')} ${b}`
  return a || b
}

function reset () {
  draft.value = clone({})
  customStart.value = ''
  customEnd.value = ''
  emit('update:modelValue', draft.value)
  emit('apply', draft.value)
}

function apply () {
  // Persist the resolved period/range so the request is built from a single
  // source of truth in the parent view.
  const payload = clone(draft.value)
  if (payload.period === 'custom') {
    payload.start = customStart.value
    payload.end = customEnd.value
  } else {
    payload.start = ''
    payload.end = ''
  }
  payload.visitorId = payload.visitorId.trim()
  draft.value = clone(payload)
  emit('update:modelValue', payload)
  emit('apply', payload)
}
</script>
