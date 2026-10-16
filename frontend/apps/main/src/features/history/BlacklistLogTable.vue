<template>
  <div
    class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border bg-background"
  >
    <Table class="min-h-0 flex-1">
      <TableHeader class="sticky top-0 z-10 bg-muted/70 backdrop-blur">
        <TableRow class="hover:bg-transparent">
          <TableHead scope="col" class="min-w-[11rem] pl-4">
            {{ $t('history.logs.operatedAt') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[10rem]">
            {{ $t('history.logs.content') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[9rem]">
            {{ $t('history.logs.operator') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[12rem]">
            {{ $t('history.logs.nickname') }}
          </TableHead>
          <TableHead scope="col" class="min-w-[14rem] pr-4">
            {{ $t('history.logs.reason') }}
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow
          v-for="row in rows"
          :key="row.id"
          class="transition-colors duration-150 hover:bg-muted/40"
        >
          <TableCell class="whitespace-nowrap pl-4 tabular-nums text-foreground">
            {{ formatDateTime(row.created_at) }}
          </TableCell>

          <TableCell>
            <span
              class="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-medium"
              :class="
                row.action === 'add'
                  ? 'bg-destructive/10 text-destructive'
                  : 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
              "
            >
              <ShieldOff v-if="row.action === 'add'" class="h-3 w-3" />
              <ShieldCheck v-else class="h-3 w-3" />
              {{ actionLabel(row.action) }}
            </span>
          </TableCell>

          <TableCell>
            <span :class="row.actor_name ? 'text-foreground' : 'text-muted-foreground'">
              {{ row.actor_name || '-' }}
            </span>
          </TableCell>

          <TableCell class="text-foreground">
            <span class="truncate" :title="row.contact_name">{{ row.contact_name }}</span>
          </TableCell>

          <TableCell class="pr-4 text-foreground">
            <span v-if="row.reason" class="line-clamp-2" :title="row.reason">
              {{ row.reason }}
            </span>
            <span v-else class="text-muted-foreground">-</span>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { format, parseISO } from 'date-fns'
import { ShieldOff, ShieldCheck } from 'lucide-vue-next'
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
  }
})

const { t } = useI18n()

const actionLabel = (action) =>
  action === 'add' ? t('history.logs.action.add') : t('history.logs.action.remove')

const formatDateTime = (value) => {
  if (!value) return '-'
  try {
    return format(parseISO(value), 'yyyy-MM-dd HH:mm:ss')
  } catch (err) {
    return value
  }
}
</script>
