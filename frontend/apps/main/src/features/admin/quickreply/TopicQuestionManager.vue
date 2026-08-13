<template>
  <div>
    <Card>
    <CardHeader class="flex-row items-center justify-between space-y-0">
      <CardTitle>{{ $t('admin.quickReply.topics') }}</CardTitle>
      <Button size="sm" variant="outline" @click="startAddTopic">
        <Plus class="w-4 h-4 mr-1.5" />
        {{ $t('admin.quickReply.addTopic') }}
      </Button>
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
              'group flex items-center gap-1.5 rounded-lg border px-3 py-2 cursor-pointer transition-colors',
              topic.id === selectedTopicId ? 'border-primary bg-primary/5' : 'hover:bg-muted'
            ]"
            @click="selectedTopicId = topic.id"
          >
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
            <input
              v-if="editingTopicId === topic.id"
              v-model="editingTopicName"
              type="text"
              class="flex-1 min-w-0 text-sm bg-transparent outline-none border-b border-primary"
              @keydown.enter="saveTopicName(topic)"
              @keydown.esc="cancelTopicEdit"
              @blur="saveTopicName(topic)"
              @click.stop
            />
            <span v-else class="flex-1 min-w-0 truncate text-sm font-medium">{{ topic.name }}</span>
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

          <div v-if="topics.length === 0" class="text-sm text-muted-foreground py-6 text-center">
            {{ $t('admin.quickReply.emptyTopics') }}
          </div>
        </div>

        <!-- Questions of the selected topic -->
        <div v-if="selectedTopic" class="space-y-3">
          <div v-for="question in selectedTopic.questions" :key="question.id" class="rounded-lg border p-3 space-y-2">
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

          <div v-if="!selectedTopic.questions.length" class="text-sm text-muted-foreground py-6 text-center">
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
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { Plus, Pencil, Trash2, ChevronUp, ChevronDown, Check, X } from 'lucide-vue-next'
import { Card, CardHeader, CardTitle, CardContent } from '@shared-ui/components/ui/card'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Label } from '@shared-ui/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@shared-ui/components/ui/dialog'
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
const addingTopic = ref(false)
const newTopicName = ref('')
const topicSaving = ref(false)
const editingTopicId = ref(null)
const editingTopicName = ref('')
const deleteDialog = reactive({ open: false, type: 'topic', item: null })

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
const isFirstQuestion = (question) => selectedTopic.value?.questions.indexOf(question) === 0
const isLastQuestion = (question) =>
  selectedTopic.value?.questions.indexOf(question) === selectedTopic.value?.questions.length - 1

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

const startTopicEdit = (topic) => {
  editingTopicId.value = topic.id
  editingTopicName.value = topic.name
}

const cancelTopicEdit = () => {
  editingTopicId.value = null
  editingTopicName.value = ''
}

const saveTopicName = async (topic) => {
  if (editingTopicId.value !== topic.id) return
  const name = editingTopicName.value.trim()
  editingTopicId.value = null
  if (!name || name === topic.name) return
  try {
    const resp = await api.updateQuickReplyTopic(topic.id, { name, sort_order: topic.sort_order })
    topic.name = resp.data.data.name
    showSuccess()
  } catch (error) {
    showError(error)
  }
}

const moveTopic = async (topic, direction) => {
  const index = topics.value.indexOf(topic)
  const swapIndex = index + direction
  if (swapIndex < 0 || swapIndex >= topics.value.length) return
  const other = topics.value[swapIndex]
  try {
    await api.updateQuickReplyTopic(topic.id, { name: topic.name, sort_order: other.sort_order })
    await api.updateQuickReplyTopic(other.id, { name: other.name, sort_order: topic.sort_order })
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
      sort_order: selectedTopic.value.questions.length
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
  const list = selectedTopic.value.questions
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

onMounted(async () => {
  try {
    await loadTopics()
  } catch (error) {
    showError(error)
  }
})
</script>
