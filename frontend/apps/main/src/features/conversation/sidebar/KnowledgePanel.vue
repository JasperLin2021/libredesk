<template>
  <div class="flex flex-col h-full min-h-0">
    <!-- Search header: a scope selector (all / title / content) plus a
         keyword input. Search hits the backend (paginated, SQL-side
         filtering), so both controls trigger a debounced refetch. -->
    <div class="border-b border-border px-3 py-2 flex items-center gap-2">
      <Select v-model="scope">
        <SelectTrigger class="h-8 w-[72px] shrink-0 text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all" class="text-xs">
            {{ $t('conversation.sidebar.knowledgeBase.scopeAll') }}
          </SelectItem>
          <SelectItem value="title" class="text-xs">
            {{ $t('conversation.sidebar.knowledgeBase.scopeTitle') }}
          </SelectItem>
          <SelectItem value="content" class="text-xs">
            {{ $t('conversation.sidebar.knowledgeBase.scopeContent') }}
          </SelectItem>
        </SelectContent>
      </Select>
      <div class="relative min-w-0 flex-1">
        <Search
          class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
        />
        <Input
          v-model="search"
          :placeholder="$t('conversation.sidebar.knowledgeBase.searchPlaceholder')"
          class="h-8 pl-8 text-xs"
        />
      </div>
      <!-- Category filter: opens a two-level picker dialog. The icon turns
           primary-colored while a category filter is active. -->
      <Button
        type="button"
        variant="outline"
        size="icon"
        class="h-8 w-8 shrink-0"
        :class="hasCategoryFilter ? 'border-primary text-primary' : 'text-muted-foreground'"
        :title="$t('conversation.sidebar.knowledgeBase.filter')"
        @click="filterOpen = true"
      >
        <Filter class="h-3.5 w-3.5" />
      </Button>
    </div>

    <!-- Initial loading state -->
    <div
      v-if="loading && articles.length === 0"
      class="flex flex-1 items-center justify-center text-xs text-muted-foreground"
    >
      {{ $t('globals.messages.loading') }}
    </div>

    <!-- Article list: scrollable list area plus a fixed footer for the
         "show more" button. The footer sits below the scroll area so the
         button is always pinned to the bottom of the tab with no trailing
         whitespace. Each row uses a `min-height` computed from the visible
         list area divided by PER_PAGE, so the first page fills the tab
         exactly and "show more" appends additional rows at the same
         height (the list scrolls once it overflows). -->
    <div v-else class="flex flex-1 min-h-0 flex-col">
      <div ref="listRef" class="flex flex-1 min-h-0 flex-col overflow-y-auto">
        <div
          v-for="article in articles"
          :key="article.id"
          :style="{ minHeight: articleMinHeight }"
          class="border-b border-border/60 last:border-b-0"
        >
          <!-- Clicking a row opens the detail drawer on the right; the row
               itself stays a compact summary. -->
          <div
            class="group flex h-full cursor-pointer flex-col justify-center px-3 py-2 transition-colors hover:bg-accent/60"
            :class="{ 'bg-accent/60': detailOpen && activeArticle?.id === article.id }"
            @click="openDetail(article)"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <div class="break-words text-xs font-medium">{{ article.title }}</div>
                <p class="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
                  {{ article.summary }}
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                class="h-6 shrink-0 px-2 text-xs"
                :disabled="sendingId === article.id"
                @click.stop="sendArticleContent(article)"
              >
                <Loader2 v-if="sendingId === article.id" class="h-3 w-3 animate-spin" />
                {{ $t('conversation.sidebar.knowledgeBase.sendContent') }}
              </Button>
            </div>
          </div>
        </div>

        <!-- Empty / no search results -->
        <div
          v-if="!loading && articles.length === 0"
          class="flex items-center justify-center py-8 text-xs text-muted-foreground"
        >
          {{
            hasSearch
              ? $t('conversation.sidebar.knowledgeBase.searchEmpty')
              : $t('conversation.sidebar.knowledgeBase.empty')
          }}
        </div>
      </div>

      <!-- Show more: visible while the backend still has pages left. Pinned
           to the bottom of the tab so it sits flush with the list above and
           leaves no empty space below. -->
      <div
        v-if="hasMore"
        class="flex justify-center border-t border-border/60 p-2"
      >
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-7 text-xs text-muted-foreground"
          :disabled="loadingMore"
          @click="fetchPage(page + 1, true)"
        >
          <Loader2 v-if="loadingMore" class="h-3 w-3 animate-spin" />
          {{ $t('conversation.sidebar.knowledgeBase.showMore') }}
        </Button>
      </div>
    </div>

    <!-- Category filter dialog: two-level picker (level-1 knowledge bases on
         the left, their sub-categories on the right). -->
    <KnowledgeFilterDialog
      :open="filterOpen"
      @update:open="(v) => (filterOpen = v)"
      :categories="categories"
      :parent-category-id="filterParentId"
      :category-id="filterCategoryId"
      :loading="categoriesLoading"
      @apply="applyCategoryFilter"
    />

    <!-- Detail drawer: full article with meta and a send action. -->
    <KnowledgeDetailDrawer
      :open="detailOpen"
      @update:open="(v) => (detailOpen = v)"
      :article="activeArticle"
      :content="activeContent"
      :loading="detailLoading"
      :sending="sendingId !== 0 && sendingId === activeArticle?.id"
      @send="sendArticleContent(activeArticle)"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Search, Loader2, Filter } from 'lucide-vue-next'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem
} from '@shared-ui/components/ui/select'
import KnowledgeDetailDrawer from '@/features/conversation/sidebar/KnowledgeDetailDrawer.vue'
import KnowledgeFilterDialog from '@/features/conversation/sidebar/KnowledgeFilterDialog.vue'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const emitter = useEmitter()

const PER_PAGE = 10
const DEBOUNCE_MS = 300

const scope = ref('all')
const search = ref('')
const hasSearch = computed(() => search.value.trim().length > 0)

const articles = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = computed(() => articles.value.length < total.value)

// Category filter state. `filterParentId` scopes the list to a whole level-1
// knowledge base; `filterCategoryId` narrows it further to one sub-category.
const categories = ref([])
const categoriesLoading = ref(false)
const filterOpen = ref(false)
const filterParentId = ref(null)
const filterCategoryId = ref(null)
const hasCategoryFilter = computed(
  () => filterParentId.value !== null || filterCategoryId.value !== null
)

// Detail drawer state. Article bodies are cached by id so reopening an
// article never refetches (the list endpoint only ships a summary).
const detailOpen = ref(false)
const activeArticle = ref(null)
const activeContent = ref('')
const detailLoading = ref(false)
const contentCache = reactive({})

const sendingId = ref(0)

// Per-row height: sized so the first PER_PAGE rows exactly fill the
// available list area. Subsequent "show more" pages reuse the same row
// height and overflow into the scroll container, so the visible rows
// never change shape when more content is appended.
const listRef = ref(null)
const articleMinHeight = ref('0px')
const updateArticleMinHeight = () => {
  if (!listRef.value) return
  const h = Math.max(40, Math.floor(listRef.value.clientHeight / PER_PAGE))
  articleMinHeight.value = `${h}px`
}
let listResizeObserver = null

let debounceTimer = null

const toastError = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })
}

// Fetches the full article body, reusing the cache when available.
const loadContent = async (article) => {
  if (contentCache[article.id] !== undefined) return contentCache[article.id]
  const resp = await api.getAgentKbArticle(article.id)
  const content = resp?.data?.data?.content || ''
  contentCache[article.id] = content
  return content
}

// Fetches one page from the agent-side endpoint. When `append` is true the
// page is appended to the current list ("显示更多"), otherwise it replaces it.
const fetchPage = async (p, append = false) => {
  if (append) loadingMore.value = true
  else loading.value = true
  try {
    const params = { page: p, per_page: PER_PAGE }
    const kw = search.value.trim()
    if (kw) {
      params.keyword_field = scope.value
      params.keywords = kw
    }
    if (filterParentId.value !== null) params.parent_category_id = filterParentId.value
    if (filterCategoryId.value !== null) params.category_id = filterCategoryId.value
    const resp = await api.listAgentKbArticles(params)
    const data = resp?.data?.data || {}
    total.value = data.total || 0
    page.value = data.page || p
    articles.value = append
      ? articles.value.concat(data.articles || [])
      : data.articles || []
  } catch (error) {
    toastError(error)
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

// Loads the two-level category tree once for the filter dialog.
const loadCategories = async () => {
  categoriesLoading.value = true
  try {
    const resp = await api.listAgentKbCategories()
    categories.value = resp?.data?.data || []
  } catch (error) {
    toastError(error)
  } finally {
    categoriesLoading.value = false
  }
}

// Applies the picked category filter and refetches from the first page.
const applyCategoryFilter = ({ parentCategoryId, categoryId }) => {
  filterParentId.value = parentCategoryId ?? null
  filterCategoryId.value = categoryId ?? null
  fetchPage(1)
}

// Opens the right-hand detail drawer for a row.
const openDetail = async (article) => {
  activeArticle.value = article
  detailOpen.value = true
  if (contentCache[article.id] !== undefined) {
    activeContent.value = contentCache[article.id]
    return
  }
  activeContent.value = ''
  detailLoading.value = true
  try {
    const content = await loadContent(article)
    if (activeArticle.value?.id === article.id) activeContent.value = content
  } catch (error) {
    toastError(error)
    if (activeArticle.value?.id === article.id) detailOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

// "发送正文": fetch the full article body (the list only ships a summary)
// and hand it to the reply box, which owns the actual send pipeline.
const sendArticleContent = async (article) => {
  if (!article || sendingId.value) return
  sendingId.value = article.id
  try {
    const content = await loadContent(article)
    if (!content) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: t('conversation.sidebar.knowledgeBase.sendFailed')
      })
      return
    }
    emitter.emit(EMITTER_EVENTS.KB_SEND_CONTENT, content)
  } catch (error) {
    toastError(error)
  } finally {
    sendingId.value = 0
  }
}

// Debounced keyword search: any keystroke resets to the first page after a
// short pause so fast typing doesn't spam the backend.
watch(search, () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => fetchPage(1), DEBOUNCE_MS)
})

// The scope selector only affects the query while a keyword is present.
watch(scope, () => {
  if (!hasSearch.value) return
  clearTimeout(debounceTimer)
  fetchPage(1)
})

onMounted(() => {
  updateArticleMinHeight()
  if (listRef.value && typeof ResizeObserver !== 'undefined') {
    listResizeObserver = new ResizeObserver(updateArticleMinHeight)
    listResizeObserver.observe(listRef.value)
  }
  fetchPage(1)
  loadCategories()
})

onBeforeUnmount(() => {
  clearTimeout(debounceTimer)
  detailOpen.value = false
  filterOpen.value = false
  if (listResizeObserver) {
    listResizeObserver.disconnect()
    listResizeObserver = null
  }
})
</script>
