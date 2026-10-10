<template>
  <Dialog :open="open" @update:open="(v) => !v && close()">
    <DialogContent class="max-w-4xl">
      <DialogHeader>
        <DialogTitle>{{ $t('admin.knowledgeBase.manage.title') }}</DialogTitle>
      </DialogHeader>
      <div class="grid grid-cols-2 gap-3 min-h-[320px]">
        <!-- Level-1 categories -->
        <div class="rounded-md border bg-muted/30 p-1 overflow-y-auto max-h-[420px]">
          <div
            v-for="root in categories"
            :key="root.id"
            class="flex items-center justify-between rounded px-3 py-2 text-sm cursor-pointer hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': selectedRootId === root.id }"
            @click="selectedRootId = root.id"
          >
            <span class="truncate flex-1">{{ root.name }}</span>
            <div class="flex items-center gap-1">
              <button
                type="button"
                class="rounded p-1 hover:bg-background/50"
                :title="$t('admin.knowledgeBase.manage.edit')"
                @click.stop="startEditCategory(root)"
              >
                <Pencil class="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                class="rounded p-1 hover:bg-background/50"
                :title="$t('admin.knowledgeBase.manage.delete')"
                @click.stop="askDelete(root)"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
          <button
            type="button"
            class="mt-2 w-full text-left text-sm text-primary px-3 py-1.5 hover:underline"
            @click="startAddRoot"
          >
            + {{ $t('admin.knowledgeBase.manage.addCategory') }}
          </button>
        </div>

        <!-- Level-2 children of the selected level-1 -->
        <div class="rounded-md border bg-muted/30 p-1 overflow-y-auto max-h-[420px]">
          <div v-if="!selectedRootId" class="px-3 py-6 text-center text-xs text-muted-foreground">
            —
          </div>
          <template v-else>
            <div
              v-for="leaf in childrenOfSelected"
              :key="leaf.id"
              class="flex items-center justify-between rounded px-3 py-2 text-sm"
              :class="{ 'bg-primary text-primary-foreground': selectedLeafId === leaf.id }"
            >
              <span class="truncate flex-1" @click="selectedLeafId = leaf.id">{{ leaf.name }}</span>
              <div class="flex items-center gap-1">
                <button
                  type="button"
                  class="rounded p-1 hover:bg-background/50"
                  :title="$t('admin.knowledgeBase.manage.edit')"
                  @click="startEditCategory(leaf)"
                >
                  <Pencil class="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  class="rounded p-1 hover:bg-background/50"
                  :title="$t('admin.knowledgeBase.manage.delete')"
                  @click="askDelete(leaf)"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
            <button
              type="button"
              class="mt-2 w-full text-left text-sm text-primary px-3 py-1.5 hover:underline"
              @click="startAddChild"
            >
              + {{ $t('admin.knowledgeBase.manage.addChild') }}
            </button>
          </template>
        </div>
      </div>
      <DialogFooter>
        <Button @click="close">{{ $t('admin.knowledgeBase.manage.confirm') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Edit / add inline dialog -->
  <Dialog :open="editing.open" @update:open="(v) => !v && cancelEdit()">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle>
          {{ editing.id ? $t('admin.knowledgeBase.manage.edit') : $t('admin.knowledgeBase.manage.add') }}
        </DialogTitle>
      </DialogHeader>
      <div class="space-y-2">
        <Label>{{ editing.parentId ? $t('admin.knowledgeBase.manage.addChild') : $t('admin.knowledgeBase.manage.addCategory') }}</Label>
        <Input v-model="editing.name" />
      </div>
      <DialogFooter>
        <Button variant="outline" @click="cancelEdit">{{ $t('admin.knowledgeBase.form.cancel') }}</Button>
        <Button :disabled="editing.saving" @click="confirmEdit">
          <Loader2 v-if="editing.saving" class="mr-1 h-4 w-4 animate-spin" />
          {{ $t('admin.knowledgeBase.form.confirm') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- Delete confirm -->
  <Dialog :open="deleting.open" @update:open="(v) => !v && cancelDelete()">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ $t('admin.knowledgeBase.manage.delete') }}</DialogTitle>
      </DialogHeader>
      <p class="text-sm">{{ $t('admin.knowledgeBase.manage.confirmDelete', { name: deleting.name }) }}</p>
      <DialogFooter>
        <Button variant="outline" @click="cancelDelete">{{ $t('admin.knowledgeBase.form.cancel') }}</Button>
        <Button variant="destructive" :disabled="deleting.busy" @click="confirmDelete">
          <Loader2 v-if="deleting.busy" class="mr-1 h-4 w-4 animate-spin" />
          {{ $t('admin.knowledgeBase.manage.delete') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, reactive, watch } from 'vue'
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
import { Pencil, Trash2, Loader2 } from 'lucide-vue-next'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'

const props = defineProps({
  open: { type: Boolean, default: false },
  categories: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:open', 'changed'])

const { t } = useI18n()
const emitter = useEmitter()

const selectedRootId = ref(null)
const selectedLeafId = ref(null)

const editing = reactive({ open: false, id: null, name: '', parentId: null, saving: false })
const deleting = reactive({ open: false, id: null, name: '', busy: false })

const childrenOfSelected = computed(() => {
  const root = props.categories.find((r) => r.id === selectedRootId.value)
  return root?.children || []
})

watch(
  () => props.open,
  (val) => {
    if (val) {
      selectedRootId.value = props.categories[0]?.id || null
      selectedLeafId.value = null
    }
  }
)

const close = () => emit('update:open', false)

const startAddRoot = () => {
  editing.id = null
  editing.name = ''
  editing.parentId = null
  editing.open = true
}

const startAddChild = () => {
  editing.id = null
  editing.name = ''
  editing.parentId = selectedRootId.value
  editing.open = true
}

const startEditCategory = (cat) => {
  editing.id = cat.id
  editing.name = cat.name
  editing.parentId = cat.parent_id || null
  editing.open = true
}

const cancelEdit = () => {
  editing.open = false
  editing.id = null
  editing.name = ''
}

const confirmEdit = async () => {
  if (!editing.name.trim()) return
  editing.saving = true
  try {
    if (editing.id) {
      await api.updateKnowledgeCategory(editing.id, {
        name: editing.name.trim(),
        parent_id: editing.parentId
      })
    } else {
      await api.createKnowledgeCategory({
        name: editing.name.trim(),
        parent_id: editing.parentId
      })
    }
    emitter.emit(EMITTER_EVENTS.TOAST, { title: t('globals.terms.saved') })
    cancelEdit()
    emit('changed')
  } catch (err) {
    handleHTTPError(err)
  } finally {
    editing.saving = false
  }
}

const askDelete = (cat) => {
  deleting.id = cat.id
  deleting.name = cat.name
  deleting.open = true
}

const cancelDelete = () => {
  deleting.open = false
  deleting.id = null
}

const confirmDelete = async () => {
  deleting.busy = true
  try {
    await api.deleteKnowledgeCategory(deleting.id)
    emitter.emit(EMITTER_EVENTS.TOAST, { title: t('globals.terms.deleted') })
    cancelDelete()
    emit('changed')
  } catch (err) {
    handleHTTPError(err)
  } finally {
    deleting.busy = false
  }
}
</script>
