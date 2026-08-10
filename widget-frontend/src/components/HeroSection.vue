<script setup>
import { ArrowRight, Play, Sparkles } from 'lucide-vue-next'
</script>

<template>
  <section id="home" class="hero-gradient pt-32 pb-28 px-4 sm:px-6 lg:px-8 overflow-hidden relative">
    <!-- Background decoration -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-96 h-96 bg-primary/20 rounded-full blur-3xl opacity-50"></div>
      <div class="absolute -bottom-20 -left-20 w-72 h-72 bg-accent/20 rounded-full blur-3xl opacity-40"></div>
      <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px]
                   bg-gradient-to-br from-primary/5 to-accent/5 rounded-full blur-3xl"></div>
    </div>

    <div class="max-w-4xl mx-auto text-center relative z-10">
      <!-- Badge -->
      <div class="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-white/[0.06] border border-white/[0.08]
                  text-sm text-slate-300 mb-8 backdrop-blur-sm">
        <Sparkles class="w-4 h-4 text-primary-light" />
        智能在线客服解决方案
      </div>

      <!-- Heading -->
      <h1 class="text-4xl sm:text-5xl lg:text-6xl font-extrabold text-white leading-tight tracking-tight">
        用 AI 驱动的
        <span class="gradient-text">智能客服</span>
        <br />
        提升每一次对话
      </h1>

      <!-- Subtitle -->
      <p class="mt-6 text-lg sm:text-xl text-slate-400 max-w-2xl mx-auto leading-relaxed">
        CloudNova 将实时聊天、AI 助手与自动化工作流融为一体，
        帮助你的团队更快响应客户，提供令人难忘的服务体验。
      </p>

      <!-- CTA Buttons -->
      <div class="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
        <button class="group flex items-center gap-2 px-8 py-3.5 bg-gradient-to-r from-primary to-accent
                       text-white font-semibold rounded-full hover:shadow-xl hover:shadow-primary/30
                       active:scale-[0.97] transition-all duration-200 text-base">
          开始免费使用
          <ArrowRight class="w-4 h-4 group-hover:translate-x-0.5 transition-transform" />
        </button>
        <button class="flex items-center gap-2 px-8 py-3.5 border border-white/[0.12] text-slate-300 font-medium
                       rounded-full hover:bg-white/[0.06] hover:text-white active:scale-[0.97]
                       transition-all duration-200 text-base backdrop-blur-sm">
          <Play class="w-4 h-4" />
          观看演示
        </button>
      </div>

      <!-- Stats -->
      <div class="mt-16 grid grid-cols-3 gap-8 max-w-lg mx-auto">
        <div v-for="stat in [
          { value: '99.9%', label: '服务可用性' },
          { value: '&lt; 5s', label: '平均响应时间' },
          { value: '10K+', label: '活跃客户' },
        ]" :key="stat.label" class="text-center">
          <div class="text-2xl sm:text-3xl font-bold text-white">{{ stat.value }}</div>
          <div class="text-xs sm:text-sm text-slate-500 mt-1">{{ stat.label }}</div>
        </div>
      </div>
    </div>
  </section>
</template>
