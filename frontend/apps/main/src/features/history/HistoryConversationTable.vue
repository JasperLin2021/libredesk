<template>
  <div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border bg-background">
    <Table class="min-h-0 flex-1">
      <TableHeader class="sticky top-0 z-10 bg-muted/70 backdrop-blur">
        <TableRow class="hover:bg-transparent">
          <TableHead scope="col" class="min-w-[14rem] pl-4">
            {{ $t('history.list.visitorName') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[9rem]">
            {{ $t('history.list.lastAgent') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[11rem]">
            {{ $t('history.list.pickedUpAt') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[11rem]">
            {{ $t('history.list.closedAt') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[8rem]">
            {{ $t('history.list.closedBy') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[8rem]">
            {{ $t('history.list.inbox') }}
          </TableHead>
          <TableHead scope="col" class="w-16 pr-4 text-right">
            <span class="sr-only">{{ $t('history.list.view') }}</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <!-- One row per conversation close event (row.closure_uuid is unique). -->
        <TableRow
          v-for="row in rows"
          :key="row.closure_uuid"
          class="cursor-pointer"
          :class="
            row.closure_uuid === selectedClosureUuid ? 'bg-primary/5 hover:bg-primary/10' : ''
          "
          :data-state="row.closure_uuid === selectedClosureUuid ? 'selected' : undefined"
          @click="$emit('select', row)"
        >
          <TableCell class="pl-4">
            <div class="flex min-w-0 items-center gap-2.5">
              <Avatar class="h-8 w-8 shrink-0">
                <AvatarImage :src="row.contact?.avatar_url || ''" class="object-cover" />
                <AvatarFallback class="bg-primary/10 text-[11px] font-semibold text-primary">
                  {{ initials(row) }}
                </AvatarFallback>
              </Avatar>
              <div class="min-w-0">
                <p class="truncate font-medium text-foreground" :title="visitorName(row)">
                  {{ visitorName(row) }}
                </p>
                <p
                  v-if="row.contact?.external_user_id"
                  class="truncate text-xs text-muted-foreground"
                  :title="row.contact.external_user_id"
                >
                  {{ row.contact.external_user_id }}
                </p>
              </div>
            </div>
          </TableCell>

          <TableCell class="text-foreground">
            <span :class="row.assigned_user_name ? '' : 'text-muted-foreground'">
              {{ lastAgent(row) }}
            </span>
          </TableCell>

          <TableCell class="whitespace-nowrap tabular-nums text-foreground">
            {{ formatDateTime(row.first_reply_at) }}
          </TableCell>

          <TableCell
            class="whitespace-nowrap tabular-nums font-medium text-foreground"
            :title="row.closure_content || ''"
          >
            {{ formatDateTime(row.closed_at) }}
          </TableCell>

          <TableCell class="truncate text-foreground" :title="row.closed_by_name || ''">
            <span :class="row.closed_by_name ? '' : 'text-muted-foreground'">
              {{ row.closed_by_name || $t('conversation.sidebar.notAvailable') }}
            </span>
          </TableCell>

          <TableCell class="truncate text-muted-foreground" :title="row.inbox_name || ''">
            {{ row.inbox_name || '-' }}
          </TableCell>

          <TableCell class="pr-4 text-right">
            <button
              type="button"
              class="cursor-pointer rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              :aria-label="$t('history.list.view') + ': ' + visitorName(row)"
              :aria-pressed="row.closure_uuid === selectedClosureUuid"
              @click.stop="$emit('select', row)"
            >
              <Eye class="h-4 w-4" />
            </button>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>

<script setup>
import { format, parseISO } from 'date-fns'
import { Eye } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'

defineProps({
  rows: {
    type: Array,
    default: () => []
  },
  // uuid of the closure message of the selected record.
  selectedClosureUuid: {
    type: String,
    default: ''
  }
})

defineEmits(['select'])

const visitorName = (row) => {
  const c = row?.contact || {}
  const name = `${c.first_name || ''} ${c.last_name || ''}`.trim()
  return name || c.email || c.external_user_id || '-'
}

const initials = (row) => {
  const c = row?.contact || {}
  const name = `${c.first_name || ''}${c.last_name || ''}`.trim()
  if (name) return name.substring(0, 2).toUpperCase()
  const fallback = (c.email || c.external_user_id || '').trim()
  return fallback ? fallback.substring(0, 2).toUpperCase() : '?'
}

const lastAgent = (row) => row?.assigned_user_name || '-'

const formatDateTime = (value) => {
  if (!value) return '-'
  try {
    return format(parseISO(value), 'yyyy-MM-dd HH:mm:ss')
  } catch (err) {
    return value
  }
}
</script>
