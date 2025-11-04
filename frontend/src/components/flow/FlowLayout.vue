<template>
  <div class="flow-shell">
    <aside class="flow-sidebar">
      <div class="flow-sidebar__brand">
        <h1>AI 外贸客户开发助手</h1>
      </div>

      <nav class="flow-sidebar__nav">
        <button
          v-for="item in navItems"
          :key="item.id"
          type="button"
          class="flow-sidebar__item"
          :class="{ 'flow-sidebar__item--active': item.route && isRouteActive([item.route]), 'flow-sidebar__item--disabled': item.disabled }"
          :disabled="item.disabled"
          @click="navigate(item.route)"
        >
          <span class="material">{{ item.icon }}</span>
          {{ item.label }}
        </button>
      </nav>


      <button
        type="button"
        class="flow-sidebar__footer"
        :class="{ 'flow-sidebar__footer--active': isRouteActive(['settings']) }"
        @click="navigate('settings')"
      >
        <span class="material">settings</span>
        全局配置
      </button>
    </aside>

    <main class="flow-main">
      <div v-if="total > 1" class="flow-steps">
        <button
          v-for="(label, index) in stepLabels"
          :key="label"
          type="button"
          class="flow-step"
          :class="stepClasses(index)"
          :disabled="index > unlockedIndex"
          @click="handleStepClick(index)"
        >
          {{ label }}
        </button>
      </div>

      <header class="flow-headline" v-if="title || subtitle">
        <h1>{{ title }}</h1>
        <p v-if="subtitle">{{ subtitle }}</p>
      </header>

      <section class="flow-content">
        <slot />
      </section>

      <footer v-if="$slots.footer" class="flow-footer">
        <slot name="footer" />
      </footer>
    </main>
  </div>
</template>

<script setup>
import { computed, inject } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const props = defineProps({
  step: { type: Number, default: 1 },
  total: { type: Number, default: 5 },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
})

const router = useRouter()
const route = useRoute()

const flowNav = inject('flowNav', null)

const navItems = computed(() => [
  { id: 'dashboard', route: null, icon: 'dashboard', label: '总览', disabled: true },
  { id: 'home', route: 'home', icon: 'add_circle', label: '新增客户', disabled: false },
  { id: 'customers', route: 'customers', icon: 'group', label: '客户管理', disabled: false },
  { id: 'email', route: null, icon: 'mark_email_unread', label: '邮件营销', disabled: true },
  { id: 'analytics', route: null, icon: 'stacked_bar_chart', label: '数据分析', disabled: true },
  { id: 'help', route: null, icon: 'help', label: '帮助中心', disabled: true },
])

const resolveIndex = (source, fallback) => {
  if (typeof source === 'number') return source
  if (source && typeof source.value === 'number') return source.value
  if (typeof source === 'function') {
    const result = source()
    return typeof result === 'number' ? result : fallback
  }
  return fallback
}

const activeIndex = computed(() => {
  const fallback = Math.max(0, props.step - 1)
  if (!flowNav) {
    return fallback
  }
  return resolveIndex(flowNav.activeIndex, fallback)
})

const unlockedIndex = computed(() => {
  const fallback = Math.max(0, props.step - 1)
  if (!flowNav) {
    return fallback
  }
  return resolveIndex(flowNav.unlockIndex, fallback)
})

const stepLabels = computed(() => {
  return Array.from({ length: props.total }, (_, index) => `Step ${index + 1}`)
})

const navigate = (name) => {
  if (!name || route.name === name) return
  router.push({ name })
}

const isRouteActive = (names) => names.includes(route.name)

const handleStepClick = (index) => {
  if (index > unlockedIndex.value) return
  flowNav?.goTo?.(index)
}

const stepClasses = (index) => {
  return {
    'flow-step--active': index === activeIndex.value,
    'flow-step--unlocked': index <= unlockedIndex.value,
  }
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined');

.flow-shell {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 220px 1fr;
  background: var(--app-background);
  color: var(--text-primary);
}

.flow-sidebar {
  background: var(--sidebar-background);
  border-right: 1px solid var(--border-default);
  padding: 32px 24px;
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.flow-sidebar__brand h1 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
}

.flow-sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.flow-sidebar__item,
.flow-sidebar__footer {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-radius: 14px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.flow-sidebar__item .material,
.flow-sidebar__footer .material {
  font-family: 'Material Symbols Outlined';
  font-size: 20px;
}

.flow-sidebar__item--active,
.flow-sidebar__footer--active {
  background: rgba(19, 73, 236, 0.12);
  color: var(--primary-500);
  font-weight: 600;
}

.flow-sidebar__item:hover,
.flow-sidebar__footer:hover {
  background: rgba(19, 73, 236, 0.08);
  color: var(--primary-500);
}

.flow-sidebar__footer {
  margin-top: auto;
  justify-content: flex-start;
}

.flow-sidebar__item--disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.flow-main {
  padding: 48px 72px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.flow-steps {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.flow-step {
  border-radius: var(--radius-full);
  padding: 10px 24px;
  background: #fff;
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.flow-step--unlocked {
  color: var(--primary-500);
}

.flow-step--active {
  background: var(--primary-500);
  border-color: transparent;
  color: #fff;
}

.flow-step:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.flow-headline h1 {
  margin: 0;
  font-size: 32px;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.flow-headline p {
  margin: 10px 0 0;
  font-size: 16px;
  color: var(--text-secondary);
}

.flow-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.flow-footer {
  margin-top: auto;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 1024px) {
  .flow-shell {
    grid-template-columns: 200px 1fr;
  }

  .flow-main {
    padding: 40px 32px;
  }
}

@media (max-width: 768px) {
  .flow-shell {
    grid-template-columns: 1fr;
  }

  .flow-sidebar {
    flex-direction: row;
    align-items: center;
    gap: 16px;
    padding: 16px 24px;
    border-right: none;
    border-bottom: 1px solid var(--border-default);
  }

  .flow-sidebar__nav {
    flex-direction: row;
    gap: 12px;
  }

  .flow-sidebar__footer {
    margin-top: 0;
  }

  .flow-main {
    padding: 32px 24px 48px;
  }

  .flow-steps {
    flex-wrap: wrap;
  }
}
</style>
