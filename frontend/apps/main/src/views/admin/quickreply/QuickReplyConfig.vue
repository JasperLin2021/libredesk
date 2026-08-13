<template>
  <div class="space-y-6">
    <div class="mb-5 flex items-center justify-between gap-4">
      <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>

    <Spinner v-if="loading" />

    <div v-else class="space-y-6">
    <!-- Inbox selector + enable toggle -->
    <Card>
      <CardContent class="p-4 flex flex-wrap items-center gap-4">
        <div class="flex items-center gap-2 min-w-[260px] flex-1">
          <Label class="whitespace-nowrap">{{ $t('admin.quickReply.selectInbox') }}</Label>
          <Select :model-value="String(selectedInboxId)" @update:model-value="onSelectInbox">
            <SelectTrigger>
              <SelectValue :placeholder="$t('admin.quickReply.selectInbox')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="inbox in inboxes" :key="inbox.id" :value="String(inbox.id)">
                {{ inbox.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <SwitchField
          :checked="config.enabled"
          :title="$t('admin.quickReply.enabled')"
          :description="$t('admin.quickReply.enabled.description')"
          @update:checked="onToggleEnabled"
        />
      </CardContent>
    </Card>

    <template v-if="selectedInboxId">
      <!-- Automatic reply messages -->
      <Card>
        <CardHeader>
          <CardTitle>{{ $t('admin.quickReply.config') }}</CardTitle>
        </CardHeader>
        <CardContent>
          <form @submit="onSubmit" novalidate class="space-y-6">
            <FormField v-slot="{ componentField }" name="welcome_message">
              <FormItem>
                <FormLabel>{{ $t('admin.quickReply.welcomeMessage') }}</FormLabel>
                <FormControl>
                  <Textarea rows="5" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.quickReply.welcomeMessage.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="transfer_keyword">
              <FormItem>
                <FormLabel>{{ $t('admin.quickReply.transferKeyword') }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.quickReply.transferKeyword.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="queue_reply">
              <FormItem>
                <FormLabel>{{ $t('admin.quickReply.queueReply') }}</FormLabel>
                <FormControl>
                  <Textarea rows="3" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.quickReply.queueReply.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="assigned_reply">
              <FormItem>
                <FormLabel>{{ $t('admin.quickReply.assignedReply') }}</FormLabel>
                <FormControl>
                  <Textarea rows="2" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.quickReply.assignedReply.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="closed_reply">
              <FormItem>
                <FormLabel>{{ $t('admin.quickReply.closedReply') }}</FormLabel>
                <FormControl>
                  <Textarea rows="2" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.quickReply.closedReply.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <div class="flex items-center gap-3">
              <Button type="submit" :disabled="isSaving">
                {{ isSaving ? $t('globals.messages.saving') : $t('globals.terms.save') }}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <!-- Topics & questions -->
      <TopicQuestionManager :inbox-id="selectedInboxId" />
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useI18n } from 'vue-i18n'
import api from '../../../api'
import { Card, CardHeader, CardTitle, CardContent } from '@shared-ui/components/ui/card'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Label } from '@shared-ui/components/ui/label'
import {
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import TopicQuestionManager from '@/features/admin/quickreply/TopicQuestionManager.vue'

const props = defineProps({
  id: {
    type: String,
    default: ''
  }
})

const { t } = useI18n()
const emitter = useEmitter()

const loading = ref(true)
const isSaving = ref(false)
const inboxes = ref([])
const selectedInboxId = ref(null)
const config = ref({ enabled: false })

const breadcrumbLinks = [
  { path: '', label: t('admin.quickReply.title') }
]

const formSchema = z.object({
  welcome_message: z.string(),
  transfer_keyword: z.string().min(1, t('globals.messages.empty', { name: t('admin.quickReply.transferKeyword') })),
  queue_reply: z.string(),
  assigned_reply: z.string(),
  closed_reply: z.string()
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {}
})

const loadConfig = async (inboxId) => {
  const cfgResp = await api.getQuickReplyConfig(inboxId)
  config.value = cfgResp.data.data || { enabled: false }
  form.setValues({
    welcome_message: config.value.welcome_message || '',
    transfer_keyword: config.value.transfer_keyword || '我要转人工',
    queue_reply: config.value.queue_reply || '',
    assigned_reply: config.value.assigned_reply || '',
    closed_reply: config.value.closed_reply || ''
  })
}

const onSelectInbox = (value) => {
  selectedInboxId.value = Number(value)
  loadConfig(selectedInboxId.value).catch((error) => {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  })
}

const onToggleEnabled = async (value) => {
  const next = Boolean(value)
  config.value.enabled = next
  await saveConfig({ enabled: next })
}

const saveConfig = async (extra = {}) => {
  if (!selectedInboxId.value) return
  isSaving.value = true
  try {
    const payload = {
      welcome_message: form.values.welcome_message,
      transfer_keyword: form.values.transfer_keyword,
      queue_reply: form.values.queue_reply,
      assigned_reply: form.values.assigned_reply,
      closed_reply: form.values.closed_reply,
      enabled: config.value.enabled,
      ...extra
    }
    const resp = await api.updateQuickReplyConfig(selectedInboxId.value, payload)
    config.value = resp.data.data
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.savedSuccessfully') })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  } finally {
    isSaving.value = false
  }
}

const onSubmit = form.handleSubmit(() => saveConfig())

onMounted(async () => {
  try {
    loading.value = true
    const resp = await api.getInboxes()
    inboxes.value = resp.data.data
    // Default to the inbox passed in the route (if any) or the first one.
    const initial = props.id ? Number(props.id) : inboxes.value[0]?.id
    if (initial) {
      selectedInboxId.value = initial
      await loadConfig(initial)
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  } finally {
    loading.value = false
  }
})
</script>
