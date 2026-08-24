<template>
  <div>
    <Card>
    <CardHeader class="flex-row items-center justify-between space-y-0">
      <CardTitle>{{ $t('admin.quickReply.topics') }}</CardTitle>
      <div class="flex items-center gap-2">
        <Button size="sm" variant="outline" @click="exportTopics">
          <Download class="w-4 h-4 mr-1.5" />
          {{ $t('admin.quickReply.export') }}
        </Button>
        <Button size="sm" variant="outline" @click="openImportDialog">
          <Upload class="w-4 h-4 mr-1.5" />
          {{ $t('admin.quickReply.import') }}
        </Button>
        <Button size="sm" variant="outline" @click="startAddTopic">
          <Plus class="w-4 h-4 mr-1.5" />
          {{ $t('admin.quickReply.addTopic') }}
        </Button>
      </div>
    </CardHeader>
    <CardContent>
      <div class="grid gap-4 md:grid-cols-[300px_1fr]">
        <!-- Topics list -->
        <div class="space-y-2">
          <div v-if="addingTopic" class="flex items-center gap-2 rounded-lg border border-primary/40 px-3 py-1.5">
            <Input
              v-model="newTopicName"
              type="text"
              class="h-8 text-sm"
              :placeholder="$t('admin.quickReply.topicNamePlaceholder')"
              @keydown.enter="createTopic"
              @keydown.esc="addingTopic = false"
            />
            <Button size="sm" class="h-8 shrink-0" :disabled="!newTopicName.trim() || topicSaving" @click="createTopic">
              <Check class="w-4 h-4" />
            </Button>
            <Button size="sm" variant="ghost" class="h-8 shrink-0" @click="addingTopic = false">
              <X class="w-4 h-4" />
            </Button>
          </div>

          <div
            v-for="topic in topics"
            :key="topic.id"
            :class="[
              'group rounded-lg border px-3 py-2 cursor-pointer transition-colors',
              topic.id === selectedTopicId ? 'border-primary bg-primary/5' : 'hover:bg-muted'
            ]"
            @click="selectedTopicId = topic.id"
          >
            <div class="flex items-center gap-1.5">
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground disabled:opacity-20"
                :disabled="isFirst(topic)"
                :title="$t('globals.terms.moveUp')"
                @click.stop="moveTopic(topic, -1)"
              >
                <ChevronUp class="w-4 h-4" />
              </button>
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground disabled:opacity-20"
                :disabled="isLast(topic)"
                :title="$t('globals.terms.moveDown')"
                @click.stop="moveTopic(topic, 1)"
              >
                <ChevronDown class="w-4 h-4" />
              </button>
              <span v-if="editingTopicId !== topic.id" class="flex-1 min-w-0 truncate text-sm font-medium">{{ topic.name }}</span>
              <span class="text-xs text-muted-foreground shrink-0">{{ topic.questions?.length || 0 }}</span>
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground shrink-0"
                :title="$t('globals.terms.edit')"
                @click.stop="startTopicEdit(topic)"
              >
                <Pencil class="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                class="text-muted-foreground hover:text-destructive shrink-0"
                :title="$t('globals.terms.delete')"
                @click.stop="openDeleteDialog('topic', topic)"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
            <!-- Show aliases below the topic name (when not editing) -->
            <div v-if="editingTopicId !== topic.id && topicNames(topic).length > 0" class="flex flex-wrap gap-1 mt-1.5 ml-9">
              <span
                v-for="(alias, idx) in topicNames(topic)"
                :key="idx"
                class="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
              >
                {{ alias }}
              </span>
            </div>
            <!-- Inline topic editor (name + aliases) -->
            <div v-if="editingTopicId === topic.id" class="mt-2 space-y-2" @click.stop>
              <div>
                <Label class="text-xs text-muted-foreground mb-1 block">{{ $t('admin.quickReply.topicPrimaryName') }}</Label>
                <Input
                  v-model="editingTopicName"
                  type="text"
                  class="h-8 text-sm"
                  :placeholder="$t('admin.quickReply.topicNamePlaceholder')"
                  @keydown.esc="cancelTopicEdit"
                />
              </div>
              <div>
                <Label class="text-xs text-muted-foreground mb-1 block">{{ $t('admin.quickReply.topicAliases') }}</Label>
                <div class="space-y-1.5">
                  <div
                    v-for="(alias, idx) in editingTopicAliases"
                    :key="idx"
                    class="flex items-center gap-1.5"
                  >
                    <Input
                      v-model="editingTopicAliases[idx]"
                      type="text"
                      class="h-8 text-sm flex-1"
                      :placeholder="$t('admin.quickReply.topicAliasPlaceholder')"
                    />
                    <Button
                      size="sm"
                      variant="ghost"
                      class="h-8 w-8 shrink-0 p-0 text-destructive"
                      @click="removeAlias(idx)"
                    >
                      <X class="w-4 h-4" />
                    </Button>
                  </div>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  class="h-7 text-xs mt-1.5"
                  @click="addAlias"
                >
                  <Plus class="w-3.5 h-3.5 mr-1" />
                  {{ $t('admin.quickReply.addAlias') }}
                </Button>
              </div>
              <div class="flex gap-2 pt-1">
                <Button size="sm" class="h-7 text-xs" :disabled="topicSaving" @click="saveTopicEdit(topic)">
                  {{ $t('globals.terms.save') }}
                </Button>
                <Button size="sm" variant="outline" class="h-7 text-xs" @click="cancelTopicEdit">
                  {{ $t('globals.terms.cancel') }}
                </Button>
              </div>
            </div>
          </div>

          <div v-if="topics.length === 0" class="text-sm text-muted-foreground py-6 text-center">
            {{ $t('admin.quickReply.emptyTopics') }}
          </div>
        </div>

        <!-- Questions of the selected topic -->
        <div v-if="selectedTopic" class="space-y-3">
          <!-- Topic hint message -->
          <div class="rounded-lg border p-3 space-y-2">
            <Label class="text-xs font-medium">{{ $t('admin.quickReply.hintMessage') }}</Label>
            <Textarea
              v-model="selectedTopic.hint_message"
              :placeholder="$t('admin.quickReply.hintMessagePlaceholder')"
              rows="2"
              class="text-sm"
              @blur="saveHintMessage(selectedTopic)"
            />
            <Button
              size="sm"
              variant="outline"
              class="h-7 text-xs"
              :disabled="selectedTopic.__savingHint"
              @click="saveHintMessage(selectedTopic)"
            >
              {{ $t('globals.terms.save') }}
            </Button>
          </div>

          <div v-for="question in (selectedTopic.questions || [])" :key="question.id" class="rounded-lg border p-3 space-y-2">
            <div class="flex items-center gap-2">
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground disabled:opacity-20"
                :disabled="isFirstQuestion(question)"
                :title="$t('globals.terms.moveUp')"
                @click="moveQuestion(question, -1)"
              >
                <ChevronUp class="w-4 h-4" />
              </button>
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground disabled:opacity-20"
                :disabled="isLastQuestion(question)"
                :title="$t('globals.terms.moveDown')"
                @click="moveQuestion(question, 1)"
              >
                <ChevronDown class="w-4 h-4" />
              </button>
              <Label class="whitespace-nowrap text-xs shrink-0">{{ $t('admin.quickReply.question') }}</Label>
              <Input v-model="question.question" type="text" class="flex-1 h-8 text-sm" />
              <Button
                size="sm"
                variant="outline"
                class="h-8 shrink-0"
                :disabled="question.__saving"
                @click="saveQuestion(question)"
              >
                {{ $t('globals.terms.save') }}
              </Button>
              <button
                type="button"
                class="text-muted-foreground hover:text-destructive shrink-0"
                :title="$t('globals.terms.delete')"
                @click="openDeleteDialog('question', question)"
              >
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
            <div class="space-y-1">
              <Label class="whitespace-nowrap text-xs">{{ $t('admin.quickReply.answer') }}</Label>
              <Textarea v-model="question.answer" rows="2" class="text-sm" />
            </div>
          </div>

          <div v-if="!selectedTopic.questions?.length" class="text-sm text-muted-foreground py-6 text-center">
            {{ $t('admin.quickReply.emptyQuestions') }}
          </div>

          <Button size="sm" variant="outline" @click="addQuestion">
            <Plus class="w-4 h-4 mr-1.5" />
            {{ $t('admin.quickReply.addQuestion') }}
          </Button>
        </div>
      </div>
    </CardContent>
    </Card>

    <!-- Delete confirmation -->
    <Dialog :open="deleteDialog.open" @update:open="deleteDialog.open = $event">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{ deleteDialog.type === 'topic' ? $t('admin.quickReply.deleteTopicTitle') : $t('admin.quickReply.deleteQuestionTitle') }}</DialogTitle>
      </DialogHeader>
      <p class="text-sm text-muted-foreground">
        {{
          deleteDialog.type === 'topic'
            ? $t('admin.quickReply.deleteTopicConfirm', { name: deleteDialog.item?.name })
            : $t('admin.quickReply.deleteQuestionConfirm', { question: deleteDialog.item?.question })
        }}
      </p>
      <DialogFooter>
        <Button variant="outline" @click="deleteDialog.open = false">
          {{ $t('globals.terms.cancel') }}
        </Button>
        <Button variant="destructive" @click="performDelete">
          {{ $t('globals.terms.delete') }}
        </Button>
      </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Import dialog -->
    <Dialog :open="importDialog.open" @update:open="importDialog.open = $event">
    <DialogContent class="max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ $t('admin.quickReply.importTitle') }}</DialogTitle>
        <p class="text-sm text-muted-foreground">{{ $t('admin.quickReply.importDescription') }}</p>
      </DialogHeader>

      <!-- File selection -->
      <div
        class="flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed px-6 py-8 text-center cursor-pointer transition-colors hover:bg-muted/50"
        @click="fileInputRef?.click()"
        @dragover.prevent
        @drop.prevent="handleFileDrop"
      >
        <input ref="fileInputRef" type="file" accept=".xlsx,.xls" class="hidden" @change="handleFileChange" />
        <FileUp class="w-8 h-8 text-muted-foreground" />
        <p class="text-sm font-medium">{{ $t('admin.quickReply.importDropzone') }}</p>
        <p class="text-xs text-muted-foreground">{{ $t('admin.quickReply.importFileHint') }}</p>
      </div>

      <!-- Parse stats -->
      <div
        v-if="importDialog.fileName"
        class="flex items-center gap-2 rounded-lg bg-muted/50 px-3 py-2 text-sm"
      >
        <FileSpreadsheet class="w-4 h-4 text-muted-foreground shrink-0" />
        <span class="min-w-0 truncate">{{ importDialog.fileName }}</span>
        <span class="ml-auto shrink-0 text-muted-foreground">
          {{ $t('admin.quickReply.importParsed', { topics: importDialog.previewTopics.length, questions: totalImportedQuestions }) }}
        </span>
      </div>
      <div v-if="importDialog.errorRows.length" class="rounded-lg bg-destructive/10 px-3 py-2 text-xs text-destructive">
        {{ $t('admin.quickReply.importEmptyRows', { rows: importDialog.errorRows.join(', ') }) }}
      </div>

      <!-- Preview table -->
      <div v-if="importDialog.previewTopics.length" class="max-h-72 overflow-y-auto rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ $t('admin.quickReply.topicPrimaryName') }}</TableHead>
              <TableHead>{{ $t('admin.quickReply.topicAliases') }}</TableHead>
              <TableHead>{{ $t('admin.quickReply.hintMessage') }}</TableHead>
              <TableHead>{{ $t('admin.quickReply.question') }}</TableHead>
              <TableHead>{{ $t('admin.quickReply.answer') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="(row, idx) in importPreviewRows" :key="idx">
              <TableCell class="font-medium">{{ row.name }}</TableCell>
              <TableCell class="text-muted-foreground">{{ row.aliases }}</TableCell>
              <TableCell class="max-w-[10rem] truncate text-muted-foreground" :title="row.hint_message">{{ row.hint_message }}</TableCell>
              <TableCell class="max-w-[10rem] truncate" :title="row.question">{{ row.question }}</TableCell>
              <TableCell class="max-w-[14rem] truncate text-muted-foreground" :title="row.answer">{{ row.answer }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
      <div v-else-if="importDialog.fileName" class="py-6 text-center text-sm text-muted-foreground">
        {{ $t('admin.quickReply.importNoData') }}
      </div>

      <DialogFooter>
        <Button variant="outline" :disabled="importing" @click="closeImportDialog">
          {{ $t('admin.quickReply.importCancel') }}
        </Button>
        <Button :disabled="importing || importDialog.previewTopics.length === 0" @click="confirmImport">
          <Loader2 v-if="importing" class="w-4 h-4 mr-1.5 animate-spin" />
          {{ $t('admin.quickReply.importConfirm') }}
        </Button>
      </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import * as XLSX from 'xlsx'
import {
  Plus,
  Pencil,
  Trash2,
  ChevronUp,
  ChevronDown,
  Check,
  X,
  Download,
  Upload,
  FileUp,
  FileSpreadsheet,
  Loader2
} from 'lucide-vue-next'
import { Card, CardHeader, CardTitle, CardContent } from '@shared-ui/components/ui/card'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Label } from '@shared-ui/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@shared-ui/components/ui/dialog'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell
} from '@shared-ui/components/ui/table'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const props = defineProps({
  inboxId: {
    type: Number,
    required: true
  }
})

const { t } = useI18n()
const emitter = useEmitter()

const topics = ref([])
const selectedTopicId = ref(null)

function ensureQuestions(topic) {
  if (topic && !topic.questions) {
    topic.questions = []
  }
}
const addingTopic = ref(false)
const newTopicName = ref('')
const topicSaving = ref(false)
const editingTopicId = ref(null)
const editingTopicName = ref('')
const editingTopicAliases = ref([])
const deleteDialog = reactive({ open: false, type: 'topic', item: null })
const importDialog = reactive({ open: false, fileName: '', previewTopics: [], errorRows: [] })
const fileInputRef = ref(null)
const importing = ref(false)

const selectedTopic = computed(() => topics.value.find((topic) => topic.id === selectedTopicId.value) || null)

const showError = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
}

const showSuccess = () => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.savedSuccessfully') })
}

const loadTopics = async () => {
  const resp = await api.getQuickReplyTopics(props.inboxId)
  topics.value = resp.data.data
  topics.value.forEach(ensureQuestions)
  if (selectedTopicId.value && !topics.value.some((topic) => topic.id === selectedTopicId.value)) {
    selectedTopicId.value = null
  }
}

watch(
  () => props.inboxId,
  async () => {
    selectedTopicId.value = null
    try {
      await loadTopics()
    } catch (error) {
      showError(error)
    }
  }
)

const isFirst = (topic) => topics.value.indexOf(topic) === 0
const isLast = (topic) => topics.value.indexOf(topic) === topics.value.length - 1
const isFirstQuestion = (question) => (selectedTopic.value?.questions || []).indexOf(question) === 0
const isLastQuestion = (question) => {
  const list = selectedTopic.value?.questions || []
  return list.indexOf(question) === list.length - 1
}

const startAddTopic = () => {
  newTopicName.value = ''
  addingTopic.value = true
}

const createTopic = async () => {
  if (!newTopicName.value.trim() || topicSaving.value) return
  topicSaving.value = true
  try {
    const resp = await api.createQuickReplyTopic(props.inboxId, {
      name: newTopicName.value.trim(),
      hint_message: t('admin.quickReply.hintMessageDefault'),
      sort_order: topics.value.length
    })
    addingTopic.value = false
    await loadTopics()
    selectedTopicId.value = resp.data.data.id
    showSuccess()
  } catch (error) {
    showError(error)
  } finally {
    topicSaving.value = false
  }
}

// Returns the alias names (all names except the primary name).
const topicNames = (topic) => {
  const names = topic.names || []
  // Skip the first element which is the primary name (same as topic.name).
  return names.length > 1 ? names.slice(1) : []
}

const startTopicEdit = (topic) => {
  editingTopicId.value = topic.id
  editingTopicName.value = topic.name
  // Populate aliases (all names except the primary).
  const names = topic.names || []
  editingTopicAliases.value = names.length > 1 ? names.slice(1).map((n) => n) : []
}

const cancelTopicEdit = () => {
  editingTopicId.value = null
  editingTopicName.value = ''
  editingTopicAliases.value = []
}

const addAlias = () => {
  editingTopicAliases.value.push('')
}

const removeAlias = (idx) => {
  editingTopicAliases.value.splice(idx, 1)
}

const saveTopicEdit = async (topic) => {
  if (editingTopicId.value !== topic.id) return
  const name = editingTopicName.value.trim()
  if (!name) return
  topicSaving.value = true
  try {
    const names = [name, ...editingTopicAliases.value.map((a) => a.trim()).filter(Boolean)]
    const resp = await api.updateQuickReplyTopic(topic.id, {
      name,
      names,
      hint_message: topic.hint_message || '',
      sort_order: topic.sort_order
    })
    Object.assign(topic, resp.data.data)
    editingTopicId.value = null
    editingTopicName.value = ''
    editingTopicAliases.value = []
    showSuccess()
  } catch (error) {
    showError(error)
  } finally {
    topicSaving.value = false
  }
}

const saveHintMessage = async (topic) => {
  if (!topic || topic.__savingHint) return
  topic.__savingHint = true
  try {
    const resp = await api.updateQuickReplyTopic(topic.id, { name: topic.name, names: topic.names || [topic.name], hint_message: topic.hint_message || '', sort_order: topic.sort_order })
    topic.hint_message = resp.data.data.hint_message
    showSuccess()
  } catch (error) {
    showError(error)
  } finally {
    topic.__savingHint = false
  }
}

const moveTopic = async (topic, direction) => {
  const index = topics.value.indexOf(topic)
  const swapIndex = index + direction
  if (swapIndex < 0 || swapIndex >= topics.value.length) return
  const other = topics.value[swapIndex]
  try {
    await api.updateQuickReplyTopic(topic.id, { name: topic.name, names: topic.names || [topic.name], hint_message: topic.hint_message || '', sort_order: other.sort_order })
    await api.updateQuickReplyTopic(other.id, { name: other.name, names: other.names || [other.name], hint_message: other.hint_message || '', sort_order: topic.sort_order })
    await loadTopics()
  } catch (error) {
    showError(error)
  }
}

const openDeleteDialog = (type, item) => {
  deleteDialog.type = type
  deleteDialog.item = item
  deleteDialog.open = true
}

const performDelete = async () => {
  const { type, item } = deleteDialog
  deleteDialog.open = false
  try {
    if (type === 'topic') {
      await api.deleteQuickReplyTopic(item.id)
      await loadTopics()
    } else {
      await api.deleteQuickReplyQuestion(item.id)
      await loadTopics()
    }
    showSuccess()
  } catch (error) {
    showError(error)
  }
}

const addQuestion = async () => {
  try {
    const resp = await api.createQuickReplyQuestion(selectedTopicId.value, {
      question: t('admin.quickReply.newQuestionPlaceholder'),
      answer: '',
      sort_order: selectedTopic.value.questions?.length || 0
    })
    await loadTopics()
    selectedTopicId.value = resp.data.data.topic_id
  } catch (error) {
    showError(error)
  }
}

const saveQuestion = async (question) => {
  if (!question.question.trim()) return
  question.__saving = true
  try {
    const resp = await api.updateQuickReplyQuestion(question.id, {
      question: question.question.trim(),
      answer: question.answer,
      sort_order: question.sort_order
    })
    Object.assign(question, resp.data.data)
    showSuccess()
  } catch (error) {
    showError(error)
  } finally {
    question.__saving = false
  }
}

const moveQuestion = async (question, direction) => {
  const list = selectedTopic.value.questions || []
  const index = list.indexOf(question)
  const swapIndex = index + direction
  if (swapIndex < 0 || swapIndex >= list.length) return
  const other = list[swapIndex]
  try {
    await api.updateQuickReplyQuestion(question.id, {
      question: question.question,
      answer: question.answer,
      sort_order: other.sort_order
    })
    await api.updateQuickReplyQuestion(other.id, {
      question: other.question,
      answer: other.answer,
      sort_order: question.sort_order
    })
    await loadTopics()
  } catch (error) {
    showError(error)
  }
}

// ---------------------------------------------------------------------------
// Export / Import (Excel)
// ---------------------------------------------------------------------------
// Maps a header cell (any supported language) to its column key.
const normalizeHeader = (cell) => {
  const s = String(cell ?? '').trim().toLowerCase()
  if (['topic name', '主题名称', '主题'].includes(s)) return 'name'
  if (['aliases', '别名'].includes(s)) return 'aliases'
  if (['hint message', 'hint', '提示语', '提示文案'].includes(s)) return 'hint_message'
  if (['question', '问题'].includes(s)) return 'question'
  if (['answer', '答案'].includes(s)) return 'answer'
  return null
}

const exportTopics = async () => {
  try {
    const resp = await api.getQuickReplyTopics(props.inboxId)
    const list = resp.data.data || []
    if (list.length === 0) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('admin.quickReply.exportEmpty') })
      return
    }
    const aoa = [[t('admin.quickReply.topicPrimaryName'), t('admin.quickReply.topicAliases'), t('admin.quickReply.hintMessage'), t('admin.quickReply.question'), t('admin.quickReply.answer')]]
    for (const topic of list) {
      const aliases = (topic.names || []).slice(1).join('; ')
      const questions = topic.questions || []
      if (questions.length === 0) {
        aoa.push([topic.name, aliases, topic.hint_message || '', '', ''])
      } else {
        for (const q of questions) {
          aoa.push([topic.name, aliases, topic.hint_message || '', q.question, q.answer])
        }
      }
    }
    const worksheet = XLSX.utils.aoa_to_sheet(aoa)
    worksheet['!cols'] = [{ wch: 24 }, { wch: 20 }, { wch: 30 }, { wch: 30 }, { wch: 40 }]
    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, worksheet, 'Quick Replies')
    XLSX.writeFile(workbook, `quick-reply-${props.inboxId}.xlsx`)
  } catch (error) {
    showError(error)
  }
}

const totalImportedQuestions = computed(() =>
  importDialog.previewTopics.reduce((sum, topic) => sum + (topic.questions?.length || 0), 0)
)

const importPreviewRows = computed(() => {
  const rows = []
  for (const item of importDialog.previewTopics) {
    const aliases = (item.aliases || []).join('; ')
    if (!item.questions?.length) {
      rows.push({ name: item.name, aliases, hint_message: item.hint_message, question: '', answer: '' })
    } else {
      for (const q of item.questions) {
        rows.push({ name: item.name, aliases, hint_message: item.hint_message, question: q.question, answer: q.answer })
      }
    }
  }
  return rows
})

const openImportDialog = () => {
  importDialog.fileName = ''
  importDialog.previewTopics = []
  importDialog.errorRows = []
  importDialog.open = true
}

const closeImportDialog = () => {
  importDialog.open = false
  importDialog.fileName = ''
  importDialog.previewTopics = []
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

// Parses an Excel file into a list of topics (merged by topic name) plus the
// row numbers that were skipped because they are missing a topic name or question.
const parseFile = async (file) => {
  try {
    const buffer = await file.arrayBuffer()
    const workbook = XLSX.read(buffer, { type: 'array' })
    const sheet = workbook.Sheets[workbook.SheetNames[0]]
    const rows = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '' })
    if (!rows.length) {
      importDialog.previewTopics = []
      importDialog.errorRows = []
      importDialog.fileName = file.name
      return
    }

    const headerMap = new Map()
    rows[0].forEach((cell, index) => {
      const key = normalizeHeader(cell)
      if (key && !headerMap.has(key)) headerMap.set(key, index)
    })
    const nameIdx = headerMap.get('name')
    const aliasesIdx = headerMap.get('aliases')
    const hintIdx = headerMap.get('hint_message')
    const questionIdx = headerMap.get('question')
    const answerIdx = headerMap.get('answer')
    if (nameIdx === undefined || questionIdx === undefined) {
      importDialog.previewTopics = []
      importDialog.errorRows = []
      importDialog.fileName = file.name
      return
    }

    const topicMap = new Map()
    const order = []
    const errorRows = []
    for (let i = 1; i < rows.length; i++) {
      const rowNum = i + 1
      const row = rows[i]
      const name = nameIdx >= 0 ? String(row[nameIdx] ?? '').trim() : ''
      const question = questionIdx >= 0 ? String(row[questionIdx] ?? '').trim() : ''
      if (!name && !question) continue // skip fully empty rows
      if (!name) {
        errorRows.push(rowNum)
        continue
      }
      // A row with a topic name but an empty question is a valid topic without
      // questions (mirrors the export format).
      const rawAliases = aliasesIdx >= 0 ? String(row[aliasesIdx] ?? '') : ''
      const aliases = rawAliases.split(/[;；,，]/).map((a) => a.trim()).filter(Boolean)
      const hint = hintIdx >= 0 ? String(row[hintIdx] ?? '').trim() : ''
      const answer = answerIdx >= 0 ? String(row[answerIdx] ?? '') : ''

      let topic = topicMap.get(name)
      if (!topic) {
        topic = { name, aliases: new Set(), hint_message: hint, questions: [] }
        topicMap.set(name, topic)
        order.push(topic)
      }
      aliases.forEach((a) => topic.aliases.add(a))
      if (!topic.hint_message && hint) topic.hint_message = hint
      if (question) topic.questions.push({ question, answer })
    }

    importDialog.previewTopics = order.map((topic) => ({
      name: topic.name,
      aliases: [...topic.aliases],
      hint_message: topic.hint_message,
      questions: topic.questions
    }))
    importDialog.errorRows = errorRows
    importDialog.fileName = file.name
  } catch (error) {
    importDialog.previewTopics = []
    importDialog.errorRows = []
    importDialog.fileName = ''
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message || t('admin.quickReply.importError')
    })
  }
}

const confirmImport = async () => {
  if (importing.value || importDialog.previewTopics.length === 0) return
  importing.value = true
  try {
    const payload = {
      topics: importDialog.previewTopics.map((topic) => ({
        name: topic.name,
        names: topic.aliases || [],
        hint_message: topic.hint_message,
        questions: topic.questions.map((q) => ({ question: q.question, answer: q.answer }))
      }))
    }
    const resp = await api.importQuickReplyTopics(props.inboxId, payload)
    const stats = resp.data.data
    closeImportDialog()
    await loadTopics()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('admin.quickReply.importStats', {
        createdTopics: stats.created_topics,
        updatedTopics: stats.updated_topics,
        createdQuestions: stats.created_questions,
        updatedQuestions: stats.updated_questions
      })
    })
  } catch (error) {
    showError(error)
  } finally {
    importing.value = false
  }
}

onMounted(async () => {
  try {
    await loadTopics()
  } catch (error) {
    showError(error)
  }
})
</script>
