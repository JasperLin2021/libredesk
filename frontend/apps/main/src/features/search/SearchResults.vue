<template>
  <div class="max-w-5xl mx-auto p-6 min-h-screen">
    <Tabs :default-value="defaultTab" v-model="activeTab">
      <TabsList class="grid w-full mb-6" :class="tabsGridClass">
        <TabsTrigger v-for="(items, type) in results" :key="type" :value="type">
          {{ $t(tabLabelKeys[type], 2) }} ({{ items.length }})
        </TabsTrigger>
      </TabsList>

      <TabsContent v-for="(items, type) in results" :key="type" :value="type" class="mt-0">
        <div class="bg-background rounded-md border overflow-hidden">
          <!-- No results message -->
          <div v-if="items.length === 0" class="p-8 text-center text-muted-foreground">
            <div class="text-lg font-medium mb-2">
              {{ $t('globals.messages.noResultsFound') }}
            </div>
            <div class="text-sm">{{ $t('search.adjustSearchTerms') }}</div>
          </div>

          <!-- Results list -->
          <div v-else class="divide-y divide-border">
            <div
              v-for="item in items"
              :key="item.id || item.uuid"
              class="p-6 hover:bg-accent/50 transition duration-200 ease-in-out group"
            >
              <button
                v-if="interactive"
                type="button"
                class="block w-full text-left"
                @click="$emit('select', {
                  type,
                  item,
                  conversationUuid: type === 'conversations' ? item.uuid : item.conversation_uuid,
                  messageUuid: type === 'conversations' ? null : (item.message_uuid || item.uuid)
                })"
              >
                <ResultItemBody :type="type" :item="item" />
              </button>
              <router-link
                v-else
                :to="{
                  name: 'inbox-conversation',
                  params: {
                    uuid: type === 'conversations' ? item.uuid : item.conversation_uuid,
                    type: 'assigned'
                  }
                }"
                class="block"
              >
                <ResultItemBody :type="type" :item="item" />
              </router-link>
            </div>
          </div>
        </div>
      </TabsContent>
    </Tabs>
  </div>
</template>
<script setup>
import { computed, ref, watch } from 'vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import ResultItemBody from './ResultItemBody.vue'

const tabLabelKeys = {
  conversations: 'globals.terms.conversation',
  messages: 'globals.terms.message'
}

const props = defineProps({
  results: {
    type: Object,
    required: true
  },
  // When true, clicking a result emits `select` instead of navigating to the
  // conversation page.
  interactive: {
    type: Boolean,
    default: false
  }
})

defineEmits(['select'])

// Get the first available tab as default
const defaultTab = computed(() => {
  const types = Object.keys(props.results)
  return types.length > 0 ? types[0] : ''
})

const activeTab = ref('')

// Watch for changes in results and set the first tab as active
watch(
  () => props.results,
  (newResults) => {
    const types = Object.keys(newResults)
    if (types.length > 0 && !activeTab.value) {
      activeTab.value = types[0]
    }
  },
  { immediate: true }
)

// Dynamic grid class based on number of tabs
const tabsGridClass = computed(() => {
  const tabCount = Object.keys(props.results).length
  if (tabCount <= 2) return 'grid-cols-2'
  if (tabCount <= 3) return 'grid-cols-3'
  if (tabCount <= 4) return 'grid-cols-4'
  return 'grid-cols-5'
})
</script>
