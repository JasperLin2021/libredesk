<template>
  <section class="space-y-3">
    <div class="flex items-center justify-between">
      <h3 class="text-sm font-medium text-foreground">
        {{ $t('history.recent.title') }}
      </h3>
      <button
        v-if="records.length"
        type="button"
        class="text-xs text-muted-foreground transition-colors hover:text-foreground"
        @click="$emit('clear')"
      >
        {{ $t('history.recent.clear') }}
      </button>
    </div>
    <ul
      v-if="records.length"
      class="space-y-1"
    >
      <li v-for="item in records" :key="item">
        <div
          class="group flex cursor-pointer items-center justify-between rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent/40 hover:text-foreground"
          @click="$emit('select', item)"
        >
          <span class="truncate">{{ item }}</span>
          <button
            type="button"
            class="rounded-md p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-background hover:text-destructive group-hover:opacity-100"
            :aria-label="$t('history.recent.remove', { query: item })"
            @click.stop="$emit('remove', item)"
          >
            <X class="h-3.5 w-3.5" />
          </button>
        </div>
      </li>
    </ul>
    <p v-else class="text-sm text-muted-foreground">
      {{ $t('history.recent.empty') }}
    </p>
  </section>
</template>

<script setup>
import { X } from 'lucide-vue-next'

defineProps({
  records: { type: Array, default: () => [] }
})
defineEmits(['select', 'remove', 'clear'])
</script>
