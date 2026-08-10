<script setup>
import { MessageCircle, Bot, BarChart3, Shield, Globe, Zap, Users, Clock } from 'lucide-vue-next'

const features = [
  {
    icon: MessageCircle,
    title: '实时在线聊天',
    desc: '与网站访客实时对话，毫秒级消息传递，支持富文本、附件和表情，让沟通更自然高效。',
    color: 'from-blue-500 to-cyan-500',
  },
  {
    icon: Bot,
    title: 'AI 智能助手',
    desc: '基于大语言模型的自动回复引擎，7x24 小时在线，准确理解用户意图并给出专业回答。',
    color: 'from-purple-500 to-pink-500',
  },
  {
    icon: BarChart3,
    title: '数据分析面板',
    desc: '全面的对话数据分析，包括满意度评分、响应时间趋势、团队绩效等关键业务指标。',
    color: 'from-emerald-500 to-teal-500',
  },
  {
    icon: Shield,
    title: '企业级安全',
    desc: '端到端加密传输，SOC 2 合规认证，灵活的权限管理体系，守护你的数据安全。',
    color: 'from-orange-500 to-red-500',
  },
  {
    icon: Globe,
    title: '多语言支持',
    desc: '内置 20+ 语言翻译引擎，自动检测访客语言偏好，打破跨境沟通障碍。',
    color: 'from-indigo-500 to-blue-500',
  },
  {
    icon: Zap,
    title: '自动化工作流',
    desc: '可视化的自动化规则编辑器，对话自动分配、标签分类、SLA 预警一键配置。',
    color: 'from-amber-500 to-orange-500',
  },
  {
    icon: Users,
    title: '团队协作',
    desc: '支持内部备注、对话转接、碰撞检测，让客服团队协作无缝高效。',
    color: 'from-rose-500 to-pink-500',
  },
  {
    icon: Clock,
    title: '多渠道接入',
    desc: '统一收件箱集成邮件、微信、Facebook Messenger 等渠道，一处管理所有客户对话。',
    color: 'from-sky-500 to-indigo-500',
  },
]
</script>

<template>
  <section id="features" class="py-24 px-4 sm:px-6 lg:px-8 bg-slate-50">
    <div class="max-w-7xl mx-auto">
      <!-- Section header -->
      <div class="text-center mb-16">
        <h2 class="text-3xl sm:text-4xl font-bold text-slate-900 tracking-tight">
          为现代客服团队打造的
          <span class="gradient-text">一站式平台</span>
        </h2>
        <p class="mt-4 text-lg text-slate-500 max-w-2xl mx-auto">
          从实时聊天到 AI 自动化，CloudNova 提供你需要的所有工具，让客户服务变得简单而高效。
        </p>
      </div>

      <!-- Feature Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div
          v-for="feature in features"
          :key="feature.title"
          class="card-hover group relative bg-white rounded-2xl p-6 border border-slate-100
                 hover:border-slate-200 cursor-default"
        >
          <!-- Icon -->
          <div
            class="w-12 h-12 rounded-xl bg-gradient-to-br mb-5 flex items-center justify-center shadow-sm"
            :class="feature.color"
          >
            <component :is="feature.icon" class="w-5 h-5 text-white" />
          </div>

          <!-- Title -->
          <h3 class="text-lg font-semibold text-slate-900 mb-2 group-hover:text-primary transition-colors duration-200">
            {{ feature.title }}
          </h3>

          <!-- Description -->
          <p class="text-sm text-slate-500 leading-relaxed">{{ feature.desc }}</p>
        </div>
      </div>
    </div>
  </section>
</template>
