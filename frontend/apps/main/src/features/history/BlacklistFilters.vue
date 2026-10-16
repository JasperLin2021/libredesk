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
        :aria-label="$t('history.filter.collapse')"
        @click="$emit('close')"
      >
        <X class="h-4 w-4" />
      </button>
    </header>

    <div class="flex-1 space-y-6 overflow-y-auto px-4 py-4">
      <!-- Sort -->
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

      <!-- Operation type (operation log page only) -->
      <section v-if="showAction" class="space-y-2">
        <span class="block text-sm font-medium text-foreground">
          {{ $t('history.filter.action') }}
        </span>
        <Select v-model="draft.action">
          <SelectTrigger :aria-label="$t('history.filter.action')">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="option in actionOptions" :key="option.value" :value="option.value">
              {{ $t(option.labelKey) }}
            </SelectItem>
          </SelectContent>
        </Select>
      </section>

      <!-- Time period -->
      <section class="space-y-2">
        <span class="block text-sm font-medium text-foreground">
          {{ $t('history.filter.period') }}
        </span>
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

      <!-- Keyword -->
      <section>
        <label
          for="blacklist-filter-search"
          class="mb-2 block text-sm font-medium text-foreground"
        >
          {{ $t('history.filter.search') }}
        </label>
        <Input
          id="blacklist-filter-search"
          v-model="draft.search"
          type="text"
          autocomplete="off"
          class="h-9 w-full min-w-0"
          :placeholder="$t('history.filter.searchPlaceholder')"
          @keyup.enter="apply"
        />
      </section>
    </div>

    <footer class="flex items-center justify-between border-t border-border px-4 py-3">
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
import { ref, watch } from 'vue'
import { X } from 'lucide-vue-next'
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
  BLACKLIST_ACTION_OPTIONS,
  BLACKLIST_PERIOD_OPTIONS,
  BLACKLIST_SORT_OPTIONS,
  defaultBlacklistFilters
} from './blacklistParams'

const props = defineProps({
  modelValue: {
    type: Object,
    required: true
  },
  // Render the operation type filter (operation log page).
  showAction: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'apply', 'close'])

const sortOptions = BLACKLIST_SORT_OPTIONS
const periodOptions = BLACKLIST_PERIOD_OPTIONS
const actionOptions = BLACKLIST_ACTION_OPTIONS

const draft = ref({ ...defaultBlacklistFilters(), ...props.modelValue })
const customStart = ref(props.modelValue.start || '')
const customEnd = ref(props.modelValue.end || '')

watch(
  () => props.modelValue,
  (next) => {
    draft.value = { ...defaultBlacklistFilters(), ...next }
    customStart.value = next.start || ''
    customEnd.value = next.end || ''
  },
  { deep: true }
)

function reset () {
  draft.value = defaultBlacklistFilters()
  customStart.value = ''
  customEnd.value = ''
  emit('update:modelValue', { ...draft.value })
  emit('apply', { ...draft.value })
}

function apply () {
  const payload = { ...defaultBlacklistFilters(), ...draft.value }
  if (payload.period === 'custom') {
    payload.start = customStart.value
    payload.end = customEnd.value
  } else {
    payload.start = ''
    payload.end = ''
  }
  payload.search = (payload.search || '').trim()
  draft.value = { ...payload }
  emit('update:modelValue', payload)
  emit('apply', payload)
}
</script>
