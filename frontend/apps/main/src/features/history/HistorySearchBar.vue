<template>
  <div class="flex items-center gap-2">
    <div class="relative flex-1">
      <Input
        ref="inputRef"
        v-model="query"
        :placeholder="placeholder"
        class="h-10 w-full pr-10"
        @keydown.enter.prevent="onSearch"
        @keydown.up.prevent="cycleHistory(-1)"
        @keydown.down.prevent="cycleHistory(1)"
      />
      <button
        type="button"
        class="absolute inset-y-0 right-2 flex items-center justify-center text-muted-foreground transition-colors hover:text-primary"
        :aria-label="$t('globals.terms.search')"
        @click="onSearch"
      >
        <Search class="h-4 w-4" />
      </button>
    </div>
    <Button
      type="button"
      class="h-10 px-3"
      @click="onSearch"
    >
      <Search class="mr-1 h-4 w-4" />
      <span>{{ $t('globals.terms.search') }}</span>
    </Button>
    <Button
      type="button"
      variant="outline"
      class="h-10 px-3"
      :class="filtersOpen ? 'border-primary text-primary' : ''"
      @click="$emit('toggleFilters')"
    >
      <ListFilter class="mr-1 h-4 w-4" />
      <span>{{ $t('history.filter.toggle') }}</span>
    </Button>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { Search, ListFilter } from 'lucide-vue-next'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  history: { type: Array, default: () => [] },
  filtersOpen: { type: Boolean, default: false }
})
const emit = defineEmits(['update:modelValue', 'search', 'toggleFilters'])

const query = ref(props.modelValue)
const historyIndex = ref(-1)

watch(
  () => props.modelValue,
  (v) => {
    if (v !== query.value) query.value = v
  }
)

// Mirror typing back to the parent so other actions (e.g. "apply filters")
// always operate on what the user currently sees in the box.
watch(query, (v) => {
  if (v !== props.modelValue) emit('update:modelValue', v)
})

function onSearch () {
  historyIndex.value = -1
  emit('update:modelValue', query.value)
  emit('search', query.value)
}

function cycleHistory (delta) {
  if (!props.history.length) return
  if (historyIndex.value === -1) {
    historyIndex.value = delta > 0 ? 0 : props.history.length - 1
  } else {
    historyIndex.value = (historyIndex.value + delta + props.history.length) % props.history.length
  }
  query.value = props.history[historyIndex.value]
}
</script>
