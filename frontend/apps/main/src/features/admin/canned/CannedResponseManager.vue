<template>
  <div class="space-y-4">
    <!-- Toolbar -->
    <Card>
      <CardContent class="flex flex-wrap items-center gap-3 p-3">
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button size="sm">
              <Plus class="mr-1.5 h-4 w-4" />
              {{ $t('admin.cannedResponses.addResponse') }}
              <ChevronDown class="ml-1.5 h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" class="w-44">
            <DropdownMenuItem :disabled="!selectedCategory" @click="startAddResponse">
              <MessageSquarePlus class="mr-2 h-4 w-4" />
              {{ $t('admin.cannedResponses.addResponse') }}
            </DropdownMenuItem>
            <DropdownMenuItem @click="startAddCategory(selectedCategoryId)">
              <FolderPlus class="mr-2 h-4 w-4" />
              {{ $t('admin.cannedResponses.addCategory') }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <div class="relative min-w-[220px] flex-1">
          <Search
            class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            v-model="search"
            type="text"
            class="h-9 pl-9 pr-9 text-sm"
            :placeholder="$t('admin.cannedResponses.searchPlaceholder')"
          />
          <button
            v-if="search"
            type="button"
            class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground"
            @click="search = ''"
          >
            <X class="h-4 w-4" />
          </button>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" @click="downloadTemplate">
            <Download class="mr-1.5 h-4 w-4" />
            {{ $t('admin.cannedResponses.downloadTemplate') }}
          </Button>
          <Button size="sm" variant="outline" @click="openImportDialog">
            <Upload class="mr-1.5 h-4 w-4" />
            {{ $t('admin.cannedResponses.import') }}
          </Button>
          <Button size="sm" variant="outline" @click="exportAll">
            <Download class="mr-1.5 h-4 w-4" />
            {{ $t('admin.cannedResponses.export') }}
          </Button>
          <Button
            size="sm"
            variant="outline"
            class="text-destructive hover:text-destructive"
            :disabled="!hasAnyData"
            @click="openDeleteDialog('all', null)"
          >
            <Trash2 class="mr-1.5 h-4 w-4" />
            {{ $t('admin.cannedResponses.deleteAll') }}
          </Button>
        </div>
      </CardContent>
    </Card>

    <Spinner v-if="loading" />

    <div v-else class="grid gap-4 md:grid-cols-[auto_1fr]">
      <!-- Categories: the column is sized by its content (longest category
           name), bounded by a min/max width so it never squeezes the responses
           panel. -->
      <Card class="self-start md:min-w-[240px] md:max-w-[360px] lg:max-w-[420px]">
        <CardHeader class="flex-row items-center justify-between space-y-0 px-3 py-3">
          <CardTitle class="text-sm font-semibold">
            {{ $t('admin.cannedResponses.categories') }}
          </CardTitle>
          <div class="flex items-center gap-0.5">
            <button
              type="button"
              class="rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              :title="allExpanded ? $t('admin.cannedResponses.collapseAll') : $t('admin.cannedResponses.expandAll')"
              @click="toggleExpandAll"
            >
              <ChevronsUpDown class="h-4 w-4" />
            </button>
            <button
              type="button"
              class="rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              :title="$t('admin.cannedResponses.addCategory')"
              @click="startAddCategory(null)"
            >
              <Plus class="h-4 w-4" />
            </button>
          </div>
        </CardHeader>
        <CardContent class="max-h-[62vh] space-y-0.5 overflow-y-auto p-2 pr-4">
          <div v-if="addingCategory && addingCategoryParent === null" class="px-1 py-1">
            <Input
              v-model="newCategoryName"
              type="text"
              class="h-8 text-sm"
              :placeholder="$t('admin.cannedResponses.categoryNamePlaceholder')"
              @keydown.enter="createCategory"
              @keydown.esc="cancelAddCategory"
            />
            <div class="mt-1.5 flex gap-1.5">
              <Button size="sm" class="h-7 text-xs" :disabled="categorySaving" @click="createCategory">
                <Check class="mr-1 h-3.5 w-3.5" />
                {{ $t('globals.terms.save') }}
              </Button>
              <Button size="sm" variant="outline" class="h-7 text-xs" @click="cancelAddCategory">
                {{ $t('globals.terms.cancel') }}
              </Button>
            </div>
          </div>

          <div v-for="row in categoryRows" :key="row.id">
          <div
            class="group flex cursor-pointer items-center gap-1 rounded-md py-1.5 pr-1.5 transition-colors"
            :class="row.id === selectedCategoryId ? 'bg-primary/10 font-medium text-primary' : 'hover:bg-muted'"
            :style="{ paddingLeft: `${row.depth * 14 + 6}px` }"
            @click="selectCategory(row.id)"
          >
            <button
              v-if="row.hasChildren"
              type="button"
              class="shrink-0 rounded text-muted-foreground hover:text-foreground"
              @click.stop="toggleExpand(row.id)"
            >
              <ChevronRight v-if="!expanded.has(row.id)" class="h-3.5 w-3.5" />
              <ChevronDown v-else class="h-3.5 w-3.5" />
            </button>
            <span v-else class="w-3.5 shrink-0" />

            <template v-if="editingCategoryId === row.id">
              <Input
                v-model="editingCategoryName"
                type="text"
                class="h-7 flex-1 text-sm"
                @click.stop
                @keydown.enter="saveCategoryEdit(row)"
                @keydown.esc="cancelCategoryEdit"
              />
              <button
                type="button"
                class="shrink-0 rounded p-1 text-primary hover:bg-muted"
                @click.stop="saveCategoryEdit(row)"
              >
                <Check class="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                class="shrink-0 rounded p-1 text-muted-foreground hover:bg-muted"
                @click.stop="cancelCategoryEdit"
              >
                <X class="h-3.5 w-3.5" />
              </button>
            </template>

            <template v-else>
              <span class="min-w-0 flex-1 truncate text-sm" :title="row.name">{{ row.name }}</span>
              <div
                class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
              >
                <button
                  type="button"
                  class="rounded p-0.5 text-muted-foreground hover:text-foreground"
                  :title="$t('admin.cannedResponses.addChildCategory')"
                  @click.stop="startAddCategory(row.id)"
                >
                  <FolderPlus class="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  class="rounded p-0.5 text-muted-foreground hover:text-foreground disabled:opacity-20"
                  :disabled="categoryMoving || row.isFirstSibling"
                  :title="$t('globals.terms.moveUp')"
                  @click.stop="moveCategory(row, -1)"
                >
                  <ChevronUp class="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  class="rounded p-0.5 text-muted-foreground hover:text-foreground disabled:opacity-20"
                  :disabled="categoryMoving || row.isLastSibling"
                  :title="$t('globals.terms.moveDown')"
                  @click.stop="moveCategory(row, 1)"
                >
                  <ChevronDown class="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  class="rounded p-0.5 text-muted-foreground hover:text-foreground"
                  :title="$t('globals.terms.edit')"
                  @click.stop="startCategoryEdit(row)"
                >
                  <Pencil class="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  class="rounded p-0.5 text-muted-foreground hover:text-destructive"
                  :title="$t('globals.terms.delete')"
                  @click.stop="openDeleteDialog('category', row)"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                </button>
              </div>
            </template>
          </div>

          <div
            v-if="addingCategory && addingCategoryParent === row.id"
            class="px-1 py-1"
            :style="{ paddingLeft: `${(row.depth + 1) * 14 + 6}px` }"
          >
            <Input
              v-model="newCategoryName"
              type="text"
              class="h-8 text-sm"
              :placeholder="$t('admin.cannedResponses.categoryNamePlaceholder')"
              @keydown.enter="createCategory"
              @keydown.esc="cancelAddCategory"
            />
            <div class="mt-1.5 flex gap-1.5">
              <Button size="sm" class="h-7 text-xs" :disabled="categorySaving" @click="createCategory">
                <Check class="mr-1 h-3.5 w-3.5" />
                {{ $t('globals.terms.save') }}
              </Button>
              <Button size="sm" variant="outline" class="h-7 text-xs" @click="cancelAddCategory">
                {{ $t('globals.terms.cancel') }}
              </Button>
            </div>
          </div>
          </div>

          <div
            v-if="!tree.length && !addingCategory"
            class="px-2 py-6 text-center text-xs text-muted-foreground"
          >
            {{ $t('admin.cannedResponses.emptyCategories') }}
          </div>
        </CardContent>
      </Card>

      <!-- Responses -->
      <Card class="self-start">
        <CardHeader class="flex-row items-center justify-between space-y-0 px-4 py-3">
          <CardTitle class="truncate text-sm font-semibold">
            <template v-if="search.trim() && selectedCategory">
              {{ selectedCategory.name }}
              <span class="font-normal text-muted-foreground">· {{ search.trim() }}</span>
              <span class="ml-1 text-xs font-normal text-muted-foreground">({{ displayedResponses.length }})</span>
            </template>
            <template v-else-if="selectedCategory">{{ selectedCategory.name }}</template>
            <template v-else>{{ $t('admin.cannedResponses.title') }}</template>
          </CardTitle>
        </CardHeader>
        <CardContent class="max-h-[62vh] space-y-2 overflow-y-auto p-3">
          <div v-if="addingResponse" class="rounded-lg border border-primary/40 p-3">
            <Textarea
              v-model="newResponseContent"
              rows="3"
              class="text-sm"
              :placeholder="$t('admin.cannedResponses.responsePlaceholder')"
              @keydown.esc="cancelAddResponse"
            />
            <div class="mt-2 flex gap-2">
              <Button size="sm" class="h-7 text-xs" :disabled="responseSaving" @click="createResponse">
                <Check class="mr-1 h-3.5 w-3.5" />
                {{ $t('globals.terms.save') }}
              </Button>
              <Button size="sm" variant="outline" class="h-7 text-xs" @click="cancelAddResponse">
                {{ $t('globals.terms.cancel') }}
              </Button>
            </div>
          </div>

          <div
            v-for="item in displayedResponses"
            :key="item.id"
            class="group rounded-lg border p-3 transition-colors hover:border-primary/40"
          >
            <div class="flex items-start gap-2">
              <div class="min-w-0 flex-1">
                <p
                  v-if="search && item.__path"
                  class="mb-1 text-xs text-muted-foreground"
                >
                  {{ item.__path }}
                </p>
                <template v-if="editingResponseId === item.id">
                  <Textarea v-model="editingResponseContent" rows="3" class="text-sm" @keydown.esc="cancelResponseEdit" />
                </template>
                <p v-else class="whitespace-pre-wrap break-words text-sm">{{ item.content }}</p>
              </div>

              <div class="flex shrink-0 items-center gap-0.5">
                <template v-if="editingResponseId === item.id">
                  <button
                    type="button"
                    class="rounded p-1 text-primary hover:bg-muted"
                    @click="saveResponseEdit(item)"
                  >
                    <Check class="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    class="rounded p-1 text-muted-foreground hover:bg-muted"
                    @click="cancelResponseEdit"
                  >
                    <X class="h-4 w-4" />
                  </button>
                </template>
                <template v-else>
                  <div
                    class="flex items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <button
                      type="button"
                      class="rounded p-1 text-muted-foreground hover:text-foreground disabled:opacity-20"
                      :disabled="search || isFirstResponse(item)"
                      :title="$t('globals.terms.moveUp')"
                      @click="moveResponse(item, -1)"
                    >
                      <ChevronUp class="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      class="rounded p-1 text-muted-foreground hover:text-foreground disabled:opacity-20"
                      :disabled="search || isLastResponse(item)"
                      :title="$t('globals.terms.moveDown')"
                      @click="moveResponse(item, 1)"
                    >
                      <ChevronDown class="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      class="rounded p-1 text-muted-foreground hover:text-foreground"
                      :title="$t('globals.terms.edit')"
                      @click="startResponseEdit(item)"
                    >
                      <Pencil class="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      class="rounded p-1 text-muted-foreground hover:text-destructive"
                      :title="$t('globals.terms.delete')"
                      @click="openDeleteDialog('response', item)"
                    >
                      <Trash2 class="h-4 w-4" />
                    </button>
                  </div>
                </template>
              </div>
            </div>
          </div>

          <div
            v-if="!addingResponse && displayedResponses.length === 0"
            class="py-10 text-center text-sm text-muted-foreground"
          >
            <template v-if="search.trim() && selectedCategory">
              {{ $t('admin.cannedResponses.noResults') }}
            </template>
            <template v-else-if="selectedCategory">{{ $t('admin.cannedResponses.emptyResponses') }}</template>
            <template v-else>{{ $t('admin.cannedResponses.selectCategory') }}</template>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Delete confirmation -->
    <Dialog :open="deleteDialog.open" @update:open="deleteDialog.open = $event">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ deleteTitle }}</DialogTitle>
        </DialogHeader>
        <p class="text-sm text-muted-foreground">{{ deleteConfirmText }}</p>
        <DialogFooter>
          <Button variant="outline" @click="deleteDialog.open = false">
            {{ $t('globals.terms.cancel') }}
          </Button>
          <Button variant="destructive" :disabled="deleting" @click="performDelete">
            <Loader2 v-if="deleting" class="mr-1.5 h-4 w-4 animate-spin" />
            {{ $t('globals.terms.delete') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Import dialog -->
    <Dialog :open="importDialog.open" @update:open="importDialog.open = $event">
      <DialogContent class="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{{ $t('admin.cannedResponses.importTitle') }}</DialogTitle>
          <p class="text-sm text-muted-foreground">{{ $t('admin.cannedResponses.importDescription') }}</p>
        </DialogHeader>

        <div
          class="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed px-6 py-8 text-center transition-colors hover:bg-muted/50"
          @click="fileInputRef?.click()"
          @dragover.prevent
          @drop.prevent="handleFileDrop"
        >
          <input ref="fileInputRef" type="file" accept=".xlsx,.xls" class="hidden" @change="handleFileChange" />
          <FileUp class="h-8 w-8 text-muted-foreground" />
          <p class="text-sm font-medium">{{ $t('admin.cannedResponses.importDropzone') }}</p>
          <p class="text-xs text-muted-foreground">{{ $t('admin.cannedResponses.importFileHint') }}</p>
        </div>

        <div v-if="importDialog.fileName" class="flex items-center gap-2 rounded-lg bg-muted/50 px-3 py-2 text-sm">
          <FileSpreadsheet class="h-4 w-4 shrink-0 text-muted-foreground" />
          <span class="min-w-0 truncate">{{ importDialog.fileName }}</span>
          <span class="ml-auto shrink-0 text-muted-foreground">
            {{
              $t('admin.cannedResponses.importParsed', {
                categories: importDialog.categoryCount,
                responses: importDialog.responseCount
              })
            }}
          </span>
        </div>
        <div
          v-if="importDialog.errorRows.length"
          class="rounded-lg bg-destructive/10 px-3 py-2 text-xs text-destructive"
        >
          {{ $t('admin.quickReply.importEmptyRows', { rows: importDialog.errorRows.join(', ') }) }}
        </div>
        <div
          v-else-if="importDialog.fileName && importDialog.preview.length === 0"
          class="py-6 text-center text-sm text-muted-foreground"
        >
          {{ $t('admin.cannedResponses.importNoData') }}
        </div>

        <div v-if="importDialog.preview.length" class="max-h-72 overflow-y-auto rounded-md border p-2">
          <div v-for="(cat, idx) in importDialog.preview" :key="idx" class="py-1">
            <div class="text-sm font-medium">{{ cat.name }}</div>
            <div
              v-for="(res, rIdx) in cat.responses"
              :key="`r-${rIdx}`"
              class="truncate pl-3 text-xs text-muted-foreground"
            >
              {{ res }}
            </div>
            <div v-for="(child, cIdx) in cat.children" :key="`c-${cIdx}`" class="pl-3">
              <div class="text-xs font-medium text-muted-foreground">[{{ child.name }}]</div>
              <div
                v-for="(res, rIdx) in child.responses"
                :key="`cr-${rIdx}`"
                class="truncate pl-3 text-xs text-muted-foreground"
              >
                {{ res }}
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" :disabled="importing" @click="closeImportDialog">
            {{ $t('admin.cannedResponses.importCancel') }}
          </Button>
          <Button :disabled="importing || importDialog.preview.length === 0" @click="confirmImport">
            <Loader2 v-if="importing" class="mr-1.5 h-4 w-4 animate-spin" />
            {{ $t('admin.cannedResponses.importConfirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import {
  Plus,
  Search,
  X,
  Check,
  Pencil,
  Trash2,
  ChevronRight,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Download,
  Upload,
  FileUp,
  FileSpreadsheet,
  Loader2,
  FolderPlus,
  MessageSquarePlus
} from 'lucide-vue-next'
import { Card, CardHeader, CardTitle, CardContent } from '@shared-ui/components/ui/card'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem
} from '@shared-ui/components/ui/dropdown-menu'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { parseCannedFile, countCannedTree, downloadCannedTemplate, exportCannedTree } from './cannedXlsx.js'

const MAX_NAME_LENGTH = 20
const MAX_CONTENT_LENGTH = 1000

// Key used to group top-level categories (those without a parent).
const ROOT_PARENT_KEY = 'root'

const { t } = useI18n()
const emitter = useEmitter()

const loading = ref(true)
const tree = ref([])
const expanded = ref(new Set())
const expandedInitialized = ref(false)
const selectedCategoryId = ref(null)
const search = ref('')

const addingCategory = ref(false)
const addingCategoryParent = ref(null)
const newCategoryName = ref('')
const categorySaving = ref(false)
const categoryMoving = ref(false)
const editingCategoryId = ref(null)
const editingCategoryName = ref('')

const addingResponse = ref(false)
const newResponseContent = ref('')
const responseSaving = ref(false)
const editingResponseId = ref(null)
const editingResponseContent = ref('')

const deleting = ref(false)
const deleteDialog = reactive({ open: false, type: null, item: null })

const importing = ref(false)
const fileInputRef = ref(null)
const importDialog = reactive({
  open: false,
  fileName: '',
  preview: [],
  errorRows: [],
  categoryCount: 0,
  responseCount: 0
})

const showValidationError = (message) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: message })
}

// Length is counted in unicode code points to match the server-side limit
// (Postgres length()) and the template's "character" rule.
const validateName = (name) => {
  if (!name) {
    showValidationError(t('admin.cannedResponses.nameRequired'))
    return false
  }
  if ([...name].length > MAX_NAME_LENGTH) {
    showValidationError(t('admin.cannedResponses.nameTooLong'))
    return false
  }
  return true
}

const validateContent = (content) => {
  if (!content) {
    showValidationError(t('admin.cannedResponses.contentRequired'))
    return false
  }
  if ([...content].length > MAX_CONTENT_LENGTH) {
    showValidationError(t('admin.cannedResponses.contentTooLong'))
    return false
  }
  return true
}

// Walk all categories once and index them by id, also exposing the full list.
const allCategories = computed(() => {
  const list = []
  const walk = (nodes, parentId) => {
    for (const node of nodes) {
      list.push({ node, parentId })
      if (node.children?.length) walk(node.children, node.id)
    }
  }
  walk(tree.value, null)
  return list
})

const categoryMap = computed(() => {
  const map = new Map()
  for (const { node } of allCategories.value) map.set(node.id, node)
  return map
})

// Sibling groups keyed by parent id (`root` for top-level categories). The
// order follows the API ordering (sort_order asc, id asc), which is the order
// the tree is assembled in, and is used by the move up/down actions.
const siblingsByParent = computed(() => {
  const map = new Map()
  for (const { node, parentId } of allCategories.value) {
    const key = parentId ?? ROOT_PARENT_KEY
    if (!map.has(key)) map.set(key, [])
    map.get(key).push(node.id)
  }
  return map
})

const selectedCategory = computed(() =>
  selectedCategoryId.value ? categoryMap.value.get(selectedCategoryId.value) : null
)

const hasAnyData = computed(() => allCategories.value.length > 0)

// Flatten the tree into visible rows honoring the expanded set, annotating each
// row with its position among its siblings so the move up/down buttons know
// when they are at either end of a group.
const categoryRows = computed(() => {
  const rows = []
  const walk = (nodes, depth, parentId) => {
    const siblings = siblingsByParent.value.get(parentId ?? ROOT_PARENT_KEY) || []
    for (const node of nodes) {
      const index = siblings.indexOf(node.id)
      rows.push({
        id: node.id,
        name: node.name,
        depth,
        parentId: parentId ?? null,
        sortOrder: node.sort_order,
        hasChildren: (node.children?.length || 0) > 0,
        isFirstSibling: index <= 0,
        isLastSibling: index === -1 || index === siblings.length - 1
      })
      if (node.children?.length && expanded.value.has(node.id)) {
        walk(node.children, depth + 1, node.id)
      }
    }
  }
  walk(tree.value, 0, null)
  return rows
})

const allExpanded = computed(() => {
  const parents = allCategories.value.filter(({ node }) => (node.children?.length || 0) > 0)
  return parents.length > 0 && parents.every(({ node }) => expanded.value.has(node.id))
})

// Search is scoped to the selected category: it covers the category itself and
// all of its descendants. Every match carries the path relative to the selected
// category so it can be located in the tree.
const displayedResponses = computed(() => {
  const selected = selectedCategory.value
  if (!selected) {
    return []
  }
  const term = search.value.trim().toLowerCase()
  if (!term) {
    return selected.responses || []
  }
  const matches = []
  const walk = (node, path) => {
    for (const response of node.responses || []) {
      if (response.content.toLowerCase().includes(term)) {
        matches.push({ ...response, __path: path.join(' / ') })
      }
    }
    for (const child of node.children || []) {
      walk(child, [...path, child.name])
    }
  }
  walk(selected, [])
  return matches
})

const showError = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })
}

const showSuccess = (description) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description })
}

const loadTree = async () => {
  const resp = await api.getCannedResponses()
  tree.value = resp.data.data || []

  // Expand everything on the first load.
  if (!expandedInitialized.value) {
    const ids = new Set()
    for (const { node } of allCategories.value) {
      if (node.children?.length) ids.add(node.id)
    }
    expanded.value = ids
    expandedInitialized.value = true
  }

  // Keep a valid selection, otherwise default to the first top-level category.
  if (selectedCategoryId.value && !categoryMap.value.has(selectedCategoryId.value)) {
    selectedCategoryId.value = null
  }
  if (!selectedCategoryId.value && tree.value.length) {
    selectedCategoryId.value = tree.value[0].id
  }
}

const refresh = async () => {
  try {
    await loadTree()
  } catch (error) {
    showError(error)
  }
}

onMounted(async () => {
  try {
    loading.value = true
    await loadTree()
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
})

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------
const toggleExpand = (id) => {
  const next = new Set(expanded.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  expanded.value = next
}

const toggleExpandAll = () => {
  if (allExpanded.value) {
    expanded.value = new Set()
    return
  }
  const ids = new Set()
  for (const { node } of allCategories.value) {
    if (node.children?.length) ids.add(node.id)
  }
  expanded.value = ids
}

// Selecting another category keeps the search term so the same keyword can be
// inspected across categories (the search is scoped to the selection).
const selectCategory = (id) => {
  selectedCategoryId.value = id
}

const startAddCategory = (parentId) => {
  addingCategory.value = true
  addingCategoryParent.value = parentId ?? null
  newCategoryName.value = ''
  if (parentId) {
    selectedCategoryId.value = parentId
    if (!expanded.value.has(parentId)) toggleExpand(parentId)
  }
}

const cancelAddCategory = () => {
  addingCategory.value = false
  addingCategoryParent.value = null
  newCategoryName.value = ''
}

const createCategory = async () => {
  const name = newCategoryName.value.trim()
  if (!validateName(name)) return

  const parentId = addingCategoryParent.value
  categorySaving.value = true
  try {
    const resp = await api.createCannedCategory({ name, parent_id: parentId })
    const created = resp.data.data
    cancelAddCategory()
    await refresh()
    if (parentId && !expanded.value.has(parentId)) toggleExpand(parentId)
    if (created?.id) selectedCategoryId.value = created.id
    showSuccess(t('globals.messages.savedSuccessfully'))
  } catch (error) {
    showError(error)
  } finally {
    categorySaving.value = false
  }
}

const startCategoryEdit = (row) => {
  editingCategoryId.value = row.id
  editingCategoryName.value = row.name
}

const cancelCategoryEdit = () => {
  editingCategoryId.value = null
  editingCategoryName.value = ''
}

const saveCategoryEdit = async (row) => {
  const name = editingCategoryName.value.trim()
  if (!validateName(name)) return
  try {
    await api.updateCannedCategory(row.id, {
      name,
      parent_id: categoryMap.value.get(row.id)?.parent_id ?? null,
      sort_order: categoryMap.value.get(row.id)?.sort_order ?? 0
    })
    cancelCategoryEdit()
    await refresh()
    showSuccess(t('globals.messages.savedSuccessfully'))
  } catch (error) {
    showError(error)
  }
}

// Moves a category up or down inside its sibling group by reassigning the sort
// order of the affected siblings (usually two). When legacy rows share the same
// sort order, the whole group is renumbered so the move always takes effect.
const moveCategory = async (row, direction) => {
  if (categoryMoving.value) return

  const siblings = siblingsByParent.value.get(row.parentId ?? ROOT_PARENT_KEY) || []
  const index = siblings.indexOf(row.id)
  const targetIndex = index + direction
  if (index === -1 || targetIndex < 0 || targetIndex >= siblings.length) {
    return
  }

  const order = [...siblings]
  const [moved] = order.splice(index, 1)
  order.splice(targetIndex, 0, moved)

  const updates = order
    .map((id, sortOrder) => ({ node: categoryMap.value.get(id), sortOrder }))
    .filter(({ node, sortOrder }) => node && node.sort_order !== sortOrder)

  if (!updates.length) return

  categoryMoving.value = true
  try {
    for (const { node, sortOrder } of updates) {
      await api.updateCannedCategory(node.id, {
        name: node.name,
        parent_id: node.parent_id ?? null,
        sort_order: sortOrder
      })
    }
    await refresh()
  } catch (error) {
    showError(error)
  } finally {
    categoryMoving.value = false
  }
}

// ---------------------------------------------------------------------------
// Responses
// ---------------------------------------------------------------------------
const startAddResponse = () => {
  if (!selectedCategoryId.value) return
  search.value = ''
  addingResponse.value = true
  newResponseContent.value = ''
}

const cancelAddResponse = () => {
  addingResponse.value = false
  newResponseContent.value = ''
}

const createResponse = async () => {
  const content = newResponseContent.value.trim()
  if (!validateContent(content)) return
  responseSaving.value = true
  try {
    await api.createCannedResponse({ category_id: selectedCategoryId.value, content })
    cancelAddResponse()
    await refresh()
    showSuccess(t('globals.messages.savedSuccessfully'))
  } catch (error) {
    showError(error)
  } finally {
    responseSaving.value = false
  }
}

const startResponseEdit = (item) => {
  editingResponseId.value = item.id
  editingResponseContent.value = item.content
}

const cancelResponseEdit = () => {
  editingResponseId.value = null
  editingResponseContent.value = ''
}

const saveResponseEdit = async (item) => {
  const content = editingResponseContent.value.trim()
  if (!validateContent(content)) return
  try {
    await api.updateCannedResponse(item.id, { content, sort_order: item.sort_order })
    cancelResponseEdit()
    await refresh()
    showSuccess(t('globals.messages.savedSuccessfully'))
  } catch (error) {
    showError(error)
  }
}

const isFirstResponse = (item) => {
  const list = selectedCategory.value?.responses || []
  return list.length === 0 || list[0].id === item.id
}

const isLastResponse = (item) => {
  const list = selectedCategory.value?.responses || []
  return list.length === 0 || list[list.length - 1].id === item.id
}

const moveResponse = async (item, direction) => {
  const list = selectedCategory.value?.responses || []
  const index = list.findIndex((r) => r.id === item.id)
  const swapWith = list[index + direction]
  if (!swapWith) return
  try {
    await api.updateCannedResponse(item.id, { content: item.content, sort_order: swapWith.sort_order })
    await api.updateCannedResponse(swapWith.id, { content: swapWith.content, sort_order: item.sort_order })
    await refresh()
  } catch (error) {
    showError(error)
  }
}

// ---------------------------------------------------------------------------
// Delete (single / all)
// ---------------------------------------------------------------------------
const openDeleteDialog = (type, item) => {
  deleteDialog.type = type
  deleteDialog.item = item
  deleteDialog.open = true
}

const deleteTitle = computed(() => {
  if (deleteDialog.type === 'category') return t('admin.cannedResponses.deleteCategoryTitle')
  if (deleteDialog.type === 'response') return t('admin.cannedResponses.deleteResponseTitle')
  return t('admin.cannedResponses.deleteAllTitle')
})

const deleteConfirmText = computed(() => {
  if (deleteDialog.type === 'category') {
    return t('admin.cannedResponses.deleteCategoryConfirm', { name: deleteDialog.item?.name })
  }
  if (deleteDialog.type === 'response') return t('admin.cannedResponses.deleteResponseConfirm')
  return t('admin.cannedResponses.deleteAllConfirm')
})

const performDelete = async () => {
  deleting.value = true
  try {
    if (deleteDialog.type === 'category') {
      await api.deleteCannedCategory(deleteDialog.item.id)
    } else if (deleteDialog.type === 'response') {
      await api.deleteCannedResponse(deleteDialog.item.id)
    } else {
      await api.deleteAllCannedResponses()
      selectedCategoryId.value = null
      showSuccess(t('admin.cannedResponses.deleteAllSuccess'))
    }
    deleteDialog.open = false
    await refresh()
  } catch (error) {
    showError(error)
  } finally {
    deleting.value = false
  }
}

// ---------------------------------------------------------------------------
// Template / import / export
// ---------------------------------------------------------------------------
const downloadTemplate = () => {
  downloadCannedTemplate(t)
}

const exportAll = () => {
  if (!tree.value.length) {
    showSuccess(t('admin.cannedResponses.exportEmpty'))
    return
  }
  exportCannedTree(tree.value, t)
}

const openImportDialog = () => {
  importDialog.fileName = ''
  importDialog.preview = []
  importDialog.errorRows = []
  importDialog.categoryCount = 0
  importDialog.responseCount = 0
  importDialog.open = true
}

const closeImportDialog = () => {
  importDialog.open = false
  importDialog.fileName = ''
  importDialog.preview = []
  importDialog.errorRows = []
  if (fileInputRef.value) fileInputRef.value.value = ''
}

const handleFileChange = (event) => {
  const file = event.target.files?.[0]
  if (file) parseFile(file)
}

const handleFileDrop = (event) => {
  const file = event.dataTransfer?.files?.[0]
  if (file) parseFile(file)
}

const parseFile = async (file) => {
  try {
    const { tree: parsedTree, errorRows } = await parseCannedFile(file)
    const counts = countCannedTree(parsedTree)
    importDialog.fileName = file.name
    importDialog.preview = parsedTree
    importDialog.errorRows = errorRows
    importDialog.categoryCount = counts.categories
    importDialog.responseCount = counts.responses
  } catch (error) {
    closeImportDialog()
    showError(error)
  }
}

const confirmImport = async () => {
  if (importing.value || importDialog.preview.length === 0) return
  importing.value = true
  try {
    const resp = await api.importCannedResponses({ categories: importDialog.preview })
    const stats = resp.data.data || {}
    closeImportDialog()
    await refresh()
    showSuccess(
      t('admin.cannedResponses.importStats', {
        createdCategories: stats.created_categories ?? 0,
        createdResponses: stats.created_responses ?? 0,
        skippedResponses: stats.skipped_responses ?? 0
      })
    )
  } catch (error) {
    showError(error)
  } finally {
    importing.value = false
  }
}
</script>
