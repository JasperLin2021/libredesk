<script setup>
import { MessageCircle } from 'lucide-vue-next'

const footerLinks = {
  产品: ['实时聊天', 'AI 助手', '数据分析', '自动化', '多渠道'],
  公司: ['关于我们', '博客', '招聘', '新闻动态', '联系我们'],
  支持: ['帮助中心', 'API 文档', '系统状态', '安全说明', '社区论坛'],
  法律: ['隐私政策', '服务条款', 'Cookie 政策', 'GDPR', 'SLA 协议'],
}
</script>

<template>
  <footer class="bg-slate-900 pt-20 pb-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-7xl mx-auto">
      <div class="grid grid-cols-2 md:grid-cols-5 gap-8">
        <!-- Brand -->
        <div class="col-span-2 md:col-span-1">
          <a href="#" class="flex items-center gap-2.5 mb-4">
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-accent flex items-center justify-center">
              <MessageCircle class="w-4 h-4 text-white" />
            </div>
            <span class="text-lg font-bold text-white tracking-tight">Cloud<span class="text-primary-light">Nova</span></span>
          </a>
          <p class="text-sm text-slate-400 leading-relaxed">
            AI 驱动的智能客服平台，帮助全球超过 10,000 家企业提供卓越的客户服务体验。
          </p>
        </div>

        <!-- Links -->
        <div v-for="(links, category) in footerLinks" :key="category">
          <h4 class="text-sm font-semibold text-white mb-4">{{ category }}</h4>
          <ul class="space-y-3">
            <li v-for="link in links" :key="link">
              <a href="#" class="text-sm text-slate-400 hover:text-white transition-colors duration-200">
                {{ link }}
              </a>
            </li>
          </ul>
        </div>
      </div>

      <!-- Bottom -->
      <div class="mt-16 pt-8 border-t border-white/[0.06] flex flex-col sm:flex-row items-center justify-between gap-4">
        <p class="text-sm text-slate-500">
          &copy; 2026 CloudNova. All rights reserved.
        </p>
        <p class="text-xs text-slate-600">
          本页面为 LibreDesk Widget 嵌入演示
        </p>
      </div>
    </div>
  </footer>
</template>
