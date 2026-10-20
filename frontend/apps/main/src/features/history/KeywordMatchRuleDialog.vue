<template>
  <Dialog :open="open" @update:open="onOpenChange">
    <DialogContent class="max-h-[90vh] max-w-[560px] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>
          {{ isEdit ? $t('keywordMatch.editRule') : $t('keywordMatch.addRule') }}
        </DialogTitle>
        <DialogDescription class="sr-only">
          {{ $t('keywordMatch.subtitle') }}
        </DialogDescription>
      </DialogHeader>

      <form class="space-y-5" novalidate @submit.prevent="onSubmit">
        <!-- Rule name. -->
        <div class="space-y-2">
          <Label for="keyword-rule-name">{{ $t('keywordMatch.ruleName') }}</Label>
          <Input
            id="keyword-rule-name"
            v-model="form.name"
            type="text"
            :placeholder="$t('keywordMatch.ruleNamePlaceholder')"
            :aria-invalid="!!errors.name"
          />
          <p v-if="errors.name" class="text-xs text-destructive">{{ errors.name }}</p>
        </div>

        <!-- Source. -->
        <div class="space-y-2">
          <Label>{{ $t('keywordMatch.source') }}</Label>
          <Select v-model="form.source">
            <SelectTrigger aria-label="keywordMatch.source">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="option in KEYWORD_MATCH_SOURCES"
                :key="option.value"
                :value="option.value"
              >
                {{ $t(option.labelKey) }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Match type. -->
        <div class="space-y-2">
          <Label>{{ $t('keywordMatch.matchType') }}</Label>
          <Select v-model="form.match_type">
            <SelectTrigger aria-label="keywordMatch.matchType">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="option in KEYWORD_MATCH_TYPES"
                :key="option.value"
                :value="option.value"
              >
                {{ $t(option.labelKey) }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Pattern. -->
        <div class="space-y-2">
          <Label for="keyword-rule-pattern">{{ $t('keywordMatch.pattern') }}</Label>
          <Textarea
            id="keyword-rule-pattern"
            v-model="form.pattern"
            rows="3"
            :placeholder="
              form.match_type === 'regex'
                ? $t('keywordMatch.patternRegexPlaceholder')
                : $t('keywordMatch.patternPlaceholder')
            "
            :aria-invalid="!!errors.pattern"
          />
          <p class="text-xs text-muted-foreground">
            {{
              form.match_type === 'regex'
                ? $t('keywordMatch.pattern.regexDescription')
                : $t('keywordMatch.pattern.description')
            }}
          </p>
          <p v-if="errors.pattern" class="text-xs text-destructive">{{ errors.pattern }}</p>
        </div>

        <!-- Case sensitive + search first. -->
        <div class="space-y-3 rounded-lg border border-border bg-muted/30 p-3">
          <div class="flex items-start gap-3">
            <Checkbox
              id="keyword-rule-case-sensitive"
              :checked="form.case_sensitive"
              aria-label="keywordMatch.caseSensitive"
              @update:checked="(value) => (form.case_sensitive = value)"
            />
            <div class="space-y-1">
              <Label for="keyword-rule-case-sensitive" class="cursor-pointer">
                {{ $t('keywordMatch.caseSensitive') }}
              </Label>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Checkbox
              id="keyword-rule-search-first"
              :checked="form.search_first"
              aria-label="keywordMatch.searchFirst"
              @update:checked="(value) => (form.search_first = value)"
            />
            <div class="space-y-1">
              <Label for="keyword-rule-search-first" class="cursor-pointer">
                {{ $t('keywordMatch.searchFirst') }}
              </Label>
              <p class="text-xs text-muted-foreground">
                {{ $t('keywordMatch.searchFirst.description') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Action. -->
        <div class="space-y-2">
          <Label>{{ $t('keywordMatch.action') }}</Label>
          <Select v-model="form.action">
            <SelectTrigger aria-label="keywordMatch.action">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="option in KEYWORD_MATCH_ACTIONS"
                :key="option.value"
                :value="option.value"
              >
                {{ $t(option.labelKey) }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Reply content. -->
        <div class="space-y-2">
          <Label for="keyword-rule-reply">{{ $t('keywordMatch.replyContent') }}</Label>
          <Textarea id="keyword-rule-reply" v-model="form.reply_content" rows="4" />
          <p class="text-xs text-muted-foreground">
            {{ $t('keywordMatch.replyContent.description') }}
          </p>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" @click="onOpenChange(false)">
            {{ $t('keywordMatch.cancel') }}
          </Button>
          <Button type="submit" :is-loading="saving">
            {{ $t('keywordMatch.save') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Label } from '@shared-ui/components/ui/label'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import {
  KEYWORD_MATCH_ACTIONS,
  KEYWORD_MATCH_SOURCES,
  KEYWORD_MATCH_TYPES,
  isValidRegex,
  toForm
} from '@/features/history/keywordMatchOptions'

const props = defineProps({
  open: { type: Boolean, default: false },
  rule: { type: Object, default: null }
})
const emit = defineEmits(['update:open', 'saved'])

const { t } = useI18n()
const emitter = useEmitter()

const saving = ref(false)
const errors = reactive({ name: '', pattern: '' })
const form = ref(toForm(null))

const isEdit = computed(() => !!props.rule?.id)

// Reset the form whenever the dialog opens or the edited rule changes.
watch(
  () => [props.open, props.rule],
  () => {
    form.value = toForm(props.rule)
    errors.name = ''
    errors.pattern = ''
  },
  { immediate: true, deep: true }
)

function showError (message) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: message })
}

function validate () {
  errors.name = form.value.name.trim() ? '' : t('keywordMatch.nameRequired')
  errors.pattern = form.value.pattern.trim() ? '' : t('keywordMatch.patternRequired')
  if (!errors.pattern && form.value.match_type === 'regex' && !isValidRegex(form.value.pattern)) {
    errors.pattern = t('keywordMatch.regexInvalid')
  }
  return !errors.name && !errors.pattern
}

function onOpenChange (next) {
  emit('update:open', next)
}

async function onSubmit () {
  if (saving.value || !validate()) return
  saving.value = true
  const payload = {
    name: form.value.name.trim(),
    source: form.value.source,
    match_type: form.value.match_type,
    pattern: form.value.pattern.trim(),
    case_sensitive: !!form.value.case_sensitive,
    search_first: !!form.value.search_first,
    action: form.value.action,
    reply_content: form.value.reply_content,
    enabled: form.value.enabled !== false,
    sort_order: Number(form.value.sort_order) || 0
  }
  try {
    if (isEdit.value) {
      await api.updateKeywordMatchRule(props.rule.id, payload)
    } else {
      await api.createKeywordMatchRule(payload)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('keywordMatch.saveSuccess') })
    emit('saved')
    onOpenChange(false)
  } catch (err) {
    showError(handleHTTPError(err).message)
  } finally {
    saving.value = false
  }
}
</script>
