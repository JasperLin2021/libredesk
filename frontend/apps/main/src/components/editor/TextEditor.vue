<template>
  <div
    ref="editorWrapperRef"
    class="editor-wrapper h-full overflow-y-auto"
    :class="{ 'pointer-events-none': disabled }"
  >
    <BubbleMenu
      :editor="editor"
      :tippy-options="{ duration: 100 }"
      :should-show="shouldShowBubble"
      v-if="editor"
      class="bg-background p-1 box will-change-transform"
    >
      <div class="flex space-x-1 items-center">
        <DropdownMenu v-if="aiPrompts.length > 0">
          <DropdownMenuTrigger>
            <Button type="button" size="sm" variant="ghost" class="flex items-center justify-center">
              <span class="flex items-center">
                <span class="text-medium">AI</span>
                <Bot size="14" class="ml-1" />
                <ChevronDown class="w-4 h-4 ml-2" />
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              v-for="prompt in aiPrompts"
              :key="prompt.key"
              @select="emitPrompt(prompt.key)"
            >
              {{ prompt.title }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleBold().run()"
          :class="{ 'bg-secondary': editor?.isActive('bold') }"
        >
          <Bold size="14" />
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleItalic().run()"
          :class="{ 'bg-secondary': editor?.isActive('italic') }"
        >
          <Italic size="14" />
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleBulletList().run()"
          :class="{ 'bg-secondary': editor?.isActive('bulletList') }"
        >
          <List size="14" />
        </Button>

        <Button
          type="button"
          size="sm"
          variant="ghost"
          @click.prevent="editor?.chain().focus().toggleOrderedList().run()"
          :class="{ 'bg-secondary': editor?.isActive('orderedList') }"
        >
          <ListOrdered size="14" />
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          @click.prevent="openLinkModal"
          :class="{ 'bg-secondary': editor?.isActive('link') }"
        >
          <LinkIcon size="14" />
        </Button>
      </div>
    </BubbleMenu>
    <EditorContent :editor="editor" class="native-html" />

    <Dialog v-model:open="showLinkDialog">
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {{
              editor?.isActive('link')
                ? $t('editor.editLinkUrl')
                : $t('editor.addLinkUrl')
            }}
          </DialogTitle>
          <DialogDescription></DialogDescription>
        </DialogHeader>
        <form @submit.stop.prevent="setLink">
          <div class="grid gap-4 py-4">
            <Input
              v-model="linkUrl"
              type="text"
              :placeholder="$t('placeholders.enterUrl')"
              @keydown.enter.prevent="setLink"
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              @click="unsetLink"
              v-if="editor?.isActive('link')"
            >
              {{ $t('actions.removeLink') }}
            </Button>
            <Button type="submit">
              {{ $t('globals.messages.save') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Knowledge base autocomplete popup. Anchored directly above the
         reply editor at the same width via `kbPopupPos`, teleported to
         body so it can sit above other absolutely-positioned editor UI
         without being clipped. Visibility is driven by `kbVisible` which
         is toggled by the editor's onUpdate / selectionUpdate handlers
         below. -->
    <Teleport to="body">
      <div
        v-if="kbVisible && kbPopupPos"
        class="kb-suggestion-popup"
        :style="kbPopupStyle"
      >
        <KnowledgeSuggestionList
          :items="allSuggestions"
          :query="kbCurrentQuery"
          :selected-index="kbSelectedIndex"
          :loading="kbLoading"
          @select="onSuggestionPicked"
          @hover="(i) => (kbSelectedIndex = i)"
        />
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted, onMounted, nextTick, computed } from 'vue'
import { useEditor, EditorContent, BubbleMenu } from '@tiptap/vue-3'
import {
  ChevronDown,
  Bold,
  Italic,
  Bot,
  List,
  ListOrdered,
  Link as LinkIcon
} from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Input } from '@shared-ui/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import Placeholder from '@tiptap/extension-placeholder'
import ResizableImage from './extensions/ResizableImage'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Mention from '@tiptap/extension-mention'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import { useTypingIndicator } from '@shared-ui/composables'
import { useConversationStore } from '@main/stores/conversation'
import { useInlineImageUpload } from '@main/composables/useInlineImageUpload'
import { useCannedResponsesStore } from '@main/stores/cannedResponses'
import { useI18n } from 'vue-i18n'
import mentionSuggestion from './mentionSuggestion'
import KnowledgeSuggestionList from './KnowledgeSuggestionList.vue'
import api from '@main/api'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })
const showLinkDialog = ref(false)
const linkUrl = ref('')

const props = defineProps({
  placeholder: String,
  insertContent: String,
  messageType: String,
  autoFocus: {
    type: Boolean,
    default: true
  },
  aiPrompts: {
    type: Array,
    default: () => []
  },
  disabled: {
    type: Boolean,
    default: false
  },
  enableMentions: {
    type: Boolean,
    default: false
  },
  getSuggestions: {
    type: Function,
    default: null
  },
  enableInlineImages: {
    type: Boolean,
    default: false
  },
  // Knowledge base autocomplete. When true, typing a 2+ character word in
  // the editor triggers a search against the published knowledge base and
  // shows a floating suggestion list. Selecting an item fetches the full
  // article body and dispatches KB_SEND_CONTENT so the parent reply box
  // sends it through the regular pipeline.
  enableKbSuggestions: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['send', 'aiPromptSelected', 'mentionsChanged', 'filesDropped'])

const emitPrompt = (key) => emit('aiPromptSelected', key)

// Suppress the formatting bubble when an image node is selected so it
// doesn't fight with the image's own size/remove toolbar.
const shouldShowBubble = ({ editor: e, state }) => {
  const { selection } = state
  if (selection.empty) return false
  if (!e.view.hasFocus()) return false
  if (selection.node?.type?.name === 'image') return false
  return true
}

const { handlePaste, handleDrop } = useInlineImageUpload({
  getEditor: () => editor.value,
  isInlineEnabled: () => props.enableInlineImages,
  onOtherFiles: (files) => emit('filesDropped', files)
})

// Set up typing indicator
const conversationStore = useConversationStore()
const { startTyping, stopTyping } = useTypingIndicator(conversationStore.sendTyping, {
  get isPrivateMessage() { return props.messageType === 'private_note' }
}) 

// Knowledge base autocomplete state. `kbArticles` holds the latest search
// results, `kbPopupPos` is the editor-anchored coordinates used to position
// the popup directly above the editor at the same width, and `kbVisible`
// gates the popup render. `kbCurrentQuery` mirrors the latest search term
// so the dropdown can highlight the keyword inside titles and summaries.
const kbEmitter = useEmitter()
const kbArticles = ref([])
// Canned responses (常用语) matched against the current keyword. Unlike the
// KB search these are filtered in-memory from the canned responses store
// (which caches the agent's pinned / personal / global responses), so they
// can be shown instantly while the KB API call is still in flight.
const cannedStore = useCannedResponsesStore()
const { t } = useI18n()
const CANNED_MAX_RESULTS = 4
const kbLoading = ref(false)
const kbSelectedIndex = ref(0)
const kbPopupPos = ref(null)
const kbVisible = ref(false)
const kbDebounceTimer = ref(null)
const kbRequestSeq = ref(0)
const kbCurrentQuery = ref('')
const kbSending = ref(false)
const KB_MIN_QUERY_LEN = 2
const KB_DEBOUNCE_MS = 300
const KB_MAX_RESULTS = 6
const KB_POPUP_MAX_HEIGHT = 320
// Match a contiguous run of letters / digits / underscore immediately
// before the caret. The `u` flag is required for `\p{L}` to include
// Han / CJK characters, which is what makes the autocomplete usable
// for Chinese keywords (e.g. "游戏常见问题").
const KB_WORD_REGEX = /[\p{L}\p{N}_]+$/u

// The popup is rendered above the editor wrapper, sharing its width and
// left edge. Anchoring it to the wrapper (instead of the caret) means
// the dropdown does not jump around as the user keeps typing - it sits
// in a stable location directly above the reply box.
const editorWrapperRef = ref(null)

const kbPopupStyle = computed(() => {
  const pos = kbPopupPos.value
  if (!pos) return { display: 'none' }
  // Cap the height to the available space above the editor so the popup
  // never bleeds into the viewport top. When the editor sits high on the
  // page this shrinks the popup and lets it scroll instead.
  const availableHeight = Math.max(
    120,
    Math.min(KB_POPUP_MAX_HEIGHT, pos.aboveHeight - 8)
  )
  return {
    position: 'fixed',
    left: `${pos.left}px`,
    // `bottom` is the distance from the popup's bottom edge to the
    // viewport's bottom edge. Setting it to `innerHeight - editorTop`
    // places the popup's bottom flush with the editor's top.
    bottom: `${pos.bottom}px`,
    width: `${pos.width}px`,
    zIndex: 9999,
    maxHeight: `${availableHeight}px`
  }
})

// Only run the autocomplete flow when the editor is in public reply mode
// and the caller opted in via `enableKbSuggestions`. Private notes are
// excluded so an agent typing in a private note never accidentally sends
// a knowledge article to themselves.
const kbSuggestionsActive = computed(
  () => props.enableKbSuggestions && props.messageType !== 'private_note' && !props.disabled
)

// Flattened suggestion list rendered by the popup: canned matches first,
// then KB articles. `kbSelectedIndex` and the keyboard handler operate on
// this combined order.
const allSuggestions = computed(() => [...cannedMatches.value, ...kbArticles.value])

// Filter the agent's canned responses (pinned / personal / global) against
// the current keyword. Runs synchronously against the cached store, so it
// can be recomputed on every keystroke without debounce. Results are
// deduplicated by their underlying response identity - a pinned global
// response shows once with the "置顶" badge instead of reappearing inside
// its category branch.
const computeCannedMatches = (query) => {
  const q = (query || '').toLowerCase()
  const results = []
  const seen = new Set()
  const push = (item, type, responseId, badge) => {
    if (!(item.content || '').toLowerCase().includes(q)) return
    const key = `${type}-${responseId}`
    if (seen.has(key)) return
    seen.add(key)
    results.push({ type: 'canned', key, content: item.content, badge })
  }
  // Pinned rows dedup against their underlying personal / global response
  // (same `source` + id), so the pinned copy wins and the original does
  // not reappear in the personal list or its category branch.
  cannedStore.pinned.forEach((p) => {
    push(p, p.source || 'global', p.response_id ?? p.id, t('editor.suggestions.pinned'))
  })
  cannedStore.personal.forEach((p) => {
    push(p, 'personal', p.id, t('editor.suggestions.personal'))
  })
  const walk = (cats) => {
    ;(cats || []).forEach((cat) => {
      ;(cat.responses || []).forEach((r) => push(r, 'global', r.id, cat.name))
      walk(cat.children)
    })
  }
  walk(cannedStore.globalTree)
  return results.slice(0, CANNED_MAX_RESULTS)
}

// Reactive view over the canned store for the active query. Being a
// computed means a late-arriving store fetch (first page load) or an edit
// made in the canned responses sidebar refreshes the popup automatically.
const cannedMatches = computed(() => {
  if (!kbCurrentQuery.value) return []
  return computeCannedMatches(kbCurrentQuery.value)
})

const closeKbPopup = () => {
  kbVisible.value = false
  kbArticles.value = []
  kbSelectedIndex.value = 0
  kbPopupPos.value = null
  kbCurrentQuery.value = ''
  if (kbDebounceTimer.value) {
    clearTimeout(kbDebounceTimer.value)
    kbDebounceTimer.value = null
  }
  // Bump the request sequence so any in-flight response from a previous
  // keystroke is recognized as stale and dropped on arrival.
  kbRequestSeq.value++
  kbLoading.value = false
}

// Compute the position of the suggestion popup relative to the editor
// wrapper. The popup is anchored to the wrapper's top edge (its bottom
// touches the editor's top) and shares the wrapper's width and left
// position so it visually sits directly above the reply box.
const computeWrapperPopupPos = () => {
  const wrapper = editorWrapperRef.value
  if (!wrapper) return null
  const rect = wrapper.getBoundingClientRect()
  if (rect.width === 0) return null
  const viewportWidth = window.innerWidth || document.documentElement.clientWidth
  const minLeft = 8
  // The popup uses the wrapper's exact width, so no extra max-width
  // clamp is needed; just make sure we never start past the right edge.
  const maxLeft = Math.max(minLeft, viewportWidth - rect.width - 8)
  const left = Math.min(Math.max(rect.left, minLeft), maxLeft)
  const viewportHeight = window.innerHeight || document.documentElement.clientHeight
  // `bottom` is the distance from the popup's bottom edge to the
  // viewport's bottom edge; `innerHeight - rect.top` places that edge
  // exactly at the editor's top.
  const bottom = viewportHeight - rect.top
  return {
    left,
    bottom,
    width: rect.width,
    aboveHeight: rect.top
  }
}

// Extract the last "word" immediately before the caret. Returns an empty
// string when the caret sits after whitespace or the document is empty.
const extractCurrentWord = () => {
  const e = editor.value
  if (!e) return ''
  const { from } = e.state.selection
  // Use the start of the current block as the left boundary so the regex
  // only sees the paragraph under the caret. This prevents the popup from
  // showing matches from earlier blocks that the user is not actively
  // editing, and it keeps the work bounded for very long drafts.
  const blockStart = e.state.selection.$from.start()
  const blockText = e.state.doc.textBetween(blockStart, from, '\n', '\n')
  const match = blockText.match(KB_WORD_REGEX)
  return match ? match[0] : ''
}

const isMentionContext = (word) => typeof word === 'string' && word.startsWith('@')

// Issue the actual backend search. Each call captures the current
// `kbRequestSeq` value and only commits results if the sequence has not
// advanced by the time the response lands. This effectively cancels
// stale responses from earlier keystrokes without needing axios
// AbortController support inside the api wrapper.
const runKbSearch = async (query) => {
  const seq = ++kbRequestSeq.value
  kbLoading.value = true
  try {
    const resp = await api.listAgentKbArticles({
      keyword_field: 'all',
      keywords: query,
      page: 1,
      per_page: KB_MAX_RESULTS
    })
    if (seq !== kbRequestSeq.value) return // a newer keystroke has superseded us
    const data = resp?.data?.data || {}
    const articles = data.articles || []
    if (articles.length === 0) {
      // No KB matches: only close the popup when the canned matches are
      // empty too. With canned results on screen the popup stays up and
      // the inline loading row simply disappears.
      kbArticles.value = []
      if (cannedMatches.value.length === 0) {
        closeKbPopup()
      }
      return
    }
    kbArticles.value = articles
    kbSelectedIndex.value = 0
  } catch (err) {
    if (seq !== kbRequestSeq.value) return
    // Silent: do not interrupt typing with a toast for a background
    // autocomplete request. Keep any canned matches that are showing.
    kbArticles.value = []
    if (cannedMatches.value.length === 0) {
      closeKbPopup()
    }
  } finally {
    if (seq === kbRequestSeq.value) {
      kbLoading.value = false
    }
  }
}

const scheduleKbSearch = (query) => {
  // Track the active query so the dropdown can highlight matches against
  // the same text the user is currently editing.
  kbCurrentQuery.value = query
  if (kbDebounceTimer.value) {
    clearTimeout(kbDebounceTimer.value)
  }
  kbDebounceTimer.value = setTimeout(() => {
    kbDebounceTimer.value = null
    runKbSearch(query)
  }, KB_DEBOUNCE_MS)
}

// React to every edit / caret move. Decides whether to open, refresh, or
// close the popup based on the current word at the caret.
const refreshKbFromCaret = () => {
  if (!kbSuggestionsActive.value) {
    closeKbPopup()
    return
  }
  const word = extractCurrentWord()
  if (isMentionContext(word) || word.length < KB_MIN_QUERY_LEN) {
    closeKbPopup()
    return
  }
  kbPopupPos.value = computeWrapperPopupPos()
  if (!kbPopupPos.value) {
    closeKbPopup()
    return
  }
  // Canned matches come from the cached store and resolve synchronously
  // (they key off `kbCurrentQuery`, set inside scheduleKbSearch), so they
  // render immediately; the debounced KB search fills in below.
  kbSelectedIndex.value = 0
  kbVisible.value = true
  scheduleKbSearch(word)
}

// User picked an article from the list: fetch the full body and hand it
// off to the reply box's existing KB_SEND_CONTENT pipeline. The reply
// box is responsible for replacing the draft and triggering the send so
// we keep all send-side concerns (optimistic message, email / tag
// warnings, rollback) in one place.
const onKbArticlePicked = async (article) => {
  if (!article || kbSending.value) return
  closeKbPopup()
  kbSending.value = true
  try {
    const resp = await api.getAgentKbArticle(article.id)
    const content = resp?.data?.data?.content
    if (!content) {
      // An article without a body cannot be sent. Surface this as a toast
      // so the agent understands why nothing happened, instead of seeing
      // the editor silently unchanged.
      kbEmitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(new Error('empty content')).message
      })
      return
    }
    kbEmitter.emit(EMITTER_EVENTS.KB_SEND_CONTENT, content)
  } catch (err) {
    kbEmitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    kbSending.value = false
  }
}

// Dispatcher: routes a pick from the unified list to the right pipeline
// based on the item type.
const onSuggestionPicked = (item) => {
  if (!item) return
  if (item.type === 'canned') {
    onCannedPicked(item)
  } else {
    onKbArticlePicked(item)
  }
}

// Canned responses are inserted into the draft (not auto-sent) via the
// same `COPILOT_INSERT_REPLY` event the canned responses sidebar uses,
// so the agent can review / tweak the text before sending.
const onCannedPicked = (item) => {
  closeKbPopup()
  if (!item.content) return
  kbEmitter.emit(EMITTER_EVENTS.COPILOT_INSERT_REPLY, item.content)
}

// To preseve the table styling in emails, need to set the table style inline.
// Created these custom extensions to set the table style inline.
const CustomTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; border: 1px solid #dee2e6 !important; width: 100%; margin:0; table-layout: fixed; border-collapse: collapse; position:relative; border-radius: 0.25rem;'
      }
    }
  }
})

const CustomTableCell = TableCell.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; border: 1px solid #dee2e6 !important; box-sizing: border-box !important; min-width: 1em !important; padding: 6px 8px !important; vertical-align: top !important;'
      }
    }
  }
})

const CustomTableHeader = TableHeader.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; background-color: #f8f9fa !important; color: #212529 !important; font-weight: bold !important; text-align: left !important; border: 1px solid #dee2e6 !important; padding: 6px 8px !important;'
      }
    }
  }
})

// Extend Mention to include 'type' attribute for agent/team distinction
const CustomMention = Mention.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      type: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-type'),
        renderHTML: (attributes) => {
          if (!attributes.type) return {}
          return { 'data-type': attributes.type }
        }
      }
    }
  }
})

const isInternalUpdate = ref(false)
// Flipped to true right before the parent watcher calls `setContent` so the
// editor's onUpdate hook can tell a user keystroke apart from a draft
// being loaded into the editor, and avoid surfacing the KB popup for
// content the agent did not just type.
const kbIsProgrammaticUpdate = ref(false)

const buildExtensions = () => {
  const extensions = [
    StarterKit.configure(),
    ResizableImage.configure({
      HTMLAttributes: { class: 'inline-image', style: 'max-width: 100%; height: auto;' },
      allowBase64: false
    }),
    Placeholder.configure({ placeholder: () => props.placeholder }),
    Link,
    CustomTable.configure({ resizable: false }),
    TableRow,
    CustomTableCell,
    CustomTableHeader,
    // Always include mention extension - it gracefully handles missing getSuggestions
    CustomMention.configure({
      HTMLAttributes: {
        class: 'ld-mention'
      },
      suggestion: mentionSuggestion
    })
  ]

  return extensions
}

// Extract mentions from editor content
const extractMentions = () => {
  if (!editor.value) return []
  const mentions = []
  const json = editor.value.getJSON()

  const traverse = (node) => {
    if (node.type === 'mention' && node.attrs) {
      mentions.push({
        id: node.attrs.id,
        type: node.attrs.type
      })
    }
    if (node.content) {
      node.content.forEach(traverse)
    }
  }

  if (json.content) {
    json.content.forEach(traverse)
  }

  return mentions
}


const editor = useEditor({
  extensions: buildExtensions(),
  autofocus: props.autoFocus,
  content: htmlContent.value,
  editorProps: {
    attributes: { class: 'outline-none' },
    getSuggestions: props.getSuggestions,
    handlePaste,
    handleDrop,
    handleKeyDown: (view, event) => {
      // Knowledge base autocomplete keyboard handling. Runs before any
      // ProseMirror plugin so a visible popup can claim ArrowUp/Down,
      // Enter, and Escape without the editor creating a new line or the
      // mention extension reacting first.
      if (kbVisible.value && allSuggestions.value.length > 0) {
        if (event.key === 'ArrowUp') {
          kbSelectedIndex.value =
            (kbSelectedIndex.value - 1 + allSuggestions.value.length) % allSuggestions.value.length
          event.preventDefault()
          return true
        }
        if (event.key === 'ArrowDown') {
          kbSelectedIndex.value = (kbSelectedIndex.value + 1) % allSuggestions.value.length
          event.preventDefault()
          return true
        }
        // Only intercept plain Enter. Shift+Enter inserts a soft break and
        // an Enter mid-IME-composition confirms the candidate, not a new
        // line, so let both pass through to the editor.
        if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
          const picked = allSuggestions.value[kbSelectedIndex.value]
          if (picked) {
            event.preventDefault()
            onSuggestionPicked(picked)
          }
          return true
        }
        if (event.key === 'Escape') {
          event.preventDefault()
          closeKbPopup()
          return true
        }
      } else if (kbVisible.value && event.key === 'Escape') {
        // Empty / loading popup: Escape still closes it.
        event.preventDefault()
        closeKbPopup()
        return true
      }
      if (event.ctrlKey && event.key.toLowerCase() === 'b') {
        event.stopPropagation()
        return false
      }
      if (event.ctrlKey && event.key === 'Enter') {
        emit('send')
        // Stop typing when sending
        stopTyping()
        return true
      }
    }
  },
  // To update state when user types.
  onUpdate: ({ editor }) => {
    isInternalUpdate.value = true
    htmlContent.value = editor.getHTML()
    textContent.value = editor.getText()
    isInternalUpdate.value = false

    // Trigger typing indicator when user types
    startTyping()

    // Emit mentions if enabled
    if (props.enableMentions) {
      emit('mentionsChanged', extractMentions())
    }

    // Only react to user-driven updates. Programmatic `setContent` (e.g.
    // when a draft is loaded by the parent) also fires this hook but
    // must not surface the autocomplete popup for content the agent did
    // not just type.
    if (!kbIsProgrammaticUpdate.value && editor.view.hasFocus()) {
      refreshKbFromCaret()
    }
  },
  // Caret moves (including arrow keys and clicks) update the current word,
  // so re-evaluate the popup here too. Without this the popup would stay
  // anchored to the old position even after the user navigated away.
  onSelectionUpdate: ({ editor }) => {
    if (!kbIsProgrammaticUpdate.value && editor.view.hasFocus()) {
      refreshKbFromCaret()
    }
  },
  onBlur: () => {
    // Stop typing when editor loses focus
    stopTyping()
    // Defer close so a click on the popup (mousedown.prevent) is not
    // racing the blur and tearing the popup down before the click lands.
    setTimeout(() => closeKbPopup(), 120)
  }
})

watch(
  htmlContent,
  (newContent) => {
    if (!isInternalUpdate.value && editor.value && newContent !== editor.value.getHTML()) {
      // Mark the upcoming editor update as programmatic so the
      // autocomplete popup does not flash open when a draft is loaded
      // into the editor from the parent.
      kbIsProgrammaticUpdate.value = true
      editor.value.commands.setContent(newContent || '', false)
      textContent.value = editor.value.getText()
      editor.value.commands.focus()
      // onUpdate fires synchronously inside setContent, so by the time
      // we reach nextTick the editor has already consumed the flag.
      nextTick(() => {
        kbIsProgrammaticUpdate.value = false
      })
    }
  },
  { immediate: true }
)

// Insert content at cursor position when insertContent prop changes.
watch(
  () => props.insertContent,
  (val) => {
    if (val) editor.value?.commands.insertContent(val)
  }
)

// When the user switches to private note or the editor becomes disabled,
// tear down the popup so the next keystroke does not resurrect a stale
// suggestion list pointing at the wrong mode.
watch(
  [() => props.messageType, () => props.disabled, () => props.enableKbSuggestions],
  ([newType, newDisabled, newEnabled]) => {
    if (newType === 'private_note' || newDisabled || !newEnabled) {
      closeKbPopup()
    }
  }
)

// Reposition the popup as the page scrolls or the viewport resizes. The
// caret's screen coordinates change in both cases, and a popup left at
// the old coordinates would float away from the actual cursor.
const repositionKbPopup = () => {
  if (!kbVisible.value) return
  const pos = computeWrapperPopupPos()
  if (pos) {
    kbPopupPos.value = pos
  } else {
    closeKbPopup()
  }
}

let kbScrollRaf = null
const onScrollOrResize = () => {
  if (!kbVisible.value) return
  if (kbScrollRaf) cancelAnimationFrame(kbScrollRaf)
  kbScrollRaf = requestAnimationFrame(repositionKbPopup)
}

onMounted(() => {
  window.addEventListener('resize', onScrollOrResize, { passive: true })
  window.addEventListener('scroll', onScrollOrResize, { passive: true })
  // Warm the canned responses cache so the first keyword match renders
  // instantly. The store no-ops when already loaded (e.g. the canned
  // responses sidebar fetched it first).
  cannedStore.fetch()
  // Also catch scroll inside the editor's own scroll container.
  nextTick(() => {
    const dom = editor.value?.view?.dom
    if (dom) {
      dom.addEventListener('scroll', onScrollOrResize, { passive: true })
    }
  })
})

onUnmounted(() => {
  closeKbPopup()
  if (kbScrollRaf) cancelAnimationFrame(kbScrollRaf)
  window.removeEventListener('resize', onScrollOrResize)
  window.removeEventListener('scroll', onScrollOrResize)
  const dom = editor.value?.view?.dom
  if (dom) dom.removeEventListener('scroll', onScrollOrResize)
  editor.value?.destroy()
})

const openLinkModal = () => {
  if (editor.value?.isActive('link')) {
    linkUrl.value = editor.value.getAttributes('link').href
  } else {
    linkUrl.value = ''
  }
  showLinkDialog.value = true
}

const setLink = () => {
  if (linkUrl.value) {
    editor.value?.chain().focus().extendMarkRange('link').setLink({ href: linkUrl.value }).run()
  }
  showLinkDialog.value = false
}

const unsetLink = () => {
  editor.value?.chain().focus().unsetLink().run()
  showLinkDialog.value = false
}

// Expose focus method for parent components
const focus = () => {
  editor.value?.commands.focus()
}

defineExpose({ focus, extractMentions })
</script>

<style lang="scss">
// Moving placeholder to the top.
.tiptap p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #adb5bd;
  pointer-events: none;
  height: 0;
  font-size: 0.875rem;
}

// Ensure the parent div has a proper height
.editor-wrapper div[aria-expanded='false'] {
  display: flex;
  flex-direction: column;
  height: 100%;
}

// Ensure the editor content has a proper height and breaks words
.tiptap.ProseMirror {
  flex: 1;
  min-height: 70px;
  overflow-y: auto;
  word-wrap: break-word !important;
  overflow-wrap: break-word !important;
  word-break: break-word;
  white-space: pre-wrap;
  max-width: 100%;
}

.tiptap {
  // Table styling
  .tableWrapper {
    margin: 1.5rem 0;
    overflow-x: auto;
  }

  // Anchor tag styling
  a {
    color: hsl(var(--link));
    cursor: pointer;

    &:hover {
      text-decoration: underline;
    }
  }

  // Mention styling
  .ld-mention {
    background-color: hsl(var(--primary) / 0.1);
    border-radius: 0.25rem;
    padding: 0 0.25rem;
    color: hsl(var(--primary));
    font-weight: 500;
  }

  .image-resizer {
    display: inline-block;
    position: relative;
    margin: 4px 5px;

    .image-upload-placeholder {
      display: none;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 28px 32px;
      min-width: 360px;
      min-height: 220px;
      max-width: 100%;
      background: hsl(var(--muted));
      border: 1px dashed hsl(var(--border));
      border-radius: 6px;
      line-height: 1.4;
      gap: 12px;
    }

    .image-upload-placeholder-row {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 10px;
      font-size: 13px;
      color: hsl(var(--muted-foreground));
    }

    .image-upload-placeholder-name {
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      max-width: 320px;
    }

    &.uploading {
      .inline-image {
        display: none;
      }
      .image-upload-placeholder {
        display: inline-flex;
      }
    }

    .image-resize-handle {
      display: none;
      position: absolute;
      width: 12px;
      height: 12px;
      background: hsl(var(--primary));
      border: 2px solid hsl(var(--background));
      border-radius: 2px;
      z-index: 10;
      box-shadow: 0 0 0 1px hsl(var(--border));
    }

    .image-resize-handle-tl { top: -6px; left: -6px; cursor: nwse-resize; }
    .image-resize-handle-tr { top: -6px; right: -6px; cursor: nesw-resize; }
    .image-resize-handle-bl { bottom: -6px; left: -6px; cursor: nesw-resize; }
    .image-resize-handle-br { bottom: -6px; right: -6px; cursor: nwse-resize; }

    // Anchored to the image's left edge (no centering) so the toolbar
    // never extends past the image's left side and into adjacent UI when
    // the image sits near the editor's left edge.
    .image-size-toolbar {
      display: none;
      position: absolute;
      top: 4px;
      left: 0;
      background: hsl(var(--background) / 0.95);
      border: 1px solid hsl(var(--border));
      border-radius: 6px;
      padding: 2px;
      z-index: 10000;
      white-space: nowrap;
      box-shadow: 0 2px 8px hsl(var(--foreground) / 0.15);
      backdrop-filter: blur(4px);

      button {
        padding: 2px 8px;
        font-size: 11px;
        color: hsl(var(--muted-foreground));
        background: none;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        line-height: 1.6;

        &:hover {
          background: hsl(var(--accent));
          color: hsl(var(--accent-foreground));
        }
      }

      .image-toolbar-sep {
        width: 1px;
        height: 14px;
        background: hsl(var(--border));
        margin: 0 2px;
        align-self: center;
      }

      .image-toolbar-remove {
        color: hsl(var(--destructive)) !important;

        &:hover {
          background: hsl(var(--destructive) / 0.1) !important;
          color: hsl(var(--destructive)) !important;
        }
      }
    }

    &.ProseMirror-selectednode .image-resize-handle,
    &.resizing .image-resize-handle {
      display: block;
    }

    &.ProseMirror-selectednode .image-size-toolbar {
      display: flex;
    }

    &.ProseMirror-selectednode .inline-image,
    &.resizing .inline-image {
      outline: 2px solid #0066cc;
    }

    &.resizing .inline-image {
      opacity: 0.8;
    }
  }
}
</style>
