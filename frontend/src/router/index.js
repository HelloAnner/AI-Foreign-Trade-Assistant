import { createRouter, createWebHistory } from 'vue-router'

const HomePage = () => import('../pages/HomePage.vue')
const MainFlow = () => import('../pages/MainFlow.vue')
const CustomersPage = () => import('../pages/CustomersPage.vue')
const SettingsPage = () => import('../pages/SettingsPage.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomePage,
    },
    {
      path: '/customers',
      name: 'customers',
      component: CustomersPage,
    },
    {
      path: '/flow',
      name: 'flow',
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
