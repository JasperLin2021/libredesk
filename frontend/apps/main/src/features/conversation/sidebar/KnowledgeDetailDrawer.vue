<template>
  <Teleport to="body">
    <Transition name="kb-drawer">
      <div
        v-if="open"
        class="fixed inset-y-0 right-0 z-[60] flex w-full max-w-[480px] flex-col border-l border-border bg-background shadow-2xl"
        role="dialog"
        aria-modal="false"
      >
        <!-- Header: title + close -->
        <div class="flex items-start justify-between gap-3 border-b border-border px-4 py-3">
          <h2 class="text-sm font-medium leading-snug">{{ article?.title }}</h2>
          <button
            type="button"
            class="shrink-0 rounded-sm text-muted-foreground opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring"
            :aria-label="$t('conversation.sidebar.knowledgeBase.close')"
            @click="close"
          >
            <X class="h-4 w-4" />
          </button>
        </div>

        <!-- Meta: updated time, author and category path -->
        <div class="space-y-1 border-b border-border px-4 py-2 text-xs text-muted-foreground">
          <div class="flex items-start justify-between gap-4">
            <span>
              {{ $t('conversation.sidebar.knowledgeBase.updatedAt') }}:
              {{ formattedUpdatedAt }}
            </span>
            <span class="shrink-0 text-right">
              {{ $t('conversation.sidebar.knowledgeBase.author') }}: {{ article?.author || '' }}
            </span>
          </div>
          <div>
            {{ $t('conversation.sidebar.knowledgeBase.category') }}: {{ categoryPath }}
          </div>
        </div>

        <!-- Full body -->
        <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
          <div
            v-if="loading"
            class="flex items-center justify-center py-8 text-xs text-muted-foreground"
          >
            {{ $t('globals.messages.loading') }}
          </div>
          <div
            v-else
            class="native-html prose prose-sm max-w-none break-words text-sm"
            v-html="content"
          ></div>
        </div>

        <!-- Footer: send the article body to the customer -->
        <div class="flex items-center justify-end border-t border-border px-4 py-2">
          <Button type="button" size="sm" class="h-7 gap-1 text-xs" :disabled="sending || loading" @click="emit('send')">
            {{ $t('conversation.sidebar.knowledgeBase.send') }}
            <Send class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { X, Send } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { format } from 'date-fns'
import { useEventListener } from '@vueuse/core'

const props = defineProps({
  open: { type: Boolean, default: false },
  article: { type: Object, default: null },
  content: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  sending: { type: Boolean, default: false }
})

const emit = defineEmits(['update:open', 'send'])

const close = () => emit('update:open', false)

// The database stores a two-level category path; mirror the design's
// "一级 -> 二级" notation.
const categoryPath = computed(() => {
  const a = props.article
  if (!a) return ''
  return [a.parent_category_name, a.category_name].filter(Boolean).join(' -> ')
})

const formattedUpdatedAt = computed(() => {
  const value = props.article?.updated_at
  if (!value) return ''
  try {
    return format(new Date(value), 'yyyy-MM-dd HH:mm:ss')
  } catch (error) {
    console.error('Error formatting knowledge updated_at', error, value)
    return ''
  }
})

useEventListener(window, 'keydown', (e) => {
  if (e.key === 'Escape' && props.open) close()
})
</script>

<style scoped>
.kb-drawer-enter-active,
.kb-drawer-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.kb-drawer-enter-from,
.kb-drawer-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .kb-drawer-enter-active,
  .kb-drawer-leave-active {
    transition: none;
  }
}
</style>
