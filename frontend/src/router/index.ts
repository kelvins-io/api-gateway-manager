import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  // BASE_URL 来自 Vite base（VITE_BASE_PATH）。fullPath 不含该前缀，登录回跳仍是 /groups/1/apis 这类应用内路径。
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      redirect: '/spaces',
      children: [
        {
          path: 'spaces',
          name: 'spaces',
          component: () => import('@/views/SpacesView.vue'),
        },
        {
          path: 'spaces/:id/members',
          name: 'space-members',
          component: () => import('@/views/MembersView.vue'),
        },
        {
          path: 'gateways',
          name: 'gateways',
          component: () => import('@/views/GatewaysView.vue'),
          meta: { systemAdmin: true },
        },
        {
          path: 'market',
          name: 'market',
          component: () => import('@/views/MarketView.vue'),
        },
        {
          path: 'groups',
          name: 'groups',
          component: () => import('@/views/GroupsView.vue'),
        },
        {
          path: 'groups/:gid/apis',
          name: 'apis',
          component: () => import('@/views/ApisView.vue'),
        },
        {
          path: 'upstreams',
          name: 'upstreams',
          component: () => import('@/views/UpstreamsView.vue'),
        },
        {
          path: 'consumers',
          name: 'consumers',
          component: () => import('@/views/ConsumersView.vue'),
        },
        {
          path: 'plugins',
          name: 'plugins',
          component: () => import('@/views/PluginsView.vue'),
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const store = useUserStore()
  if (to.meta.public) {
    if (store.isLoggedIn && (to.name === 'login' || to.name === 'register')) {
      return { path: '/' }
    }
    return true
  }
  if (!store.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  // always refresh profile for role-sensitive pages
  if (store.token && (!store.user || to.meta.systemAdmin)) {
    try {
      await store.fetchMe()
    } catch {
      store.logout()
      return { path: '/login' }
    }
  }
  if (to.meta.systemAdmin && store.user?.role !== 'system_admin') {
    return { path: '/spaces' }
  }
  return true
})

export default router
