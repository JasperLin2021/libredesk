<template>
  <div v-if="items.length || transferKeyword" class="flex flex-col gap-1.5 mt-2">
    <span
      v-for="(item, index) in items"
      :key="index"
      class="inline-block w-fit px-3 py-1.5 text-xs rounded-full bg-muted text-muted-foreground border border-border"
    >
      {{ item.label }}
    </span>
    <span
      v-if="transferKeyword"
      class="inline-block w-fit px-3 py-1.5 text-xs rounded-full bg-secondary/10 text-foreground border border-secondary/40"
    >
      {{ transferKeyword }}
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  message: {
    type: Object,
    required: true
  }
})

const items = computed(() => {
  const meta = props.message.meta
  if (!meta?.items || !Array.isArray(meta.items)) return []
  return meta.items
})

const transferKeyword = computed(() => {
  return props.message?.meta?.transfer_keyword || ''
})
</script>
