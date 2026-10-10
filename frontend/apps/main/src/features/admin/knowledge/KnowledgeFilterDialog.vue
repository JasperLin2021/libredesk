<template>
  <Dialog :open="open" @update:open="(v) => !v && close()">
    <DialogContent class="max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ $t('admin.knowledgeBase.filter.title') }}</DialogTitle>
      </DialogHeader>
      <div class="space-y-5 py-2">
        <div>
          <Label class="mb-2 block text-sm font-medium">{{ $t('admin.knowledgeBase.filter.timeOrder') }}</Label>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              class="rounded-md border px-3 py-2 text-sm transition-colors"
              :class="form.sort === 'desc' ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-muted'"
              @click="form.sort = 'desc'"
            >
              {{ $t('admin.knowledgeBase.filter.timeDesc') }}
            </button>
            <button
              type="button"
              class="rounded-md border px-3 py-2 text-sm transition-colors"
              :class="form.sort === 'asc' ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-muted'"
              @click="form.sort = 'asc'"
            >
              {{ $t('admin.knowledgeBase.filter.timeAsc') }}
            </button>
          </div>
        </div>

        <div>
          <Label class="mb-2 block text-sm font-medium">{{ $t('admin.knowledgeBase.filter.timeRange') }}</Label>
          <div class="space-y-2">
            <Select v-model="form.dateField">
              <SelectTrigger>
                <SelectValue :placeholder="$t('admin.knowledgeBase.filter.timeRangeCreatedAt')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="created_at">{{ $t('admin.knowledgeBase.filter.timeRangeCreatedAt') }}</SelectItem>
                <SelectItem value="updated_at">{{ $t('admin.knowledgeBase.filter.timeRangeUpdatedAt') }}</SelectItem>
              </SelectContent>
            </Select>
            <Select v-model="form.dateRange">
              <SelectTrigger>
                <SelectValue :placeholder="$t('admin.knowledgeBase.filter.timeRangeNoLimit')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{{ $t('admin.knowledgeBase.filter.timeRangeNoLimit') }}</SelectItem>
                <SelectItem value="today">{{ $t('admin.knowledgeBase.filter.timeRangeToday') }}</SelectItem>
                <SelectItem value="yesterday">{{ $t('admin.knowledgeBase.filter.timeRangeYesterday') }}</SelectItem>
                <SelectItem value="this_week">{{ $t('admin.knowledgeBase.filter.timeRangeThisWeek') }}</SelectItem>
                <SelectItem value="this_month">{{ $t('admin.knowledgeBase.filter.timeRangeThisMonth') }}</SelectItem>
                <SelectItem value="last_month">{{ $t('admin.knowledgeBase.filter.timeRangeLastMonth') }}</SelectItem>
                <SelectItem value="custom">{{ $t('admin.knowledgeBase.filter.timeRangeCustom') }}</SelectItem>
              </SelectContent>
            </Select>
            <div v-if="form.dateRange === 'custom'" class="grid grid-cols-2 gap-2">
              <Input v-model="form.dateFrom" type="date" />
              <Input v-model="form.dateTo" type="date" />
            </div>
          </div>
        </div>

        <div>
          <Label class="mb-2 block text-sm font-medium">{{ $t('admin.knowledgeBase.filter.keyword') }}</Label>
          <div class="flex gap-2">
            <Select v-model="form.keywordField" class="shrink-0">
              <SelectTrigger class="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{{ $t('admin.knowledgeBase.filter.keywordAll') }}</SelectItem>
                <SelectItem value="title">{{ $t('admin.knowledgeBase.filter.keywordTitle') }}</SelectItem>
                <SelectItem value="content">{{ $t('admin.knowledgeBase.filter.keywordContent') }}</SelectItem>
              </SelectContent>
            </Select>
            <Input
              v-model="form.keywords"
              class="min-w-0 flex-1"
              :placeholder="$t('admin.knowledgeBase.filter.keywordPlaceholder')"
            />
          </div>
        </div>
      </div>
      <DialogFooter class="flex items-center justify-between gap-2 sm:justify-between">
        <Button variant="ghost" @click="clear">{{ $t('admin.knowledgeBase.filter.clear') }}</Button>
        <Button @click="apply">{{ $t('admin.knowledgeBase.filter.apply') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch, reactive } from 'vue'
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
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem
} from '@shared-ui/components/ui/select'

const props = defineProps({
  open: { type: Boolean, default: false },
  initial: {
    type: Object,
    default: () => ({})
  }
})
const emit = defineEmits(['update:open', 'apply'])

const defaultForm = () => ({
  sort: 'desc',
  dateField: 'created_at',
  dateRange: 'all',
  dateFrom: '',
  dateTo: '',
  keywordField: 'all',
  keywords: ''
})

const form = reactive(defaultForm())

watch(
  () => props.open,
  (val) => {
    if (val) {
      Object.assign(form, defaultForm(), props.initial || {})
    }
  }
)

const close = () => emit('update:open', false)

const clear = () => {
  Object.assign(form, defaultForm())
  emit('apply', { ...form })
  close()
}

const apply = () => {
  emit('apply', { ...form })
  close()
}
</script>
