import { createRouter, createWebHistory } from 'vue-router'

const MainFlow = () => import('../pages/MainFlow.vue')
const SettingsPage = () => import('../pages/SettingsPage.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: MainFlow,
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsPage,
    },
  ],
})

export default router
