<template>
  <div>
    <div
      v-if="!items.length"
      class="pl-8 pr-3 py-2 text-xs text-muted-foreground"
    >
      {{ searchEmpty ? searchEmptyText : emptyText }}
    </div>
    <ul v-else class="flex flex-col">
      <li
        v-for="item in items"
        :key="keyFor(item)"
        class="group flex items-center gap-2 pl-8 pr-3 py-1.5 text-xs hover:bg-accent/50 cursor-pointer"
        @click="emit('select', item)"
      >
        <span class="flex-1 truncate" :title="item.content">{{ item.content }}</span>
        <span
          v-if="actionsFor(item).length"
          class="flex shrink-0 items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
          @click.stop
        >
          <Button
            v-for="a in actionsFor(item)"
            :key="a.key"
            type="button"
            variant="ghost"
            size="icon"
            class="h-6 w-6 text-muted-foreground hover:text-foreground"
            :title="a.label"
            @click="emit('action', { action: a, item })"
          >
            <component :is="iconFor(a.icon)" class="h-3.5 w-3.5" />
            <span class="sr-only">{{ a.label }}</span>
          </Button>
        </span>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Pin, PinOff, Pencil, Trash2 } from 'lucide-vue-next'

// `CannedResponseList` is the presentational row list used for the
// top-level pinned / personal groups. Recursive categories render their
// own rows through `CannedCategoryGroup` instead.

const props = defineProps({
  items: { type: Array, required: true },
  emptyText: { type: String, default: '' },
  searchEmpty: { type: Boolean, default: false },
  searchEmptyText: { type: String, default: '' },
  // `actions` is a list of descriptors from the parent. Each descriptor
  // exposes a `match(item)` predicate that filters out actions not
  // relevant to the current row (e.g. "pin" is hidden when the response
  // is already pinned).
  actions: { type: Array, default: () => [] }
})
const emit = defineEmits(['select', 'action'])

const iconFor = (name) => {
  switch (name) {
    case 'pin':
      return Pin
    case 'pin-off':
      return PinOff
    case 'edit':
      return Pencil
    case 'trash':
      return Trash2
    default:
      return null
  }
}

const actionsFor = (item) =>
  (props.actions || []).filter((a) => !a.match || a.match(item))

// A stable key per item. Pinned entries use the (source, response_id)
// pair so the same response keeps the same key across pin/personal views.
const keyFor = (item) => {
  if (item.source && item.response_id != null) {
    return `${item.source}-${item.response_id}`
  }
  return item.id
}
</script>
