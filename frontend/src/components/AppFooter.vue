<template>
  <footer class="app-footer" aria-label="作者信息">
    <span class="app-footer__text">
      作者邮箱
      <a class="app-footer__link" href="mailto:helloanner@qq.com">helloanner@qq.com</a>
      · 有问题请联系作者
    </span>
  </footer>
</template>

<style scoped>
.app-footer {
  margin-top: auto;
  padding: 14px 20px;
  text-align: center;
  border-top: 1px solid var(--border-default);
  background: rgba(23, 27, 44, 0.02);
}

.app-footer__text {
  font-size: 13px;
  color: var(--text-tertiary);
  letter-spacing: 0.01em;
}

.app-footer__link {
  color: inherit;
  font-weight: 500;
}

.app-footer__link:hover {
  color: var(--primary-500);
}
</style>
