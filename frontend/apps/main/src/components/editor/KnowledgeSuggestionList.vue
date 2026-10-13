<template>
  <div
    class="kb-suggestion-list bg-background border rounded-lg shadow-lg overflow-hidden"
    role="listbox"
    :aria-label="$t('globals.terms.reply')"
  >
    <div v-if="loading" class="px-3 py-3 text-xs text-muted-foreground flex items-center gap-2">
      <Loader2 class="h-3.5 w-3.5 animate-spin" />
      {{ $t('globals.messages.loading') }}
    </div>
    <div v-else class="kb-suggestion-scroll overflow-y-auto">
      <button
        v-for="(item, index) in items"
        :key="item.id"
        type="button"
        role="option"
        :aria-selected="index === selectedIndex"
        class="kb-suggestion-item w-full text-left px-3 py-2 flex flex-col gap-0.5 hover:bg-muted border-b border-border/40 last:border-b-0"
        :class="{ 'bg-muted': index === selectedIndex }"
        @mousedown.prevent
        @click="$emit('select', item)"
        @mouseenter="$emit('hover', index)"
      >
        <span class="text-sm font-medium truncate">
          <span
            v-for="(seg, i) in highlightSegments(item.title)"
            :key="`t-${item.id}-${i}`"
          >
            <mark v-if="seg.match" class="kb-hit">{{ seg.text }}</mark>
            <template v-else>{{ seg.text }}</template>
          </span>
        </span>
        <span
          v-if="item.summary"
          class="text-xs text-muted-foreground line-clamp-2"
        >
          <span
            v-for="(seg, i) in highlightSegments(item.summary)"
            :key="`s-${item.id}-${i}`"
          >
            <mark v-if="seg.match" class="kb-hit">{{ seg.text }}</mark>
            <template v-else>{{ seg.text }}</template>
          </span>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Loader2 } from 'lucide-vue-next'

const props = defineProps({
  items: {
    type: Array,
    default: () => []
  },
  query: {
    type: String,
    default: ''
  },
  selectedIndex: {
    type: Number,
    default: 0
  },
  loading: {
    type: Boolean,
    default: false
  }
})

defineEmits(['select', 'hover'])

// Pre-compute the regex once per query change. The `g` flag enables
// repeated matching for splitting, and `i` keeps the highlight
// case-insensitive so "TLBC" still matches "tlbc" in article bodies.
// Special regex characters in the query are escaped so user input
// cannot break the splitter.
const highlightRegex = computed(() => {
  if (!props.query) return null
  const escaped = props.query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return new RegExp(escaped, 'gi')
})

// Split a string into alternating plain / highlighted segments so the
// template can render each piece with the right styling. Returns the
// original text as a single non-match segment when the query is empty.
const highlightSegments = (text) => {
  if (!text) return [{ text: '', match: false }]
  const re = highlightRegex.value
  if (!re) return [{ text, match: false }]
  const segments = []
  let lastIndex = 0
  // Reset the regex's lastIndex because we are reusing the same RegExp
  // instance across multiple text inputs.
  re.lastIndex = 0
  let match
  while ((match = re.exec(text)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ text: text.slice(lastIndex, match.index), match: false })
    }
    segments.push({ text: match[0], match: true })
    lastIndex = re.lastIndex
    // Guard against zero-width matches causing an infinite loop.
    if (match.index === re.lastIndex) re.lastIndex++
  }
  if (lastIndex < text.length) {
    segments.push({ text: text.slice(lastIndex), match: false })
  }
  return segments
}
</script>

<style scoped>
.kb-suggestion-scroll {
  /* The actual cap is set inline by the parent so the popup can clamp
     itself to whatever vertical space is available above the editor.
     This fallback keeps the list readable if that inline max-height
     is ever removed. */
  max-height: 20rem;
}
.kb-suggestion-item {
  cursor: pointer;
  transition: background-color 80ms ease;
}
/* Match-highlight styling. Uses the project's primary color tokens so
   the highlight fits whatever theme (light / dark) is active. */
.kb-hit {
  background-color: hsl(var(--primary) / 0.18);
  color: hsl(var(--primary));
  padding: 0 1px;
  border-radius: 2px;
  font-weight: 600;
}
</style>
