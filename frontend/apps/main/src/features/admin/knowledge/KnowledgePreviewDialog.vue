<template>
  <Dialog :open="open" @update:open="(v) => !v && close()">
    <DialogContent class="max-w-3xl max-h-[80vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ title }}</DialogTitle>
      </DialogHeader>
      <div v-if="meta" class="text-xs text-muted-foreground space-x-2">
        <span v-if="meta.author">{{ meta.author }}</span>
        <span v-if="meta.category">· {{ meta.category }}</span>
      </div>
      <div class="prose prose-sm max-w-none native-html" v-html="content"></div>
      <DialogFooter>
        <Button variant="outline" @click="close">{{ $t('admin.knowledgeBase.form.cancel') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'

defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '' },
  content: { type: String, default: '' },
  meta: { type: Object, default: null }
})
const emit = defineEmits(['update:open'])
const close = () => emit('update:open', false)
</script>
