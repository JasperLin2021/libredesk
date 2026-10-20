<template>
  <div class="flex h-full w-full p-6">
    <div class="flex flex-1 flex-col gap-5 overflow-hidden">
      <header class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold text-foreground">
            {{ $t('keywordMatch.title') }}
          </h1>
          <p class="mt-1 text-sm text-muted-foreground">
            {{ $t('keywordMatch.subtitle') }}
          </p>
        </div>

        <Button v-if="canManage" class="shrink-0 gap-2" @click="openCreate">
          <Plus class="h-4 w-4" />
          {{ $t('keywordMatch.addRule') }}
        </Button>
      </header>

      <div class="min-h-0 flex-1 overflow-auto rounded-lg border border-border bg-card">
        <Spinner v-if="loading" class="my-16" />

        <div v-else-if="error" class="my-16 space-y-4 text-center">
          <p role="alert" class="text-lg text-destructive">{{ error }}</p>
          <Button variant="outline" @click="fetchRules">{{ $t('globals.terms.tryAgain') }}</Button>
        </div>

        <div v-else-if="!rules.length" class="my-16 space-y-2 text-center">
          <p class="text-base font-medium text-foreground">{{ $t('keywordMatch.empty') }}</p>
          <p class="text-sm text-muted-foreground">
            {{ $t('keywordMatch.emptyDescription') }}
          </p>
        </div>

        <Table v-else>
          <TableHeader>
            <TableRow class="bg-muted/40 hover:bg-muted/40">
              <TableHead class="w-[160px]">{{ $t('keywordMatch.ruleName') }}</TableHead>
              <TableHead class="w-[110px]">{{ $t('keywordMatch.source') }}</TableHead>
              <TableHead class="w-[100px]">{{ $t('keywordMatch.matchType') }}</TableHead>
              <TableHead class="min-w-[160px]">{{ $t('keywordMatch.pattern') }}</TableHead>
              <TableHead class="w-[110px] text-center">{{ $t('keywordMatch.caseSensitive') }}</TableHead>
              <TableHead class="w-[110px] text-center">{{ $t('keywordMatch.searchFirst') }}</TableHead>
              <TableHead class="w-[190px]">{{ $t('keywordMatch.action') }}</TableHead>
              <TableHead class="min-w-[200px]">{{ $t('keywordMatch.replyContent') }}</TableHead>
              <TableHead v-if="canManage" class="w-[80px] text-right">
                {{ $t('keywordMatch.actions') }}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow
              v-for="rule in rules"
              :key="rule.id"
              class="cursor-pointer transition-colors duration-150 hover:bg-accent/40"
              @click="openEdit(rule)"
            >
              <TableCell class="font-medium">{{ rule.name }}</TableCell>
              <TableCell>
                <Badge :variant="rule.source === 'agent' ? 'secondary' : 'default'">
                  {{ sourceLabel(rule.source, $t) }}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge variant="outline">{{ matchTypeLabel(rule.match_type, $t) }}</Badge>
              </TableCell>
              <TableCell class="max-w-[260px]">
                <span class="block truncate font-mono text-xs" :title="rule.pattern">
                  {{ rule.pattern }}
                </span>
              </TableCell>
              <TableCell class="text-center">
                <component :is="rule.case_sensitive ? Check : Minus" class="mx-auto h-4 w-4" :class="rule.case_sensitive ? 'text-primary' : 'text-muted-foreground'" />
                <span class="sr-only">{{ rule.case_sensitive ? $t('globals.messages.yes') : $t('globals.messages.no') }}</span>
              </TableCell>
              <TableCell class="text-center">
                <component :is="rule.search_first ? Check : Minus" class="mx-auto h-4 w-4" :class="rule.search_first ? 'text-primary' : 'text-muted-foreground'" />
                <span class="sr-only">{{ rule.search_first ? $t('globals.messages.yes') : $t('globals.messages.no') }}</span>
              </TableCell>
              <TableCell>{{ actionLabel(rule.action, $t) }}</TableCell>
              <TableCell class="max-w-[280px]">
                <span class="block truncate" :title="rule.reply_content">
                  {{ rule.reply_content }}
                </span>
              </TableCell>
              <TableCell v-if="canManage" class="text-right">
                <Button
                  variant="ghost"
                  size="icon"
                  class="text-muted-foreground transition-colors duration-150 hover:bg-destructive/10 hover:text-destructive"
                  :aria-label="$t('keywordMatch.delete')"
                  @click.stop="confirmDelete(rule)"
                >
                  <Trash2 class="h-4 w-4" />
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>

    <KeywordMatchRuleDialog
      :open="dialogOpen"
      :rule="editingRule"
      @update:open="dialogOpen = $event"
      @saved="fetchRules"
    />

    <AlertDialog :open="!!ruleToDelete" @update:open="ruleToDelete = null">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('keywordMatch.deleteTitle') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ $t('keywordMatch.deleteDescription', { name: ruleToDelete?.name || '' }) }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('keywordMatch.cancel') }}</AlertDialogCancel>
          <AlertDialogAction
            class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            :disabled="deleting"
            @click="deleteRule"
          >
            {{ $t('keywordMatch.delete') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Minus, Plus, Trash2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'
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
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useUserStore } from '@main/stores/user'
import { permissions } from '@main/constants/permissions'
import KeywordMatchRuleDialog from '@/features/history/KeywordMatchRuleDialog.vue'
import { actionLabel, matchTypeLabel, sourceLabel } from '@/features/history/keywordMatchOptions'

const emitter = useEmitter()
const { t } = useI18n()
const userStore = useUserStore()

const rules = ref([])
const loading = ref(false)
const error = ref(null)
const dialogOpen = ref(false)
const editingRule = ref(null)
const ruleToDelete = ref(null)
const deleting = ref(false)

const canManage = computed(() => userStore.can(permissions.KEYWORD_MATCH_MANAGE))

function showError (message) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: message })
}

async function fetchRules () {
  loading.value = true
  error.value = null
  try {
    const resp = await api.getKeywordMatchRules()
    rules.value = resp?.data?.data || []
  } catch (err) {
    const message = handleHTTPError(err).message || t('keywordMatch.loadError')
    error.value = message
    showError(message)
  } finally {
    loading.value = false
  }
}

function openCreate () {
  if (!canManage.value) {
    showError(t('keywordMatch.noPermission'))
    return
  }
  editingRule.value = null
  dialogOpen.value = true
}

function openEdit (rule) {
  if (!canManage.value) return
  editingRule.value = { ...rule }
  dialogOpen.value = true
}

function confirmDelete (rule) {
  ruleToDelete.value = rule
}

async function deleteRule () {
  if (!ruleToDelete.value || deleting.value) return
  deleting.value = true
  try {
    await api.deleteKeywordMatchRule(ruleToDelete.value.id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('keywordMatch.deleteSuccess') })
    ruleToDelete.value = null
    await fetchRules()
  } catch (err) {
    showError(handleHTTPError(err).message)
  } finally {
    deleting.value = false
  }
}

onMounted(fetchRules)
</script>
