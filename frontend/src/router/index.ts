// Composables
import { createRouter, createWebHistory } from 'vue-router'
import Login from '@/views/Login.vue'
import Data from '@/store/modules/data'

const routes = [
  {
    path: '/login',
    name: 'pages.login',
    component: Login,
  },
  {
    path: '/',
    component: () => import('@/layouts/default/Default.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '/',
        name: 'pages.home',
        component: () => import('@/views/Home.vue'),
      },
      {
        path: '/inbounds',
        name: 'pages.inbounds',
        component: () => import('@/views/Inbounds.vue'),
      },
      {
        path: '/outbounds',
        name: 'pages.outbounds',
        component: () => import('@/views/Outbounds.vue'),
      },
      {
        path: '/endpoints',
        name: 'pages.endpoints',
        component: () => import('@/views/Endpoints.vue'),
      },
      {
        path: '/rules',
        name: 'pages.rules',
        component: () => import('@/views/Rules.vue'),
      },
      {
        path: '/block-rules',
        name: 'pages.blockRules',
        component: () => import('@/views/BlockRules.vue'),
      },
      {
        path: '/tls',
        name: 'pages.tls',
        component: () => import('@/views/Tls.vue'),
      },
      {
        path: '/dns',
        name: 'pages.dns',
        component: () => import('@/views/Dns.vue'),
      },
      {
        path: '/api',
        name: 'pages.api',
        component: () => import('@/views/Api.vue'),
      },
      {
        path: '/settings',
        name: 'pages.settings',
        component: () => import('@/views/Settings.vue'),
      },
      // 老路径兼容:/clients 已合并进 /inbounds(每个入站点「客户端」按钮),
      // /basics 已合并进 /settings 的「内核」tab,/admins 改密合并进「账号」tab。
      // 这些 redirect 让旧书签 / 浏览器历史不至于报 No match 警告。
      { path: '/clients', redirect: '/inbounds' },
      { path: '/basics', redirect: '/settings' },
      { path: '/admins', redirect: '/settings' },
      { path: '/services', redirect: '/inbounds' },
      // catch-all:其它未知路径回首页
      { path: '/:pathMatch(.*)*', redirect: '/' },
    ],
  },
]

const router = createRouter({
  history: createWebHistory((window as any).BASE_URL),
  routes,
})

const DEFAULT_TITLE = 'S-UI'
let intervalId:any

// Navigation guard to check authentication state
router.beforeEach((to) => {
  // Check the session cookie
  const sessionCookie = document.cookie.split(';').find(cookie => cookie.trim().startsWith('nexcore-s-ui='))
  const isAuthenticated = !!sessionCookie

  // If the route requires authentication and the user is not authenticated, redirect to /login
  if (to.meta.requiresAuth && !isAuthenticated) {
    return '/login'
  }
  if (to.path === '/login' && isAuthenticated) {
    // If already authenticated and visiting /login, redirect to '/'
    return '/'
  }

  // Load default data
  if (to.path !== '/login') {
    loadDataInterval()
  } else {
    if (intervalId) {
      clearInterval(intervalId)
      intervalId = undefined
    }
  }
})

const loadDataInterval = () => {
  if (intervalId) return
  Data().loadData()
  intervalId = setInterval(() => {
    Data().loadData()
  }, 10000)
}

export default router
