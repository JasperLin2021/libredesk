<template>
  <div class="space-y-6">
    <div class="mb-5 flex items-center justify-between gap-4">
      <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>
    <KnowledgeManager />
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import KnowledgeManager from '@/features/admin/knowledge/KnowledgeManager.vue'

const { t } = useI18n()
const breadcrumbLinks = [{ path: '', label: t('admin.knowledgeBase.title') }]
</script>
