import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from '../utils/auth'

const HomePage = () => import('../pages/HomePage.vue')
const MainFlow = () => import('../pages/MainFlow.vue')
const CustomersPage = () => import('../pages/CustomersPage.vue')
const SettingsPage = () => import('../pages/SettingsPage.vue')
const LoginPage = () => import('../pages/LoginPage.vue')

const history = createWebHistory(import.meta.env.BASE_URL)

const router = createRouter({
  history,
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginPage,
      meta: { public: true },
    },
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
const PUBLIC_ROUTES = new Set(['login'])

router.beforeEach((to, from, next) => {
  const token = getToken()
  if (!token && !PUBLIC_ROUTES.has(to.name)) {
    const redirect = to.fullPath && to.fullPath !== '/' ? { name: 'login', query: { redirect: to.fullPath } } : { name: 'login' }
    next(redirect)
    return
  }

  if (token && to.name === 'login') {
    const fallback = typeof to.query.redirect === 'string' && to.query.redirect ? to.query.redirect : from.fullPath || '/'
    next(fallback)
    return
  }
  next()
})

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
    if (!PUBLIC_ROUTES.has(to.name)) {
      sessionStorage.setItem(LAST_ROUTE_KEY, to.fullPath || '/')
    }
  })
}

export default router
