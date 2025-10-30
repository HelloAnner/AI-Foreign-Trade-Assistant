<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="avatar"></div>
        <div>
          <p class="brand-title">AI外贸助手</p>
          <p class="brand-sub">智能客户开发</p>
        </div>
      </div>
      <nav class="menu">
        <button class="menu-item menu-item--active">
          <span class="material">search</span>
          客户开发
        </button>
        <button class="menu-item">
          <span class="material">group</span>
          客户管理
        </button>
        <button class="menu-item">
          <span class="material">mail</span>
          邮件营销
        </button>
        <button class="menu-item">
          <span class="material">bar_chart</span>
          数据分析
        </button>
        <button class="menu-item">
          <span class="material">settings</span>
          全局配置
        </button>
      </nav>
    </aside>

    <main class="main">
      <div class="progress">
        <div class="progress-top">
          <p>当前进度：{{ step }}/{{ total }}</p>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${Math.min(100, Math.max(0, progress))}%` }"></div>
        </div>
      </div>

      <header class="headline">
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
      </header>

      <section class="content">
        <slot />
      </section>

      <footer class="footer">
        <slot name="footer" />
      </footer>
    </main>
  </div>
</template>

<script setup>
const props = defineProps({
  step: { type: Number, default: 1 },
  total: { type: Number, default: 5 },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  progress: { type: Number, default: 0 },
})
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined');

.shell {
  min-height: 100vh;
  display: flex;
  background: var(--surface-background);
  color: var(--text-primary);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.sidebar {
  width: 240px;
  background: #fff;
  border-right: 1px solid var(--border-subtle);
  padding: 32px 24px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.brand {
  display: flex;
  gap: 12px;
  align-items: center;
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
}

.brand-title {
  margin: 0;
  font-weight: 600;
}

.brand-sub {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.menu {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  border: none;
  border-radius: 12px;
  padding: 10px 14px;
  font-size: 14px;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
}

.menu-item--active {
  background: rgba(37, 99, 235, 0.12);
  color: var(--accent-color);
}

.material {
  font-family: 'Material Symbols Outlined';
  font-size: 20px;
  display: inline-flex;
}

.main {
  flex: 1;
  padding: 40px 64px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.progress {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.progress-top {
  display: flex;
  justify-content: space-between;
  color: var(--text-secondary);
  font-size: 14px;
}

.progress-bar {
  height: 8px;
  border-radius: 999px;
  background: var(--border-subtle);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
}

.headline h1 {
  margin: 0;
  font-size: 32px;
  font-weight: 800;
}

.headline p {
  margin: 8px 0 0;
  color: var(--text-secondary);
  font-size: 16px;
}

.content {
  flex: 1;
}

.footer {
  display: flex;
  justify-content: flex-end;
}
</style>
