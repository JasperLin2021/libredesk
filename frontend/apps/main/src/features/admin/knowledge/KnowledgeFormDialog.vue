<template>
  <Dialog :open="open" @update:open="(v) => !v && cancel()">
    <DialogContent class="max-w-3xl">
      <DialogHeader>
        <DialogTitle>
          {{ article ? $t('admin.knowledgeBase.edit') : $t('admin.knowledgeBase.addArticle') }}
        </DialogTitle>
      </DialogHeader>

      <div class="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Label class="mb-1 block">{{ $t('admin.knowledgeBase.form.titleLabel') }}</Label>
            <Input
              v-model="form.title"
              :placeholder="$t('admin.knowledgeBase.form.titlePlaceholder')"
              :class="{ 'border-destructive': errors.title }"
            />
            <p v-if="errors.title" class="text-xs text-destructive mt-1">{{ errors.title }}</p>
          </div>
          <div>
            <Label class="mb-1 block">{{ $t('admin.knowledgeBase.form.authorLabel') }}</Label>
            <Input
              v-model="form.author"
              :placeholder="$t('admin.knowledgeBase.form.authorPlaceholder')"
            />
          </div>
        </div>

        <div>
          <Label class="mb-1 block">
            {{ $t('admin.knowledgeBase.form.categoryLabel') }}
            <span class="text-destructive">*</span>
          </Label>
          <button
            type="button"
            class="flex w-full items-center justify-between rounded-md border bg-background px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'border-destructive': errors.category_id }"
            @click="openCategoryPicker = true"
          >
            <span :class="{ 'text-muted-foreground': !selectedCategoryLabel }">
              {{ selectedCategoryLabel || $t('admin.knowledgeBase.form.categoryPlaceholder') }}
            </span>
            <ChevronDown class="h-4 w-4 text-muted-foreground" />
          </button>
          <p v-if="errors.category_id" class="text-xs text-destructive mt-1">
            {{ $t('admin.knowledgeBase.form.categoryRequired') }}
          </p>
        </div>

        <div>
          <Label class="mb-1 block">
            {{ $t('admin.knowledgeBase.form.contentLabel') }}
            <span class="text-destructive">*</span>
          </Label>
          <div class="rounded-md border bg-background">
            <TextEditor
              v-model:htmlContent="form.content"
              :placeholder="$t('admin.knowledgeBase.form.contentLabel')"
            />
          </div>
          <p v-if="errors.content" class="text-xs text-destructive mt-1">{{ errors.content }}</p>
        </div>

        <div>
          <Label class="mb-1 block">{{ $t('admin.knowledgeBase.form.attachmentLabel') }}</Label>
          <div class="flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="flex h-16 w-16 flex-col items-center justify-center rounded-md border border-dashed text-muted-foreground hover:text-foreground"
              :disabled="uploading"
              @click="triggerUpload"
            >
              <Loader2 v-if="uploading" class="h-4 w-4 animate-spin" />
              <Plus v-else class="h-4 w-4" />
            </button>
            <input ref="fileInput" type="file" class="hidden" multiple @change="onPickFiles" />
            <div
              v-for="att in form.attachments"
              :key="att.media_uuid"
              class="flex items-center gap-2 rounded-md border bg-muted/40 px-2 py-1 text-xs"
            >
              <Paperclip class="h-3.5 w-3.5" />
              <span class="max-w-[160px] truncate">{{ att.file_name || att.media_uuid }}</span>
              <button type="button" class="text-muted-foreground hover:text-destructive" @click="removeAttachment(att)">
                <X class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
          <p class="text-xs text-muted-foreground mt-1">
            {{ $t('admin.knowledgeBase.form.attachmentHint') }}
          </p>
        </div>
      </div>

      <DialogFooter class="flex items-center justify-between gap-2 sm:justify-between">
        <Button variant="outline" @click="cancel">
          {{ $t('admin.knowledgeBase.form.cancel') }}
        </Button>
        <div class="flex items-center gap-2">
          <Button :disabled="saving" @click="primarySubmit('publish')">
            <Loader2 v-if="saving" class="mr-1.5 h-4 w-4 animate-spin" />
            <Check v-else class="mr-1.5 h-4 w-4" />
            {{ $t('admin.knowledgeBase.form.publish') }}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button size="icon" :disabled="saving" class="px-2">
                <ChevronDown class="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem @select="primarySubmit('publish')">
                {{ $t('admin.knowledgeBase.form.publish') }}
              </DropdownMenuItem>
              <DropdownMenuItem @select="primarySubmit('preview')">
                <Eye class="mr-2 h-4 w-4" />
                {{ $t('admin.knowledgeBase.form.previewArticle') }}
              </DropdownMenuItem>
              <DropdownMenuItem @select="primarySubmit('draft')">
                <FileText class="mr-2 h-4 w-4" />
                {{ $t('admin.knowledgeBase.form.saveDraft') }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Two-level category picker -->
  <Dialog :open="openCategoryPicker" @update:open="(v) => (openCategoryPicker = v)">
    <DialogContent class="max-w-3xl">
      <DialogHeader>
        <DialogTitle>{{ $t('admin.knowledgeBase.form.categoryLabel') }}</DialogTitle>
      </DialogHeader>
      <div class="grid grid-cols-2 gap-3 min-h-[260px]">
        <div class="rounded-md border bg-muted/30 p-1 overflow-y-auto max-h-[320px]">
          <button
            v-for="root in roots"
            :key="root.id"
            type="button"
            class="flex w-full items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': pickedRootId === root.id }"
            @click="pickedRootId = root.id"
          >
            <span class="truncate">{{ root.name }}</span>
            <ChevronRight class="h-4 w-4" />
          </button>
          <div v-if="!roots.length" class="px-3 py-6 text-center text-xs text-muted-foreground">
            {{ $t('admin.knowledgeBase.manage.addCategory') }}
          </div>
        </div>
        <div class="rounded-md border bg-muted/30 p-1 overflow-y-auto max-h-[320px]">
          <button
            v-for="leaf in leavesOfPicked"
            :key="leaf.id"
            type="button"
            class="flex w-full items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': form.category_id === leaf.id }"
            @click="form.category_id = leaf.id"
          >
            <span class="truncate">{{ leaf.name }}</span>
            <Check v-if="form.category_id === leaf.id" class="h-4 w-4" />
          </button>
          <div v-if="pickedRootId && !leavesOfPicked.length" class="px-3 py-6 text-center text-xs text-muted-foreground">
            {{ $t('admin.knowledgeBase.manage.addChild') }}
          </div>
        </div>
      </div>
      <DialogFooter class="flex items-center justify-between gap-2 sm:justify-between">
        <span class="text-xs text-muted-foreground truncate">
          {{ selectedCategoryLabel
            ? $t('admin.knowledgeBase.form.selectResult', { name: selectedCategoryLabel })
            : '' }}
        </span>
        <Button @click="openCategoryPicker = false">
          {{ $t('admin.knowledgeBase.form.confirm') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem
} from '@shared-ui/components/ui/dropdown-menu'
import {
  Plus,
  ChevronDown,
  ChevronRight,
  Check,
  X,
  Eye,
  FileText,
  Loader2,
  Paperclip
} from 'lucide-vue-next'
import TextEditor from '@main/components/editor/TextEditor.vue'
import api, { http } from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'

const props = defineProps({
  open: { type: Boolean, default: false },
  article: { type: Object, default: null },
  categories: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:open', 'saved', 'preview'])

const { t } = useI18n()
const emitter = useEmitter()

const initialState = () => ({
  title: '',
  author: '',
  category_id: null,
  content: '',
  attachments: []
})

const form = reactive(initialState())
const errors = reactive({})
const saving = ref(false)
const uploading = ref(false)
const openCategoryPicker = ref(false)
const pickedRootId = ref(null)
const fileInput = ref(null)

const roots = computed(() => props.categories || [])
const leavesOfPicked = computed(() => {
  const root = roots.value.find((r) => r.id === pickedRootId.value)
  return root?.children || []
})
const selectedCategoryLabel = computed(() => {
  for (const root of roots.value) {
    if (form.category_id === root.id) return root.name
    for (const leaf of root.children || []) {
      if (form.category_id === leaf.id) return `${root.name} / ${leaf.name}`
    }
  }
  return ''
})

watch(
  () => props.open,
  (val) => {
    if (val) {
      Object.assign(form, initialState())
      Object.keys(errors).forEach((k) => delete errors[k])
      if (props.article) {
        form.title = props.article.title || ''
        form.author = props.article.author || ''
        form.category_id = props.article.category_id || null
        form.content = props.article.content || ''
        form.attachments = (props.article.attachments || []).map((a) => ({
          media_uuid: a.media_uuid,
          file_name: a.file_name,
          content_type: a.content_type,
          size: a.size
        }))
        if (props.article.parent_category_id) {
          pickedRootId.value = props.article.parent_category_id
        } else if (props.article.category_id) {
          const leaf = findLeafById(props.article.category_id)
          pickedRootId.value = leaf?.parent_id || null
        }
      }
    }
  }
)

const findLeafById = (id) => {
  for (const root of roots.value) {
    for (const leaf of root.children || []) {
      if (leaf.id === id) return leaf
    }
  }
  return null
}

const triggerUpload = () => fileInput.value?.click()

const onPickFiles = async (event) => {
  const files = Array.from(event.target.files || [])
  if (!files.length) return
  uploading.value = true
  try {
    for (const file of files) {
      const fd = new FormData()
      fd.append('file', file)
      const res = await http.post('/api/v1/media', fd, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      const payload = res?.data?.data || res?.data
      form.attachments.push({
        media_uuid: payload?.uuid,
        file_name: file.name,
        content_type: file.type,
        size: file.size
      })
    }
  } catch (err) {
    handleHTTPError(err)
  } finally {
    uploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

const removeAttachment = (att) => {
  form.attachments = form.attachments.filter((a) => a.media_uuid !== att.media_uuid)
}

const validate = () => {
  Object.keys(errors).forEach((k) => delete errors[k])
  if (!form.title.trim()) errors.title = t('validation.knowledgeTitleRequired')
  if (!form.content || !form.content.trim()) errors.content = t('admin.knowledgeBase.form.contentRequired')
  return Object.keys(errors).length === 0
}

const buildPayload = (status) => ({
  title: form.title.trim(),
  author: form.author.trim() || null,
  category_id: status === 'draft' ? null : form.category_id,
  content: form.content || '',
  status,
  attachments: form.attachments
})

const primarySubmit = async (mode) => {
  if (mode === 'preview') {
    if (!validate()) return
    emit('preview', {
      title: form.title.trim(),
      author: form.author.trim(),
      category: selectedCategoryLabel.value,
      content: form.content
    })
    return
  }
  if (!validate()) return
  if (mode === 'publish' && !form.category_id) {
    errors.category_id = true
    emitter.emit(EMITTER_EVENTS.TOAST, {
      variant: 'destructive',
      title: t('validation.knowledgeCategoryRequired')
    })
    return
  }
  saving.value = true
  try {
    const payload = buildPayload(mode === 'draft' ? 'draft' : 'published')
    let result
    if (props.article) {
      result = await api.updateKnowledgeArticle(props.article.id, payload)
    } else {
      result = await api.createKnowledgeArticle(payload)
    }
    emit('saved', result?.data || result)
    emit('update:open', false)
  } catch (err) {
    handleHTTPError(err)
  } finally {
    saving.value = false
  }
}

const cancel = () => emit('update:open', false)
</script>
