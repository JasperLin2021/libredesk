<template>
  <div class="blacklist-action">
    <!-- Full width button: used below the customer info card in the sidebar. -->
    <Button
      v-if="!compact"
      type="button"
      variant="outline"
      size="sm"
      class="w-full justify-start gap-2 border-destructive/40 text-destructive transition-colors duration-150 hover:border-destructive hover:bg-destructive/10 hover:text-destructive focus-visible:ring-destructive/40"
      :disabled="!contactId || loading || submitting"
      :aria-busy="loading || submitting"
      @click="openDialog"
    >
      <Loader2 v-if="loading || submitting" class="h-4 w-4 shrink-0 animate-spin" />
      <ShieldOff v-else-if="!blocked" class="h-4 w-4 shrink-0" />
      <ShieldCheck v-else class="h-4 w-4 shrink-0" />
      <span class="truncate">{{ blocked ? $t('blacklist.remove') : $t('blacklist.add') }}</span>
    </Button>

    <!-- Compact icon button: used as the row action of the customer table. -->
    <button
      v-else-if="blocked"
      type="button"
      class="inline-flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-xs text-muted-foreground transition-colors duration-150 hover:bg-destructive/10 hover:text-destructive focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      :disabled="!contactId || loading || submitting"
      :aria-label="$t('blacklist.remove')"
      :title="$t('blacklist.remove')"
      @click="openDialog"
    >
      <Loader2 v-if="loading || submitting" class="h-3.5 w-3.5 animate-spin" />
      <ShieldOff v-else class="h-3.5 w-3.5" />
      <span>{{ $t('blacklist.remove') }}</span>
    </button>
    <span v-else class="text-xs text-muted-foreground">-</span>

    <!-- Add to blacklist: requires a reason. -->
    <Dialog :open="addDialog" @update:open="addDialog = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader class="gap-y-3">
          <DialogTitle>{{ $t('blacklist.addTitle') }}</DialogTitle>
          <DialogDescription>{{ $t('blacklist.addConfirm') }}</DialogDescription>
        </DialogHeader>

        <div class="space-y-2 py-2">
          <Label for="blacklist-reason" class="text-sm font-medium">
            {{ $t('blacklist.reason') }}
          </Label>
          <Textarea
            id="blacklist-reason"
            v-model="reasonInput"
            rows="4"
            maxlength="500"
            :placeholder="$t('blacklist.reasonPlaceholder')"
            class="resize-none"
          />
          <p v-if="reasonTouched && !reasonInput.trim()" class="text-xs text-destructive">
            {{ $t('blacklist.reasonRequired') }}
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="addDialog = false">
            {{ $t('globals.messages.cancel') }}
          </Button>
          <Button variant="destructive" :disabled="!reasonInput.trim() || submitting" @click="confirmAdd">
            <Loader2 v-if="submitting" class="mr-1 h-4 w-4 animate-spin" />
            {{ $t('blacklist.confirm') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Remove from blacklist: plain confirmation, no reason needed. -->
    <Dialog :open="removeDialog" @update:open="removeDialog = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader class="gap-y-3">
          <DialogTitle>{{ $t('blacklist.removeTitle') }}</DialogTitle>
          <DialogDescription>{{ $t('blacklist.removeConfirm') }}</DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button variant="outline" @click="removeDialog = false">
            {{ $t('globals.messages.cancel') }}
          </Button>
          <Button variant="destructive" :disabled="submitting" @click="confirmRemove">
            <Loader2 v-if="submitting" class="mr-1 h-4 w-4 animate-spin" />
            {{ $t('blacklist.remove') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { Loader2, ShieldOff, ShieldCheck } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Label } from '@shared-ui/components/ui/label'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { useBlacklist } from './useBlacklist'

const props = defineProps({
  contactId: {
    type: [Number, String],
    default: 0
  },
  // Known blacklist state. Pass it when the caller (e.g. the customer table)
  // already has it so the component skips the status request.
  initialBlocked: {
    type: Boolean,
    default: null
  },
  // Icon-only variant for table rows.
  compact: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['updated'])

const { blocked, reason, loading, submitting, fetchStatus, add, remove } = useBlacklist()

const addDialog = ref(false)
const removeDialog = ref(false)
const reasonInput = ref('')
const reasonTouched = ref(false)

function syncInitialState () {
  if (props.initialBlocked === null) return
  blocked.value = props.initialBlocked
}

onMounted(() => {
  syncInitialState()
  if (props.initialBlocked === null && props.contactId) {
    fetchStatus(props.contactId)
  }
})

watch(
  () => props.contactId,
  (id) => {
    syncInitialState()
    reasonInput.value = ''
    reasonTouched.value = false
    if (props.initialBlocked === null && id) fetchStatus(id)
  }
)

watch(
  () => props.initialBlocked,
  () => syncInitialState()
)

function openDialog () {
  if (!props.contactId) return
  if (blocked.value) {
    removeDialog.value = true
    return
  }
  reasonInput.value = reason.value || ''
  reasonTouched.value = false
  addDialog.value = true
}

async function confirmAdd () {
  reasonTouched.value = true
  if (!reasonInput.value.trim()) return
  const ok = await add(props.contactId, reasonInput.value.trim())
  if (!ok) return
  addDialog.value = false
  emit('updated', { contactId: props.contactId, blocked: true, reason: reason.value })
}

async function confirmRemove () {
  const ok = await remove(props.contactId)
  if (!ok) return
  removeDialog.value = false
  emit('updated', { contactId: props.contactId, blocked: false, reason: '' })
}
</script>
