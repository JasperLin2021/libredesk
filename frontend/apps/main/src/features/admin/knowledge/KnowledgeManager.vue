<template>
  <div class="space-y-4">
    <!-- Toolbar -->
    <Card>
      <CardContent class="flex flex-wrap items-center justify-between gap-3 p-3">
        <div class="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" @click="downloadTemplate">
            <Download class="mr-1.5 h-4 w-4" />
            {{ $t('admin.knowledgeBase.downloadTemplate') }}
          </Button>
          <Button size="sm" variant="outline" @click="openImportDialog">
            <Upload class="mr-1.5 h-4 w-4" />
            {{ $t('admin.knowledgeBase.import') }}
          </Button>
          <Button size="sm" variant="outline" @click="exportAll">
            <Download class="mr-1.5 h-4 w-4" />
            {{ $t('admin.knowledgeBase.export') }}
          </Button>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" @click="filterOpen = true">
            <SlidersHorizontal class="mr-1.5 h-4 w-4" />
            {{ $t('admin.knowledgeBase.filterSort') }}
          </Button>
          <Button size="sm" @click="openCreate">
            <Plus class="mr-1.5 h-4 w-4" />
            {{ $t('admin.knowledgeBase.addArticle') }}
          </Button>
        </div>
      </CardContent>
    </Card>

    <Spinner v-if="loading && !categories.length" />

    <div v-else class="grid gap-4 md:grid-cols-[260px_1fr]">
      <!-- Left: categories tree -->
      <Card class="self-start">
        <CardHeader class="flex-row items-center justify-between space-y-0 px-3 py-3">
          <CardTitle class="text-sm font-semibold">
            <button
              type="button"
              class="flex items-center gap-1.5 text-sm"
              @click="allCollapsed = !allCollapsed"
            >
              <ChevronRight v-if="allCollapsed" class="h-3.5 w-3.5" />
              <ChevronDown v-else class="h-3.5 w-3.5" />
              {{ $t('admin.knowledgeBase.allCategories') }}
            </button>
          </CardTitle>
        </CardHeader>
        <CardContent class="p-1 max-h-[60vh] overflow-y-auto">
          <!-- Virtual "drafts" group -->
          <button
            v-if="!allCollapsed"
            type="button"
            class="flex w-full items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': selectedFilter.draft }"
            @click="selectDraft"
          >
            <span class="truncate">{{ $t('admin.knowledgeBase.draftCategory') }}</span>
          </button>
          <div v-if="!allCollapsed" class="space-y-0.5">
            <div v-for="root in categories" :key="root.id" class="rounded">
              <button
                type="button"
                class="flex w-full items-center justify-between gap-2 rounded px-3 py-2 text-sm hover:bg-muted"
                :class="{ 'bg-primary text-primary-foreground': isRootSelected(root) }"
                @click="selectCategory(root)"
              >
                <span class="truncate">{{ root.name }}</span>
                <span
                  v-if="(root.children || []).length"
                  class="shrink-0 rounded p-0.5 hover:bg-black/10"
                  @click.stop="toggleRoot(root.id)"
                >
                  <ChevronDown v-if="isRootExpanded(root.id)" class="h-3.5 w-3.5" />
                  <ChevronRight v-else class="h-3.5 w-3.5" />
                </span>
                <ChevronRight v-else class="h-3.5 w-3.5 shrink-0 opacity-50" />
              </button>
              <div v-if="(root.children || []).length && isRootExpanded(root.id)" class="ml-3 space-y-0.5">
                <button
                  v-for="leaf in root.children"
                  :key="leaf.id"
                  type="button"
                  class="flex w-full items-center justify-between rounded px-3 py-1.5 text-sm hover:bg-muted"
                  :class="{ 'bg-primary text-primary-foreground': selectedFilter.categoryId === leaf.id }"
                  @click="selectCategory(leaf)"
                >
                  <span class="truncate">{{ leaf.name }}</span>
                </button>
              </div>
            </div>
          </div>
        </CardContent>
        <CardFooter class="px-3 py-2">
          <button
            type="button"
            class="text-sm text-primary hover:underline"
            @click="manageOpen = true"
          >
            {{ $t('admin.knowledgeBase.manageCategories') }}
          </button>
        </CardFooter>
      </Card>

      <!-- Right: article list -->
      <Card>
        <CardContent class="p-0">
          <div v-if="loading" class="p-6"><Spinner /></div>
          <div v-else-if="!articles.length" class="p-10 text-center text-sm text-muted-foreground">
            {{ $t('admin.knowledgeBase.empty') }}
          </div>
          <ul v-else class="divide-y">
            <li
              v-for="a in articles"
              :key="a.id"
              class="group flex items-start gap-3 px-4 py-3 hover:bg-muted/40"
            >
              <FileText class="mt-1 h-4 w-4 text-muted-foreground" />
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 text-sm font-medium text-foreground">
                  <span v-if="a.parent_category_name" class="text-xs text-muted-foreground">
                    【{{ a.parent_category_name }}】
                  </span>
                  <span v-if="a.category_name" class="text-xs text-muted-foreground">
                    【{{ a.category_name }}】
                  </span>
                  <span class="truncate">{{ a.title }}</span>
                  <span
                    v-if="a.status === 'draft'"
                    class="ml-1 inline-flex items-center rounded bg-amber-100 px-1.5 py-0.5 text-[10px] text-amber-700"
                  >
                    {{ $t('admin.knowledgeBase.draft') }}
                  </span>
                </div>
                <p class="text-xs text-muted-foreground line-clamp-2 mt-1">{{ a.summary }}</p>
              </div>
              <div class="flex flex-col items-end gap-1 text-xs text-muted-foreground">
                <span>{{ $t('admin.knowledgeBase.updatedAt', { time: formatTime(a.updated_at) }) }}</span>
                <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button type="button" class="rounded p-1 hover:bg-muted" @click="openPreview(a)">
                    <Eye class="h-3.5 w-3.5" />
                  </button>
                  <button type="button" class="rounded p-1 hover:bg-muted" @click="openEdit(a)">
                    <Pencil class="h-3.5 w-3.5" />
                  </button>
                  <button type="button" class="rounded p-1 hover:bg-muted text-destructive" @click="askDelete(a)">
                    <Trash2 class="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </li>
          </ul>
        </CardContent>
        <CardFooter class="flex items-center justify-between gap-2 px-3 py-3 text-xs text-muted-foreground">
          <span>{{ $t('admin.knowledgeBase.totalCount', { total: total }) }}</span>
          <div class="flex items-center gap-1">
            <Button size="sm" variant="outline" :disabled="page <= 1" @click="page = Math.max(1, page - 1)">
              <ChevronLeft class="h-3.5 w-3.5" />
            </Button>
            <span class="px-2">{{ page }} / {{ totalPages || 1 }}</span>
            <Button size="sm" variant="outline" :disabled="page >= totalPages" @click="page = Math.min(totalPages, page + 1)">
              <ChevronRight class="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardFooter>
      </Card>
    </div>

    <KnowledgeFormDialog
      v-model:open="formOpen"
      :article="editing"
      :categories="categories"
      @saved="onSaved"
      @preview="openPreviewFromForm"
    />
    <KnowledgeFilterDialog
      v-model:open="filterOpen"
      :initial="filterInitial"
      @apply="onApplyFilter"
    />
    <ManageCategoriesDialog
      v-model:open="manageOpen"
      :categories="categories"
      @changed="loadCategories"
    />
    <KnowledgePreviewDialog
      v-model:open="previewOpen"
      :title="preview.title"
      :content="preview.content"
      :meta="preview.meta"
    />

    <!-- Import dialog -->
    <Dialog v-model:open="importOpen">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t('admin.knowledgeBase.importTitle') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-3">
          <p class="text-xs text-muted-foreground">{{ $t('admin.knowledgeBase.importDescription') }}</p>
          <div
            class="rounded-md border-2 border-dashed p-6 text-center cursor-pointer hover:bg-muted/30"
            @click="triggerImportFile"
            @dragover.prevent
            @drop.prevent="onImportDrop"
          >
            <Upload class="mx-auto h-6 w-6 text-muted-foreground" />
            <p class="mt-2 text-sm">{{ $t('admin.knowledgeBase.importDropzone') }}</p>
            <p class="text-xs text-muted-foreground mt-1">{{ $t('admin.knowledgeBase.importFileHint') }}</p>
            <input ref="importFileInput" type="file" class="hidden" accept=".xlsx,.xls" @change="onImportPick" />
          </div>
          <div v-if="importPreview.length" class="text-xs text-muted-foreground">
            已解析 {{ importPreview.length }} 条记录，前 5 条预览：
            <ul class="mt-1 list-disc pl-4">
              <li v-for="(row, i) in importPreview.slice(0, 5)" :key="i">
                {{ [row.parent_category_name, row.category_name].filter(Boolean).join(' / ') || '—' }} · {{ row.title }}
              </li>
            </ul>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="importOpen = false">{{ $t('admin.knowledgeBase.form.cancel') }}</Button>
          <Button :disabled="!importPreview.length || importing" @click="confirmImport">
            <Loader2 v-if="importing" class="mr-1.5 h-4 w-4 animate-spin" />
            {{ $t('admin.knowledgeBase.form.confirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete confirm -->
    <Dialog v-model:open="deleteOpen">
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle>{{ $t('admin.knowledgeBase.delete') }}</DialogTitle>
        </DialogHeader>
        <p class="text-sm">{{ $t('admin.knowledgeBase.confirmDelete') }}</p>
        <DialogFooter>
          <Button variant="outline" @click="deleteOpen = false">{{ $t('admin.knowledgeBase.form.cancel') }}</Button>
          <Button variant="destructive" :disabled="deleting" @click="confirmDelete">
            <Loader2 v-if="deleting" class="mr-1.5 h-4 w-4 animate-spin" />
            {{ $t('admin.knowledgeBase.delete') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@shared-ui/components/ui/card'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import {
  Plus, Download, Upload, SlidersHorizontal, ChevronDown, ChevronRight,
  ChevronLeft, FileText, Pencil, Trash2, Eye, Loader2
} from 'lucide-vue-next'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import KnowledgeFormDialog from './KnowledgeFormDialog.vue'
import KnowledgeFilterDialog from './KnowledgeFilterDialog.vue'
import ManageCategoriesDialog from './ManageCategoriesDialog.vue'
import KnowledgePreviewDialog from './KnowledgePreviewDialog.vue'
import {
  parseKnowledgeFile, downloadKnowledgeTemplate, exportKnowledgeArticles
} from './knowledgeXlsx.js'

const { t } = useI18n()
const emitter = useEmitter()

const loading = ref(false)
const categories = ref([])
const articles = ref([])
const total = ref(0)
const page = ref(1)
const perPage = ref(20)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / perPage.value)))
const allCollapsed = ref(false)
// Roots whose children are collapsed in the left tree. Default is collapsed
// so the page opens compact; loadCategories() will then mark every root as
// collapsed to keep this behaviour consistent on subsequent refreshes.
const collapsedRoots = reactive({})

const isRootExpanded = (id) => !collapsedRoots[id]
const toggleRoot = (id) => {
  collapsedRoots[id] = !collapsedRoots[id]
}
// A root is "selected" when the click scoped to it (parent only, no leaf).
const isRootSelected = (root) =>
  !selectedFilter.draft &&
  !selectedFilter.categoryId &&
  selectedFilter.parentCategoryId === root.id
const collapseAllRoots = () => {
  for (const root of categories.value || []) collapsedRoots[root.id] = true
}

const filterOpen = ref(false)
const formOpen = ref(false)
const manageOpen = ref(false)
const previewOpen = ref(false)
const importOpen = ref(false)
const deleteOpen = ref(false)
const importing = ref(false)
const deleting = ref(false)

const editing = ref(null)
const pendingDelete = ref(null)
const importFileInput = ref(null)
const importPreview = ref([])

const selectedFilter = reactive({
  categoryId: null,
  parentCategoryId: null,
  draft: false
})

const filterInitial = ref({})
const filterState = reactive({
  sort: 'desc',
  dateField: 'created_at',
  dateRange: 'all',
  dateFrom: '',
  dateTo: '',
  keywordField: 'all',
  keywords: ''
})

const preview = reactive({ title: '', content: '', meta: null })

const onMount = onMounted
onMount(async () => {
  await loadCategories()
  await loadArticles()
})

watch(page, () => loadArticles())

watch(
  () => ({ ...selectedFilter }),
  () => {
    page.value = 1
    loadArticles()
  },
  { deep: true }
)

const loadCategories = async () => {
  try {
    const { data } = await api.getKnowledgeCategories()
    categories.value = data?.data || data || []
    // Keep all root categories collapsed after every refresh; toggleRoot
    // still works as an opt-in expansion per node.
    collapseAllRoots()
  } catch (err) {
    handleHTTPError(err)
  }
}

// Resolve the selected time-range preset into concrete [from, to) bounds.
// All bounds are local dates; `from` is inclusive and `to` exclusive.
const resolveDateRange = (range) => {
  const startOfDay = (d) => {
    const x = new Date(d)
    x.setHours(0, 0, 0, 0)
    return x
  }
  const endOfDay = (d) => {
    const x = new Date(d)
    x.setHours(0, 0, 0, 0)
    x.setDate(x.getDate() + 1)
    return x
  }
  const now = new Date()
  switch (range) {
    case 'today':
      return [startOfDay(now), endOfDay(now)]
    case 'yesterday': {
      const y = new Date(now)
      y.setDate(y.getDate() - 1)
      return [startOfDay(y), endOfDay(y)]
    }
    case 'this_week': {
      // Week starts on Monday.
      const w = new Date(now)
      const dow = (w.getDay() + 6) % 7
      w.setDate(w.getDate() - dow)
      return [startOfDay(w), endOfDay(now)]
    }
    case 'this_month': {
      const m = new Date(now.getFullYear(), now.getMonth(), 1)
      return [startOfDay(m), endOfDay(now)]
    }
    case 'last_month': {
      const first = new Date(now.getFullYear(), now.getMonth() - 1, 1)
      const next = new Date(now.getFullYear(), now.getMonth(), 1)
      return [startOfDay(first), startOfDay(next)]
    }
    default:
      return [null, null]
  }
}

const buildListParams = () => {
  const params = {
    page: page.value,
    per_page: perPage.value,
    sort: filterState.sort,
    date_field: filterState.dateField || 'created_at',
    keyword_field: filterState.keywordField,
    keywords: filterState.keywords
  }
  if (selectedFilter.draft) {
    params.status = 'draft'
  } else {
    params.status = 'published'
  }
  // A leaf (level-2) click scopes by category_id; a root (level-1) click
  // scopes by parent_category_id so every article under that root (its
  // direct articles and its children's) is included.
  if (selectedFilter.categoryId) {
    params.category_id = selectedFilter.categoryId
  } else if (selectedFilter.parentCategoryId) {
    params.parent_category_id = selectedFilter.parentCategoryId
  }
  let from = null
  let to = null
  if (filterState.dateRange === 'custom') {
    from = filterState.dateFrom ? new Date(filterState.dateFrom) : null
    if (filterState.dateTo) {
      to = new Date(filterState.dateTo)
      to.setDate(to.getDate() + 1)
    }
  } else {
    ;[from, to] = resolveDateRange(filterState.dateRange)
  }
  if (from) params.date_from = from.toISOString()
  if (to) params.date_to = to.toISOString()
  return params
}

const loadArticles = async () => {
  loading.value = true
  try {
    const { data } = await api.listKnowledgeArticles(buildListParams())
    const payload = data?.data || data
    articles.value = payload?.articles || []
    total.value = payload?.total || 0
  } catch (err) {
    handleHTTPError(err)
  } finally {
    loading.value = false
  }
}

const selectCategory = (cat) => {
  selectedFilter.draft = false
  selectedFilter.parentCategoryId = cat.parent_id || null
  selectedFilter.categoryId = cat.parent_id ? cat.id : null
  // For a level-1 root, scope by parent_category_id so its leaves are included.
  if (!cat.parent_id) {
    selectedFilter.categoryId = null
    selectedFilter.parentCategoryId = cat.id
  }
}

const selectDraft = () => {
  selectedFilter.draft = true
  selectedFilter.categoryId = null
  selectedFilter.parentCategoryId = null
}

const openCreate = () => {
  editing.value = null
  formOpen.value = true
}

// The list endpoint intentionally omits `content` (only a 200-char preview
// is returned). Fetch the full article here so the editor mounts with the
// real body, not an empty string.
const openEdit = async (a) => {
  try {
    const { data } = await api.getKnowledgeArticle(a.id)
    editing.value = data?.data || data
  } catch (err) {
    handleHTTPError(err)
    return
  }
  formOpen.value = true
}

const askDelete = (a) => {
  pendingDelete.value = a
  deleteOpen.value = true
}

const confirmDelete = async () => {
  deleting.value = true
  try {
    await api.deleteKnowledgeArticle(pendingDelete.value.id)
    emitter.emit(EMITTER_EVENTS.TOAST, { title: t('globals.terms.deleted') })
    deleteOpen.value = false
    await loadArticles()
  } catch (err) {
    handleHTTPError(err)
  } finally {
    deleting.value = false
  }
}

const openPreview = async (a) => {
  try {
    const { data } = await api.getKnowledgeArticle(a.id)
    const article = data?.data || data
    preview.title = article.title
    preview.content = article.content
    preview.meta = {
      author: article.author,
      category: [article.parent_category_name, article.category_name].filter(Boolean).join(' / ')
    }
    previewOpen.value = true
  } catch (err) {
    handleHTTPError(err)
  }
}

const openPreviewFromForm = (payload) => {
  preview.title = payload.title
  preview.content = payload.content
  preview.meta = { author: payload.author, category: payload.category }
  previewOpen.value = true
}

const onSaved = async () => {
  emitter.emit(EMITTER_EVENTS.TOAST, { title: t('globals.terms.saved') })
  await loadArticles()
}

const onApplyFilter = (form) => {
  Object.assign(filterState, form)
  page.value = 1
  loadArticles()
}

const downloadTemplate = () => downloadKnowledgeTemplate(t)

const openImportDialog = () => {
  importPreview.value = []
  importOpen.value = true
}

const triggerImportFile = () => importFileInput.value?.click()
const onImportPick = (e) => parseImportFile(e.target.files)
const onImportDrop = (e) => parseImportFile(e.dataTransfer.files)
const parseImportFile = async (files) => {
  if (!files || !files.length) return
  try {
    const items = await parseKnowledgeFile(files[0])
    importPreview.value = items
  } catch (err) {
    handleHTTPError(err)
  }
}

const confirmImport = async () => {
  importing.value = true
  try {
    const items = importPreview.value.map((row) => ({
      category_code: row.category_code,
      category_name: row.category_name,
      parent_category_code: row.parent_category_code,
      parent_category_name: row.parent_category_name,
      external_id: row.external_id,
      article_code: row.article_code,
      title: row.title,
      author: row.author,
      content: row.content
    }))
    const { data } = await api.importKnowledgeArticles({ items })
    const stats = data?.data || data
    emitter.emit(EMITTER_EVENTS.TOAST, {
      title: t('admin.knowledgeBase.importStats', {
        categories: stats?.created_categories || 0,
        created: stats?.created_articles || 0,
        updated: stats?.updated_articles || 0,
        errors: stats?.errors?.length || 0
      })
    })
    importOpen.value = false
    await loadCategories()
    await loadArticles()
  } catch (err) {
    handleHTTPError(err)
  } finally {
    importing.value = false
  }
}

const exportAll = async () => {
  try {
    const { data } = await api.exportKnowledgeArticles()
    const rows = data?.data || data || []
    if (!rows.length) {
      emitter.emit(EMITTER_EVENTS.TOAST, { title: t('admin.knowledgeBase.exportEmpty') })
      return
    }
    exportKnowledgeArticles(rows, t)
  } catch (err) {
    handleHTTPError(err)
  }
}

const formatTime = (iso) => {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}
</script>
