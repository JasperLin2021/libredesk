<template>
  <Dialog :open="open" @update:open="(v) => emit('update:open', v)">
    <DialogContent class="max-w-3xl">
      <DialogHeader>
        <DialogTitle>
          {{ $t('conversation.sidebar.knowledgeBase.filterTitle') }}
        </DialogTitle>
      </DialogHeader>

      <!-- Two-level picker: level-1 on the left ("全部分类" plus each
           knowledge base), level-2 on the right. Picking only a level-1
           node filters the whole knowledge base it owns. -->
      <div class="grid grid-cols-2 gap-3 min-h-[260px]">
        <div class="rounded-md border bg-muted/30 p-1 overflow-y-auto max-h-[320px]">
          <button
            type="button"
            class="flex w-full items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': draftParentId === null }"
            @click="pickRoot(null)"
          >
            <span class="truncate">
              {{ $t('conversation.sidebar.knowledgeBase.allCategories') }}
            </span>
            <Check v-if="draftParentId === null" class="h-4 w-4" />
          </button>
          <button
            v-for="root in roots"
            :key="root.id"
            type="button"
            class="flex w-full items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': draftParentId === root.id }"
            @click="pickRoot(root.id)"
          >
            <span class="truncate">{{ root.name }}</span>
            <Check v-if="draftParentId === root.id" class="h-4 w-4" />
          </button>
          <div
            v-if="!roots.length && !loading"
            class="px-3 py-6 text-center text-xs text-muted-foreground"
          >
            {{ $t('conversation.sidebar.knowledgeBase.noCategories') }}
          </div>
        </div>

        <div class="rounded-md border bg-muted/30 p-1 overflow-y-auto max-h-[320px]">
          <button
            v-if="draftParentId !== null"
            type="button"
            class="flex w-full items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': draftCategoryId === null }"
            @click="pickLeaf(null)"
          >
            <span class="truncate">
              {{ $t('conversation.sidebar.knowledgeBase.filterAll') }}
            </span>
            <Check v-if="draftCategoryId === null" class="h-4 w-4" />
          </button>
          <button
            v-for="leaf in leavesOfPicked"
            :key="leaf.id"
            type="button"
            class="flex w-full items-center justify-between rounded px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-primary text-primary-foreground': draftCategoryId === leaf.id }"
            @click="pickLeaf(leaf.id)"
          >
            <span class="truncate">{{ leaf.name }}</span>
            <Check v-if="draftCategoryId === leaf.id" class="h-4 w-4" />
          </button>
          <div
            v-if="draftParentId !== null && !leavesOfPicked.length && !loading"
            class="px-3 py-6 text-center text-xs text-muted-foreground"
          >
            {{ $t('conversation.sidebar.knowledgeBase.noSubCategories') }}
          </div>
        </div>
      </div>

      <DialogFooter class="flex items-center justify-between gap-2 sm:justify-between">
        <span class="text-xs text-muted-foreground truncate">
          {{ $t('conversation.sidebar.knowledgeBase.selectResult', { name: resultLabel }) }}
        </span>
        <div class="flex items-center gap-2">
          <Button variant="outline" @click="cancel">
            {{ $t('conversation.sidebar.knowledgeBase.cancel') }}
          </Button>
          <Button @click="confirm">
            {{ $t('conversation.sidebar.knowledgeBase.confirm') }}
          </Button>
        </div>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Check } from 'lucide-vue-next'

const { t } = useI18n()

const props = defineProps({
  open: { type: Boolean, default: false },
  categories: { type: Array, default: () => [] },
  // Currently applied filter, echoed back into the draft when reopening.
  parentCategoryId: { type: Number, default: null },
  categoryId: { type: Number, default: null },
  loading: { type: Boolean, default: false }
})
const emit = defineEmits(['update:open', 'apply'])

const roots = computed(() => props.categories || [])

// Draft selection so "取消" leaves the applied filter untouched.
const draftParentId = ref(props.parentCategoryId)
const draftCategoryId = ref(props.categoryId)

const leavesOfPicked = computed(() => {
  const root = roots.value.find((r) => r.id === draftParentId.value)
  return root?.children || []
})

const resultLabel = computed(() => {
  if (draftParentId.value === null) {
    return t('conversation.sidebar.knowledgeBase.allCategories')
  }
  const root = roots.value.find((r) => r.id === draftParentId.value)
  const rootName = root?.name || ''
  if (draftCategoryId.value === null) return rootName
  const leaf = (root?.children || []).find((l) => l.id === draftCategoryId.value)
  return leaf ? `${rootName} / ${leaf.name}` : rootName
})

// Reset the draft every time the dialog opens so it mirrors the applied filter.
watch(
  () => props.open,
  (val) => {
    if (!val) return
    draftParentId.value = props.parentCategoryId
    draftCategoryId.value = props.categoryId
  }
)

// Picking a level-1 node clears any level-2 selection ("whole knowledge base").
const pickRoot = (id) => {
  draftParentId.value = id
  draftCategoryId.value = null
}

const pickLeaf = (id) => {
  draftCategoryId.value = id
}

const cancel = () => emit('update:open', false)

const confirm = () => {
  emit('apply', {
    parentCategoryId: draftParentId.value,
    categoryId: draftCategoryId.value
  })
  emit('update:open', false)
}
</script>
