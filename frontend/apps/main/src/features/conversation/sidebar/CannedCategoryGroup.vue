<template>
  <div>
    <!-- Responses that live directly under this category. -->
    <CannedResponseList
      v-if="directResponses.length"
      :items="directResponses"
      :actions="actions"
      :search-empty="!!search"
      :search-empty-text="$t('conversation.sidebar.cannedResponses.searchEmpty')"
      :empty-text="$t('conversation.sidebar.cannedResponses.globalEmpty')"
      @select="emit('select', $event)"
      @action="emit('action', $event)"
    />
    <div
      v-else-if="!visibleChildren.length"
      class="px-3 py-2 text-xs text-muted-foreground"
    >
      {{
        search
          ? $t('conversation.sidebar.cannedResponses.searchEmpty')
          : $t('conversation.sidebar.cannedResponses.globalEmpty')
      }}
    </div>

    <!-- Sub-categories recurse. The nested tree matches the admin "公共
         常用语" page so the agent sees the same shape here. -->
    <div v-if="visibleChildren.length" class="pl-2 border-l border-border ml-3">
      <Accordion
        type="multiple"
        :model-value="nestedModel"
        @update:modelValue="onNestedChange"
        class="w-full"
      >
        <AccordionItem
          v-for="child in visibleChildren"
          :key="`sub-${child.id}`"
          :value="`sub-${child.id}`"
          class="border-b-0"
        >
          <AccordionTrigger class="px-2 py-1.5 text-xs font-normal hover:no-underline text-muted-foreground">
            {{ child.name }}
          </AccordionTrigger>
          <AccordionContent class="pb-1">
            <CannedCategoryGroup
              :category="child"
              :search="search"
              :actions="actions"
              @select="emit('select', $event)"
              @action="emit('action', $event)"
            />
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import CannedResponseList from './CannedResponseList.vue'
import { matchCategory } from './cannedTree.js'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'

// `CannedCategoryGroup` renders one node of the global canned-response
// tree. It shows the direct responses of the category with the shared
// list component and recursively renders its sub-categories.

const props = defineProps({
  category: { type: Object, required: true },
  search: { type: String, default: '' },
  actions: { type: Array, default: () => [] }
})
const emit = defineEmits(['select', 'action'])

const hasSearch = computed(() => !!(props.search || '').trim())

const directResponses = computed(() => {
  const responses = props.category.responses || []
  const q = (props.search || '').trim().toLowerCase()
  if (!q) return responses
  return responses.filter((r) => (r.content || '').toLowerCase().includes(q))
})

// While searching, sub-categories without a match anywhere in their subtree
// are pruned so the panel only shows branches that can contain a hit. The
// parent panel already pruned top-level categories the same way.
const visibleChildren = computed(() => {
  const children = props.category.children || []
  const q = (props.search || '').trim().toLowerCase()
  if (!q) return children
  return children.map((c) => matchCategory(c, q)).filter(Boolean)
})

// Sub-accordions are collapsed by default so the tree stays compact, but a
// search must force them open (on every nesting level) or matches stay
// hidden behind a collapsed trigger. Outside of a search the open state is
// the user's own, kept in a local ref.
const openSubs = ref([])
const subValues = computed(() => visibleChildren.value.map((c) => `sub-${c.id}`))
const nestedModel = computed(() =>
  hasSearch.value ? subValues.value : openSubs.value
)
const onNestedChange = (v) => {
  openSubs.value = v || []
}
</script>
