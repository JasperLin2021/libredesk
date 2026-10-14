<template>
  <div
    class="kb-suggestion-list bg-background border rounded-lg shadow-lg overflow-hidden"
    role="listbox"
    :aria-label="$t('globals.terms.reply')"
  >
    <div v-if="loading && items.length === 0" class="px-3 py-3 text-xs text-muted-foreground flex items-center gap-2">
      <Loader2 class="h-3.5 w-3.5 animate-spin" />
      {{ $t('globals.messages.loading') }}
    </div>
    <div v-else class="kb-suggestion-scroll overflow-y-auto">
      <!-- Canned responses group -->
      <template v-if="cannedItems.length">
        <div class="px-3 pt-2 pb-1 text-[11px] font-medium text-muted-foreground uppercase tracking-wide">
          {{ $t('editor.suggestions.cannedSection') }}
        </div>
        <button
          v-for="(item, index) in cannedItems"
          :key="item.key || `c-${index}`"
          type="button"
          role="option"
          :aria-selected="index === selectedIndex"
          class="kb-suggestion-item w-full px-3 py-2 flex items-start gap-3 hover:bg-muted border-b border-border/40"
          :class="{ 'bg-muted': index === selectedIndex }"
          @mousedown.prevent
          @click="$emit('select', item)"
          @mouseenter="$emit('hover', index)"
        >
          <span
            v-if="item.badge"
            class="kb-suggestion-badge shrink-0 mt-0.5 max-w-[6rem] truncate"
            :title="item.badge"
          >
            {{ item.badge }}
          </span>
          <span class="text-sm text-foreground line-clamp-2 flex-1 min-w-0 whitespace-pre-line">
            <span
              v-for="(seg, i) in highlightSegments(item.content)"
              :key="`c-${item.key}-${i}`"
            >
              <mark v-if="seg.match" class="kb-hit">{{ seg.text }}</mark>
              <template v-else>{{ seg.text }}</template>
            </span>
          </span>
        </button>
      </template>

      <!-- Knowledge base group. Indices continue after the canned group so
           the shared `selectedIndex` spans the whole flattened list. -->
      <template v-if="kbItems.length">
        <div
          class="px-3 pt-2 pb-1 text-[11px] font-medium text-muted-foreground uppercase tracking-wide"
          :class="{ 'mt-1': cannedItems.length }"
        >
          {{ $t('editor.suggestions.kbSection') }}
        </div>
        <button
          v-for="(item, index) in kbItems"
          :key="item.id || `k-${index}`"
          type="button"
          role="option"
          :aria-selected="cannedItems.length + index === selectedIndex"
          class="kb-suggestion-item w-full text-left px-3 py-2 flex flex-col gap-0.5 hover:bg-muted border-b border-border/40 last:border-b-0"
          :class="{ 'bg-muted': cannedItems.length + index === selectedIndex }"
          @mousedown.prevent
          @click="$emit('select', item)"
          @mouseenter="$emit('hover', cannedItems.length + index)"
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
      </template>

      <!-- Slim inline loading row shown while the KB search is in-flight
           but canned matches are already visible above. -->
      <div
        v-if="loading && items.length > 0"
        class="px-3 py-1.5 text-[11px] text-muted-foreground flex items-center gap-1.5 border-t border-border/40"
      >
        <Loader2 class="h-3 w-3 animate-spin" />
        {{ $t('globals.messages.loading') }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Loader2 } from 'lucide-vue-next'

const props = defineProps({
  // Unified suggestion items. `type: 'canned'` rows render `content` plus an
  // optional `badge` (置顶 / 个人 / category); `type: 'kb'` rows render
  // `title` + `summary`. Order is canned-first, kb-second, and
  // `selectedIndex` refers to that flattened order.
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

const cannedItems = computed(() => props.items.filter((i) => i.type === 'canned'))
const kbItems = computed(() => props.items.filter((i) => i.type !== 'canned'))

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
/* Small source pill for canned response rows (置顶 / 个人 / 分类名). */
.kb-suggestion-badge {
  font-size: 10px;
  line-height: 1;
  padding: 3px 5px;
  border-radius: 4px;
  background-color: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  white-space: nowrap;
}
</style>
