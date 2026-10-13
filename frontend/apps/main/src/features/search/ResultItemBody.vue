<template>
  <div class="flex justify-between items-start">
    <div class="flex-grow">
      <!-- Reference number and status -->
      <div
        class="text-sm font-semibold mb-2 text-muted-foreground group-hover:text-primary transition duration-200 flex items-center gap-2"
      >
        <span>
          #{{
            type === 'conversations'
              ? item.reference_number
              : item.conversation_reference_number
          }}
        </span>
        <Badge variant="outline" class="text-xs font-medium">
          {{ type === 'conversations' ? item.status : item.conversation_status }}
        </Badge>
      </div>

      <!-- Content -->
      <div
        class="text-foreground font-medium mb-2 text-lg group-hover:text-primary transition duration-200"
      >
        {{ truncateText(type === 'conversations' ? item.subject : item.text_content, 100) }}
      </div>

      <!-- Timestamp -->
      <div class="text-sm text-muted-foreground flex items-center">
        <ClockIcon class="h-4 w-4 mr-1" />
        {{ formatDate(type === 'conversations' ? item.created_at : item.conversation_created_at) }}
      </div>
    </div>

    <!-- Right arrow icon -->
    <div
      class="bg-secondary rounded-full p-2 group-hover:bg-primary transition duration-200"
    >
      <ChevronRightIcon
        class="h-5 w-5 text-secondary-foreground group-hover:text-primary-foreground"
        aria-hidden="true"
      />
    </div>
  </div>
</template>

<script setup>
import { ChevronRightIcon, ClockIcon } from 'lucide-vue-next'
import { format, parseISO } from 'date-fns'
import { Badge } from '@shared-ui/components/ui/badge'

defineProps({
  type: {
    type: String,
    required: true
  },
  item: {
    type: Object,
    required: true
  }
})

const formatDate = (dateString) => {
  const date = parseISO(dateString)
  return format(date, 'MMM d, yyyy HH:mm')
}

const truncateText = (text, length) => {
  if (!text) return ''
  if (text.length <= length) return text
  return text.slice(0, length) + '...'
}
</script>
