<script setup>
import { ref } from 'vue'
import Navbar from './Navbar.vue'
import HeroSection from './HeroSection.vue'
import FeatureCards from './FeatureCards.vue'
import WidgetControls from './WidgetControls.vue'
import Footer from './Footer.vue'
import { Settings, X } from 'lucide-vue-next'

const props = defineProps({
  isReady: Boolean,
  isVisible: Boolean,
  unreadCount: Number,
  error: String,
  config: Object,
})

const emit = defineEmits(['show', 'hide', 'setUser', 'logout', 'destroy', 'reload', 'openConfig'])

const controlsOpen = ref(true)
</script>

<template>
  <div class="relative">
    <!-- Navbar -->
    <Navbar />

    <!-- Hero -->
    <HeroSection />

    <!-- Feature Cards -->
    <FeatureCards />

    <!-- Footer -->
    <Footer />

    <!-- Widget Controls Toggle -->
    <button
      @click="controlsOpen = !controlsOpen"
      class="fixed left-6 bottom-6 z-50 w-12 h-12 rounded-full bg-slate-900/90 backdrop-blur-md
             border border-slate-700/50 text-white flex items-center justify-center
             hover:bg-slate-800 transition-all duration-200 shadow-lg"
      :title="controlsOpen ? '关闭控制面板' : '打开控制面板'"
    >
      <Settings v-if="!controlsOpen" class="w-5 h-5" />
      <X v-else class="w-5 h-5" />
    </button>

    <!-- Widget Controls Panel -->
    <WidgetControls
      v-if="controlsOpen"
      :is-ready="isReady"
      :is-visible="isVisible"
      :unread-count="unreadCount"
      :error="error"
      :config="config"
      @show="emit('show')"
      @hide="emit('hide')"
      @set-user="(jwt) => emit('setUser', jwt)"
      @logout="emit('logout')"
      @destroy="emit('destroy')"
      @reload="emit('reload')"
      @open-config="emit('openConfig')"
    />
  </div>
</template>
