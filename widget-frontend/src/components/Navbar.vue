<script setup>
import { ref } from 'vue'
import { Menu, X, MessageCircle } from 'lucide-vue-next'

const mobileMenuOpen = ref(false)

const navLinks = [
  { label: '首页', href: '#home' },
  { label: '产品', href: '#features' },
  { label: '定价', href: '#pricing' },
  { label: '关于', href: '#about' },
]
</script>

<template>
  <nav class="fixed top-0 left-0 right-0 z-40 bg-slate-900/80 backdrop-blur-xl border-b border-white/[0.06]">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-16">
        <!-- Logo -->
        <a href="#" class="flex items-center gap-2.5 group">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-accent flex items-center justify-center
                      group-hover:shadow-lg group-hover:shadow-primary/30 transition-shadow duration-300">
            <MessageCircle class="w-4 h-4 text-white" />
          </div>
          <span class="text-lg font-bold text-white tracking-tight">Cloud<span class="text-primary-light">Nova</span></span>
        </a>

        <!-- Desktop links -->
        <div class="hidden md:flex items-center gap-8">
          <a
            v-for="link in navLinks" :key="link.label"
            :href="link.href"
            class="text-sm font-medium text-slate-300 hover:text-white transition-colors duration-200 relative
                   after:absolute after:bottom-[-4px] after:left-0 after:h-[2px] after:w-0
                   after:bg-primary-light after:transition-all after:duration-300 hover:after:w-full"
          >
            {{ link.label }}
          </a>
          <button class="ml-2 px-5 py-2 bg-gradient-to-r from-primary to-accent text-white text-sm font-semibold
                         rounded-full hover:shadow-lg hover:shadow-primary/30 active:scale-95 transition-all duration-200">
            免费试用
          </button>
        </div>

        <!-- Mobile menu button -->
        <button
          @click="mobileMenuOpen = !mobileMenuOpen"
          class="md:hidden p-2 text-slate-300 hover:text-white transition-colors"
        >
          <Menu v-if="!mobileMenuOpen" class="w-6 h-6" />
          <X v-else class="w-6 h-6" />
        </button>
      </div>
    </div>

    <!-- Mobile menu -->
    <Transition
      enter-active-class="transition-all duration-300 ease-out"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition-all duration-200 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div v-if="mobileMenuOpen" class="md:hidden border-t border-white/[0.06] bg-slate-900/95 backdrop-blur-xl">
        <div class="px-4 py-4 space-y-3">
          <a
            v-for="link in navLinks" :key="link.label"
            :href="link.href"
            @click="mobileMenuOpen = false"
            class="block px-3 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-white/[0.06] rounded-lg transition-colors"
          >
            {{ link.label }}
          </a>
          <button class="w-full mt-2 px-5 py-2.5 bg-gradient-to-r from-primary to-accent text-white text-sm font-semibold
                         rounded-lg active:scale-95 transition-all duration-200">
            免费试用
          </button>
        </div>
      </div>
    </Transition>
  </nav>
</template>
