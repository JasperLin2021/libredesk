<template>
  <div class="flex flex-col h-full min-h-0">
    <!-- Search header. Filtering is done in-memory so it stays responsive
         even as the agent types fast. -->
    <div class="border-b border-border px-3 py-2">
      <div class="relative">
        <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          v-model="search"
          :placeholder="$t('conversation.sidebar.cannedResponses.searchPlaceholder')"
          class="h-8 pl-8 text-xs"
        />
      </div>
    </div>

    <div v-if="loading && !loaded" class="flex-1 flex items-center justify-center text-xs text-muted-foreground">
      {{ $t('globals.messages.loading') }}
    </div>
    <div v-else class="flex-1 overflow-y-auto min-h-0">
      <Accordion
        type="multiple"
        :model-value="accordionModel"
        @update:modelValue="onAccordionChange"
        class="w-full"
      >
        <!-- Pinned canned responses -->
        <AccordionItem v-if="!hasSearch || filteredPinned.length" value="pinned" class="border-b-0">
          <AccordionTrigger class="px-3 py-2 text-xs font-medium hover:no-underline">
            <span class="flex items-center gap-1.5">
              <Pin class="h-3.5 w-3.5 text-muted-foreground" />
              {{ $t('conversation.sidebar.cannedResponses.pinned') }}
            </span>
          </AccordionTrigger>
          <AccordionContent class="pb-1">
            <CannedResponseList
              :items="filteredPinned"
              :empty-text="$t('conversation.sidebar.cannedResponses.pinnedEmpty')"
              :search-empty="hasSearch"
              :search-empty-text="$t('conversation.sidebar.cannedResponses.searchEmpty')"
              :actions="pinnedActions"
              @select="insertIntoReply"
              @action="onAction"
            />
          </AccordionContent>
        </AccordionItem>

        <!-- Personal canned responses: add / edit / delete / pin -->
        <AccordionItem v-if="!hasSearch || filteredPersonal.length" value="personal" class="border-b-0">
          <AccordionTrigger class="px-3 py-2 text-xs font-medium hover:no-underline">
            <span class="flex-1 text-left">{{ $t('conversation.sidebar.cannedResponses.personal') }}</span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="h-6 w-6 text-muted-foreground hover:text-foreground"
              @click.stop="openAddDialog()"
            >
              <Plus class="h-3.5 w-3.5" />
              <span class="sr-only">{{ $t('conversation.sidebar.cannedResponses.add') }}</span>
            </Button>
          </AccordionTrigger>
          <AccordionContent class="pb-1">
            <CannedResponseList
              :items="filteredPersonal"
              :empty-text="$t('conversation.sidebar.cannedResponses.personalEmpty')"
              :search-empty="hasSearch"
              :search-empty-text="$t('conversation.sidebar.cannedResponses.searchEmpty')"
              :actions="personalActions"
              @select="insertIntoReply"
              @action="onAction"
            />
          </AccordionContent>
        </AccordionItem>

        <!-- Global canned responses, rendered as one accordion section per
             top-level category so the tree mirrors the admin "公共常用语" page. -->
        <AccordionItem
          v-for="cat in filteredGlobalTree"
          :key="`cat-${cat.id}`"
          :value="`cat-${cat.id}`"
          class="border-b-0"
        >
          <AccordionTrigger class="px-3 py-2 text-xs font-medium hover:no-underline">
            {{ cat.name }}
          </AccordionTrigger>
          <AccordionContent class="pb-1">
            <CannedCategoryGroup
              :category="cat"
              :search="search"
              :actions="globalActions"
              @select="insertIntoReply"
              @action="onAction"
            />
          </AccordionContent>
        </AccordionItem>
      </Accordion>
      <!-- Global "no results" state: while searching, every group without a
           match is hidden above, so an empty list here means the whole panel
           found nothing. -->
      <div
        v-if="hasSearch && !hasAnyResult"
        class="flex items-center justify-center py-8 text-xs text-muted-foreground"
      >
        {{ $t('conversation.sidebar.cannedResponses.searchEmpty') }}
      </div>
    </div>

    <!-- Add / edit personal canned response dialog -->
    <Dialog :open="editDialog.open" @update:open="(v) => (v ? (editDialog.open = true) : closeEditDialog())">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {{
              editDialog.id
                ? $t('conversation.sidebar.cannedResponses.edit')
                : $t('conversation.sidebar.cannedResponses.add')
            }}
          </DialogTitle>
        </DialogHeader>
        <Textarea
          v-model="editDialog.content"
          rows="5"
          :placeholder="$t('conversation.sidebar.cannedResponses.addPlaceholder')"
          class="text-sm"
        />
        <DialogFooter class="gap-2">
          <Button variant="outline" @click="closeEditDialog">
            {{ $t('globals.terms.cancel') }}
          </Button>
          <Button :disabled="!editDialog.content.trim()" @click="savePersonal">
            {{ $t('globals.terms.save') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete confirmation dialog -->
    <AlertDialog :open="deleteDialog.open" @update:open="(v) => (deleteDialog.open = v)">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {{ $t('conversation.sidebar.cannedResponses.delete') }}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {{ $t('conversation.sidebar.cannedResponses.deleteConfirm') }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.terms.cancel') }}</AlertDialogCancel>
          <AlertDialogAction @click="confirmDelete">
            {{ $t('globals.terms.delete') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted } from 'vue'
import { Search, Plus, Pin } from 'lucide-vue-next'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import CannedResponseList from './CannedResponseList.vue'
import CannedCategoryGroup from './CannedCategoryGroup.vue'
import { filterCannedTree } from './cannedTree.js'
import { useCannedResponsesStore, PIN_SOURCE } from '@/stores/cannedResponses'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const store = useCannedResponsesStore()
const emitter = useEmitter()

const search = ref('')
const hasSearch = computed(() => search.value.trim().length > 0)

// The pin and personal sections are open by default so a fresh agent sees
// their own content without having to expand the accordion. Global
// categories are closed to keep the first paint compact.
const openSections = ref(['pinned', 'personal'])
const onAccordionChange = (val) => {
  openSections.value = val || []
}

const loading = computed(() => store.loading)
const loaded = computed(() => store.loaded)
const personal = computed(() => store.personal)
const pinned = computed(() => store.pinned)
const globalTree = computed(() => store.globalTree)

const filteredPinned = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return pinned.value
  return pinned.value.filter((p) => (p.content || '').toLowerCase().includes(q))
})

const filteredPersonal = computed(() => {
  const q = search.value.trim().toLowerCase()
  const list = q
    ? personal.value.filter((p) => (p.content || '').toLowerCase().includes(q))
    : personal.value
  // Tag rows with their source so `onAction` can tell personal entries
  // apart from global responses without duplicating lookup logic in the
  // presentational list component.
  return list.map((p) => ({ ...p, kind: PIN_SOURCE.PERSONAL }))
})

// Global tree pruned to branches containing matches while searching.
const filteredGlobalTree = computed(() => filterCannedTree(globalTree.value, search.value))

const hasAnyResult = computed(
  () =>
    filteredPinned.value.length > 0 ||
    filteredPersonal.value.length > 0 ||
    filteredGlobalTree.value.length > 0
)

// While searching, matches inside collapsed sections would be invisible and
// make the search look broken. Instead every surviving section is force
// opened (the tree is already pruned, so only matching groups remain).
const searchSectionValues = computed(() => {
  const vals = ['pinned', 'personal']
  const walk = (cats) => {
    ;(cats || []).forEach((c) => {
      vals.push(`cat-${c.id}`)
      walk(c.children)
    })
  }
  walk(filteredGlobalTree.value)
  return vals
})
const accordionModel = computed(() =>
  hasSearch.value ? searchSectionValues.value : openSections.value
)

onMounted(() => {
  store.fetch()
})

// Action descriptors consumed by the leaf list / category components. The
// `match` predicate decides whether the action is offered for a given item.
// Centralising them here keeps the per-section lists purely presentational.
const personalActions = computed(() => [
  {
    key: 'edit',
    label: t('conversation.sidebar.cannedResponses.edit'),
    icon: 'edit',
    match: () => true
  },
  {
    key: 'delete',
    label: t('conversation.sidebar.cannedResponses.delete'),
    icon: 'trash',
    match: () => true
  },
  {
    key: 'pin',
    label: t('conversation.sidebar.cannedResponses.pin'),
    icon: 'pin',
    match: (item) =>
      !pinned.value.some(
        (p) => p.source === PIN_SOURCE.PERSONAL && p.response_id === item.id
      )
  }
])

const pinnedActions = computed(() => [
  {
    key: 'unpin',
    label: t('conversation.sidebar.cannedResponses.unpin'),
    icon: 'pin-off',
    match: () => true
  }
])

const globalActions = computed(() => [
  {
    key: 'pin',
    label: t('conversation.sidebar.cannedResponses.pin'),
    icon: 'pin',
    // A global response is pinnable only when the agent has not already
    // pinned the same id. The store will silently no-op otherwise, but the
    // UI hides the action to make the affordance clear.
    match: (item) =>
      !pinned.value.some(
        (p) => p.source === PIN_SOURCE.GLOBAL && p.response_id === item.id
      )
  }
])

// Click a row → fill the reply box. Reusing the existing
// `COPILOT_INSERT_REPLY` event keeps the new panel decoupled from the
// rich-text editor and matches the Copilot "Insert into reply" behaviour.
const insertIntoReply = (item) => {
  const content = item?.content
  if (!content) return
  emitter.emit(EMITTER_EVENTS.COPILOT_INSERT_REPLY, content)
}

const editDialog = reactive({ open: false, id: null, content: '' })
const openAddDialog = () => {
  editDialog.id = null
  editDialog.content = ''
  editDialog.open = true
}
const openEditDialog = (item) => {
  editDialog.id = item.id
  editDialog.content = item.content || ''
  editDialog.open = true
}
const closeEditDialog = () => {
  editDialog.open = false
  editDialog.id = null
  editDialog.content = ''
}
const savePersonal = async () => {
  const content = editDialog.content.trim()
  if (!content) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('conversation.sidebar.cannedResponses.contentRequired')
    })
    return
  }
  if (content.length > 1000) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('conversation.sidebar.cannedResponses.contentTooLong')
    })
    return
  }
  if (editDialog.id) {
    const updated = await store.updatePersonal(editDialog.id, content)
    if (updated) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('conversation.sidebar.cannedResponses.saved')
      })
      closeEditDialog()
    }
  } else {
    const created = await store.createPersonal(content)
    if (created) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('conversation.sidebar.cannedResponses.saved')
      })
      closeEditDialog()
    }
  }
}

const deleteDialog = reactive({ open: false, id: null })
const requestDelete = (item) => {
  deleteDialog.id = item.id
  deleteDialog.open = true
}
const confirmDelete = async () => {
  if (!deleteDialog.id) return
  const ok = await store.deletePersonal(deleteDialog.id)
  if (ok) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('conversation.sidebar.cannedResponses.deleted')
    })
  }
  deleteDialog.open = false
  deleteDialog.id = null
}

const onAction = async ({ action, item }) => {
  if (!action || !item) return
  switch (action.key) {
    case 'edit':
      openEditDialog(item)
      break
    case 'delete':
      requestDelete(item)
      break
    case 'pin': {
      const source = item.kind === PIN_SOURCE.PERSONAL ? PIN_SOURCE.PERSONAL : PIN_SOURCE.GLOBAL
      const ok = await store.pin(source, item.id)
      if (ok) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          description: t('conversation.sidebar.cannedResponses.pinnedSuccess')
        })
      }
      break
    }
    case 'unpin': {
      // Pinned rows carry the pair (source, response_id); the underlying
      // response id lives on `response_id`, not `id`.
      const source = item.source === PIN_SOURCE.PERSONAL ? PIN_SOURCE.PERSONAL : PIN_SOURCE.GLOBAL
      const ok = await store.unpin(source, item.response_id ?? item.id)
      if (!ok) break
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('conversation.sidebar.cannedResponses.unpinnedSuccess')
      })
      break
    }
  }
}

</script>
