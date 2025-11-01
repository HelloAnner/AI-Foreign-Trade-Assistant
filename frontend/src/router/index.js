import { createRouter, createWebHistory } from 'vue-router'

const HomePage = () => import('../pages/HomePage.vue')
const MainFlow = () => import('../pages/MainFlow.vue')
const CustomersPage = () => import('../pages/CustomersPage.vue')
const SettingsPage = () => import('../pages/SettingsPage.vue')

const history = createWebHistory(import.meta.env.BASE_URL)

const router = createRouter({
  history,
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

const LAST_ROUTE_KEY = 'fta:last-route'

if (typeof window !== 'undefined') {
  const initialPath = window.location.pathname + window.location.search + window.location.hash
  const navEntries = performance.getEntriesByType?.('navigation') || []
  const isReload = navEntries.length ? navEntries[0].type === 'reload' : performance.navigation?.type === performance.navigation?.TYPE_RELOAD

  if (initialPath && initialPath !== '/' && router.currentRoute.value.fullPath === '/') {
    router.replace(initialPath).catch(() => {})
  } else if (isReload && initialPath === '/') {
    const saved = sessionStorage.getItem(LAST_ROUTE_KEY)
    if (saved && saved !== '/' && saved !== router.currentRoute.value.fullPath) {
      router.replace(saved).catch(() => {})
    }
  }

  router.afterEach((to) => {
    sessionStorage.setItem(LAST_ROUTE_KEY, to.fullPath || '/')
  })
}

export default router
