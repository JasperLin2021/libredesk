<template>
  <aside
    class="flex h-full w-full min-w-0 shrink-0 flex-col rounded-lg border border-border bg-background"
    aria-label="filter panel"
  >
    <header class="flex items-center justify-between border-b border-border px-4 py-3">
      <h3 class="text-sm font-semibold text-foreground">
        {{ $t('history.filter.title') }}
      </h3>
      <button
        type="button"
        class="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        :aria-label="collapsed ? $t('history.filter.expand') : $t('history.filter.collapse')"
        @click="collapsed = !collapsed"
      >
        <ChevronDown
          class="h-4 w-4 transition-transform"
          :class="collapsed ? '-rotate-90' : 'rotate-0'"
        />
      </button>
    </header>

    <div v-show="!collapsed" class="flex-1 space-y-5 overflow-y-auto px-4 py-4">
      <!-- Time sort -->
      <section>
        <label class="mb-2 block text-sm font-medium text-foreground">
          {{ $t('history.filter.sort') }}
        </label>
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="option in sortOptions"
            :key="option.value"
            type="button"
            class="rounded-md border px-3 py-2 text-xs font-medium transition-colors"
            :class="
              draft.sort === option.value
                ? 'border-primary bg-primary text-primary-foreground shadow-sm'
                : 'border-border bg-background text-muted-foreground hover:border-primary/50 hover:text-foreground'
            "
            @click="draft.sort = option.value"
          >
            {{ $t(option.labelKey) }}
          </button>
        </div>
      </section>

      <!-- Time period -->
      <section>
        <label class="mb-2 block text-sm font-medium text-foreground">
          {{ $t('history.filter.period') }}
        </label>
        <Select v-model="draft.period">
          <SelectTrigger>
            <SelectValue :placeholder="$t('history.filter.periodPlaceholder')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="option in periodOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ $t(option.labelKey) }}
            </SelectItem>
          </SelectContent>
        </Select>

        <div class="mt-2 text-xs text-muted-foreground" v-if="rangeLabel">
          {{ rangeLabel }}
        </div>

        <!-- Custom range inputs: stacked so the end datetime is always
             reachable, even at the minimum panel width. -->
        <div v-if="draft.period === 'custom'" class="mt-3 space-y-2">
          <Input
            v-model="customStart"
            type="datetime-local"
            class="h-9 w-full min-w-0 text-xs"
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
          />
        </div>
      </section>

      <!-- Inboxes (multi-select) -->
      <section>
        <div class="mb-2 flex items-center justify-between">
          <label class="text-sm font-medium text-foreground">
            {{ $t('history.filter.inbox') }}
          </label>
          <button
            type="button"
            class="text-xs text-primary transition-opacity hover:opacity-80"
            @click="inboxPickerOpen = !inboxPickerOpen"
          >
            {{ inboxPickerOpen ? $t('globals.messages.close') : $t('history.filter.addInbox') }}
          </button>
        </div>

        <div v-if="selectedInboxes.length" class="space-y-2">
          <div
            v-for="inb in selectedInboxes"
            :key="inb.id"
            class="flex items-center justify-between rounded-md border border-border bg-muted/40 px-2 py-1.5"
          >
            <div class="flex min-w-0 items-center gap-2">
              <span
                class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-semibold text-primary"
              >
                {{ initials(inb.name) }}
              </span>
              <span class="truncate text-sm text-foreground">{{ inb.name }}</span>
            </div>
            <button
              type="button"
              class="rounded-md p-1 text-muted-foreground transition-colors hover:bg-background hover:text-destructive"
              :aria-label="$t('history.filter.removeInbox', { name: inb.name })"
              @click="removeInbox(inb.id)"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
        <p v-else class="text-xs text-muted-foreground">
          {{ $t('history.filter.noInbox') }}
        </p>

        <div
          v-if="inboxPickerOpen"
          class="mt-2 max-h-48 overflow-y-auto rounded-md border border-border"
        >
          <button
            v-for="inb in availableInboxes"
            :key="inb.id"
            type="button"
            class="flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors hover:bg-accent"
            :class="draft.inboxIds.includes(inb.id) ? 'bg-primary/5' : ''"
            @click="toggleInbox(inb.id)"
          >
            <div class="flex min-w-0 items-center gap-2">
              <span
                class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-muted text-[10px] font-semibold text-muted-foreground"
              >
                {{ initials(inb.name) }}
              </span>
              <span class="truncate">{{ inb.name }}</span>
            </div>
            <Check
              v-if="draft.inboxIds.includes(inb.id)"
              class="h-4 w-4 text-primary"
            />
          </button>
          <p
            v-if="!availableInboxes.length"
            class="px-3 py-2 text-xs text-muted-foreground"
          >
            {{ $t('history.filter.noInboxAvailable') }}
          </p>
        </div>
      </section>

      <!-- Agent (single select) -->
      <section>
        <label class="mb-2 block text-sm font-medium text-foreground">
          {{ $t('history.filter.agent') }}
        </label>
        <ComboBox
          v-model="draft.agentId"
          :items="agentOptions"
          :placeholder="$t('history.filter.allAgents')"
          :default-label="$t('history.filter.allAgents')"
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
        {{ $t('history.filter.clear') }}
      </button>
      <Button size="sm" @click="apply">
        {{ $t('history.filter.apply') }}
      </Button>
    </footer>
  </aside>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown, X, Check } from 'lucide-vue-next'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import { useInboxStore } from '@main/stores/inbox'
import { useUsersStore } from '@main/stores/users'
import {
  ALL_AGENTS,
  PERIOD_OPTIONS,
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
const inboxStore = useInboxStore()
const usersStore = useUsersStore()

const sortOptions = SORT_OPTIONS
const periodOptions = PERIOD_OPTIONS

const draft = ref(clone(props.modelValue))
const collapsed = ref(false)
const inboxPickerOpen = ref(false)
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

// Ensure the stores are populated so the dropdowns have data to render.
inboxStore.fetchInboxes()
usersStore.fetchUsers()

const availableInboxes = computed(() => inboxStore.inboxes || [])
const selectedInboxes = computed(() =>
  availableInboxes.value.filter((i) => draft.value.inboxIds.includes(i.id))
)

const agentOptions = computed(() => {
  const list = (usersStore.options || []).map((u) => ({
    label: u.label,
    value: String(u.value)
  }))
  return [{ label: t('history.filter.allAgents'), value: String(ALL_AGENTS) }, ...list]
})

const rangeLabel = computed(() => {
  if (draft.value.period === 'custom') {
    return formatRange(customStart.value, customEnd.value)
  }
  if (draft.value.start || draft.value.end) {
    return formatRange(draft.value.start, draft.value.end)
  }
  return ''
})

function clone (obj) {
  return {
    sort: obj?.sort ?? 'newest',
    period: obj?.period ?? '',
    start: obj?.start ?? '',
    end: obj?.end ?? '',
    inboxIds: Array.isArray(obj?.inboxIds) ? [...obj.inboxIds] : [],
    agentId: typeof obj?.agentId === 'number' ? obj.agentId : ALL_AGENTS
  }
}

function initials (name) {
  if (!name) return '?'
  const parts = String(name).trim().split(/\s+/)
  return ((parts[0]?.[0] || '') + (parts[1]?.[0] || '')).toUpperCase() || '?'
}

function toDatetimeLocal (raw) {
  if (!raw) return ''
  // Accept either "YYYY/MM/DD HH:mm:ss" or ISO; normalise to datetime-local.
  const normalised = String(raw).replace(/\//g, '-').replace(' ', 'T')
  // Trim sub-second precision to keep <input type="datetime-local"> happy.
  return normalised.slice(0, 16)
}

function formatRange (start, end) {
  const fmt = (v) => {
    if (!v) return ''
    const d = new Date(String(v).replace(/-/g, '/').replace('T', ' '))
    if (Number.isNaN(d.getTime())) return v
    const pad = (n) => String(n).padStart(2, '0')
    return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }
  const a = fmt(start)
  const b = fmt(end)
  if (a && b) return `${a} ${t('history.filter.to')} ${b}`
  return a || b
}

function toggleInbox (id) {
  const set = new Set(draft.value.inboxIds)
  if (set.has(id)) set.delete(id)
  else set.add(id)
  draft.value.inboxIds = [...set]
}

function removeInbox (id) {
  draft.value.inboxIds = draft.value.inboxIds.filter((i) => i !== id)
}

function reset () {
  draft.value = clone({})
  customStart.value = ''
  customEnd.value = ''
  emit('update:modelValue', draft.value)
  emit('apply', draft.value)
}

function apply () {
  // Persist the period value and resolved start/end to the parent so the
  // search request can be built from a single source of truth.
  const payload = clone(draft.value)
  if (payload.period === 'custom') {
    payload.start = customStart.value
    payload.end = customEnd.value
  } else {
    payload.start = ''
    payload.end = ''
  }
  payload.agentId = Number(payload.agentId) || ALL_AGENTS
  draft.value = clone(payload)
  emit('update:modelValue', payload)
  emit('apply', payload)
}
</script>
