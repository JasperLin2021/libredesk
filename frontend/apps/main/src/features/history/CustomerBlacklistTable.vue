<template>
  <div
    class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border bg-background"
  >
    <Table class="min-h-0 flex-1">
      <TableHeader class="sticky top-0 z-10 bg-muted/70 backdrop-blur">
        <TableRow class="hover:bg-transparent">
          <TableHead scope="col" class="min-w-[12rem] pl-4">
            {{ $t('history.customer.nickname') }}
          </TableHead>

          <!-- "全部" tab: contact profile columns. -->
          <template v-if="columns === 'all'">
            <TableHead scope="col" class="min-w-[11rem]">
              {{ $t('history.customer.createdAt') }}
            </TableHead>
            <TableHead scope="col" class="min-w-[10rem]">
              {{ $t('history.customer.externalId') }}
            </TableHead>
            <TableHead scope="col" class="min-w-[10rem]">
              {{ $t('globals.terms.phoneNumber') }}
            </TableHead>
            <TableHead scope="col" class="min-w-[12rem]">
              {{ $t('globals.terms.email') }}
            </TableHead>
          </template>

          <!-- "在线黑名单" tab: who blocked them and why. -->
          <template v-else>
            <TableHead scope="col" class="min-w-[9rem]">
              {{ $t('history.customer.operator') }}
            </TableHead>
            <TableHead scope="col" class="min-w-[11rem]">
              {{ $t('history.customer.operatedAt') }}
            </TableHead>
            <TableHead scope="col" class="min-w-[14rem]">
              {{ $t('history.customer.reason') }}
            </TableHead>
          </template>

          <TableHead scope="col" class="w-[8rem] pr-4 text-right">
            {{ $t('history.customer.action') }}
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow
          v-for="row in rows"
          :key="row.contact_id"
          class="transition-colors duration-150 hover:bg-muted/40"
        >
          <TableCell class="pl-4">
            <div class="flex min-w-0 items-center gap-2.5">
              <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary"
              >
                {{ initials(row.contact_name) }}
              </span>
              <div class="min-w-0">
                <p class="truncate font-medium text-foreground" :title="row.contact_name">
                  {{ row.contact_name }}
                </p>
                <p class="truncate text-xs text-muted-foreground">
                  {{ contactTypeLabel(row.contact_type) }}
                </p>
              </div>
            </div>
          </TableCell>

          <template v-if="columns === 'all'">
            <TableCell class="whitespace-nowrap tabular-nums text-foreground">
              {{ formatDateTime(row.contact_created_at) }}
            </TableCell>
            <TableCell class="tabular-nums text-foreground">
              <span :class="row.external_user_id ? '' : 'text-muted-foreground'">
                {{ row.external_user_id || '-' }}
              </span>
            </TableCell>
            <TableCell class="whitespace-nowrap text-foreground">
              <span :class="row.phone_number ? '' : 'text-muted-foreground'">
                {{ phoneNumber(row) || '-' }}
              </span>
            </TableCell>
            <TableCell class="text-foreground">
              <span :class="row.contact_email ? '' : 'text-muted-foreground'">
                {{ row.contact_email || '-' }}
              </span>
            </TableCell>
          </template>

          <template v-else>
            <TableCell>
              <span :class="row.actor_name ? 'text-foreground' : 'text-muted-foreground'">
                {{ row.actor_name || '-' }}
              </span>
            </TableCell>
            <TableCell class="whitespace-nowrap tabular-nums text-foreground">
              {{ formatDateTime(row.blacklisted_at) }}
            </TableCell>
            <TableCell class="text-foreground">
              <span v-if="row.reason" class="line-clamp-2" :title="row.reason">
                {{ row.reason }}
              </span>
              <span v-else class="text-muted-foreground">-</span>
            </TableCell>
          </template>

          <TableCell class="pr-4 text-right">
            <!-- "全部" tab: proactively open a conversation with the customer. -->
            <button
              v-if="columns === 'all'"
              type="button"
              class="inline-flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-xs text-primary transition-colors duration-150 hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              :disabled="callbackContactId === row.contact_id"
              :aria-label="$t('history.customer.callback')"
              :title="$t('history.customer.callback')"
              @click="$emit('callback', row)"
            >
              <Loader2 v-if="callbackContactId === row.contact_id" class="h-3.5 w-3.5 animate-spin" />
              <PhoneOutgoing v-else class="h-3.5 w-3.5" />
              <span>{{ $t('history.customer.callback') }}</span>
            </button>

            <!-- "在线黑名单" tab: remove the customer from the blacklist. -->
            <BlacklistAction
              v-else
              :contact-id="row.contact_id"
              :initial-blocked="row.blocked"
              compact
              @updated="$emit('updated', $event)"
            />
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { format, parseISO } from 'date-fns'
import { Loader2, PhoneOutgoing } from 'lucide-vue-next'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'
import BlacklistAction from '@/features/blacklist/BlacklistAction.vue'

defineProps({
  rows: {
    type: Array,
    default: () => []
  },
  // Column set to render: 'all' (contact profile + callback) or 'blocked'.
  columns: {
    type: String,
    default: 'all'
  },
  // Contact whose callback request is in flight (spinner state).
  callbackContactId: {
    type: [Number, String],
    default: 0
  }
})

defineEmits(['updated', 'callback'])

const { t } = useI18n()

const initials = (name) => {
  const value = (name || '').trim()
  if (!value) return '?'
  return value.substring(0, 2).toUpperCase()
}

const contactTypeLabel = (type) => {
  if (type === 'visitor') return t('contact.type.visitor')
  return t('contact.type.contact')
}

const phoneNumber = (row) => {
  if (!row?.phone_number) return ''
  const code = row.phone_number_country_code ? `+${row.phone_number_country_code} ` : ''
  return `${code}${row.phone_number}`
}

const formatDateTime = (value) => {
  if (!value) return '-'
  try {
    return format(parseISO(value), 'yyyy-MM-dd HH:mm:ss')
  } catch (err) {
    return value
  }
}
</script>
