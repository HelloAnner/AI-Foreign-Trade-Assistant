<template>
  <aside class="app-sidebar">
    <div class="app-sidebar__brand">
      <div class="app-sidebar__emblem">
        <span class="material">auto_awesome</span>
      </div>
      <div class="app-sidebar__brand-text">
        <p class="app-sidebar__brand-title">AI外贸助手</p>
        <p class="app-sidebar__brand-sub">智能客户开发</p>
      </div>
    </div>

    <nav class="app-sidebar__menu">
      <button
        v-for="item in mainItems"
        :key="item.id"
        type="button"
        class="app-sidebar__menu-item"
        :class="{
          'app-sidebar__menu-item--active': isActive(item),
          'app-sidebar__menu-item--disabled': item.disabled,
        }"
        :disabled="item.disabled"
        @click="handleClick(item)"
      >
        <span class="app-sidebar__icon material">{{ item.icon }}</span>
        <span class="app-sidebar__label">{{ item.label }}</span>
      </button>
    </nav>

    <button
      type="button"
      class="app-sidebar__footer"
      :class="{ 'app-sidebar__footer--active': isActive(settingsItem) }"
      @click="handleClick(settingsItem)"
    >
      <span class="app-sidebar__icon material">{{ settingsItem.icon }}</span>
      <span class="app-sidebar__label">{{ settingsItem.label }}</span>
    </button>
  </aside>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  activeRoute: { type: String, default: '' },
})

const emit = defineEmits(['navigate'])

const mainItems = computed(() => [
  { id: 'home', route: 'home', icon: 'travel_explore', label: '客户开发', disabled: false },
  { id: 'clients', route: null, icon: 'folder_shared', label: '客户管理', disabled: true },
  { id: 'email', route: null, icon: 'mark_email_unread', label: '邮件营销', disabled: true },
  { id: 'analytics', route: null, icon: 'stacked_bar_chart', label: '数据分析', disabled: true },
])

const settingsItem = { id: 'settings', route: 'settings', icon: 'settings', label: '全局配置', disabled: false }

const isActive = (item) => item.route && props.activeRoute === item.route

const handleClick = (item) => {
  if (item.disabled || !item.route || isActive(item)) return
  emit('navigate', item.route)
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined');

.app-sidebar {
  width: 260px;
  background: #f5f7ff;
  border-right: 1px solid rgba(148, 163, 184, 0.12);
  padding: 32px 24px;
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.app-sidebar__brand {
  display: flex;
  align-items: center;
  gap: 14px;
}

.app-sidebar__emblem {
  width: 52px;
  height: 52px;
  border-radius: 18px;
  background: linear-gradient(145deg, #7c3aed, #4338ca);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 18px 34px rgba(76, 29, 149, 0.28);
}

.app-sidebar__brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.app-sidebar__brand-title {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
}

.app-sidebar__brand-sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.app-sidebar__menu {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.app-sidebar__menu-item,
.app-sidebar__footer {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border-radius: 20px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 15px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.app-sidebar__menu-item--active {
  background: linear-gradient(135deg, rgba(124, 92, 255, 0.16), rgba(79, 70, 229, 0.08));
  color: #4f46e5;
  font-weight: 600;
  box-shadow: 0 14px 28px rgba(79, 70, 229, 0.12), inset 0 0 0 1px rgba(79, 70, 229, 0.25);
}

.app-sidebar__menu-item--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.app-sidebar__menu-item:not(.app-sidebar__menu-item--disabled):hover,
.app-sidebar__footer:hover {
  background: rgba(79, 70, 229, 0.1);
  color: #4f46e5;
}

.app-sidebar__footer {
  margin-top: auto;
  justify-content: flex-start;
  font-weight: 500;
}

.app-sidebar__footer--active {
  color: #4f46e5;
  background: linear-gradient(135deg, rgba(124, 92, 255, 0.16), rgba(79, 70, 229, 0.08));
  box-shadow: 0 12px 24px rgba(79, 70, 229, 0.12), inset 0 0 0 1px rgba(79, 70, 229, 0.25);
}

.material {
  font-family: 'Material Symbols Outlined';
  font-size: 22px;
  line-height: 1;
}

.app-sidebar__icon {
  font-size: 21px;
}

.app-sidebar__label {
  letter-spacing: 0.01em;
}
</style>
