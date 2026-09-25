<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="brand">API Gateway Manager</div>
      <el-menu
        :key="menuKey"
        :default-active="active"
        router
        background-color="transparent"
        text-color="#3b4a63"
        active-text-color="#3b6dff"
      >
        <el-menu-item index="/spaces">
          <el-icon><OfficeBuilding /></el-icon>
          <span>空间管理</span>
        </el-menu-item>
        <el-menu-item index="/market">
          <el-icon><Shop /></el-icon>
          <span>API 市场</span>
        </el-menu-item>
        <el-menu-item v-if="isSystemAdmin" index="/gateways">
          <el-icon><Connection /></el-icon>
          <span>网关管理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-title">{{ title }}</div>
        <div class="header-right">
          <el-tag size="small" :type="isSystemAdmin ? 'danger' : 'info'">{{ roleLabel }}</el-tag>
          <span class="username">{{ user?.username }}</span>
          <el-button text type="danger" @click="onLogout">退出</el-button>
        </div>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'

const store = useUserStore()
const { user, isSystemAdmin } = storeToRefs(store)
const route = useRoute()
const router = useRouter()

const menuKey = computed(() => `menu-${user.value?.role || 'none'}-${isSystemAdmin.value}`)

const active = computed(() => {
  if (route.path.startsWith('/gateways')) return '/gateways'
  if (route.path.startsWith('/market')) return '/market'
  if (route.path.startsWith('/spaces') || route.path.startsWith('/groups') || route.path.startsWith('/upstreams') || route.path.startsWith('/consumers') || route.path.startsWith('/plugins')) return '/spaces'
  return route.path
})

const title = computed(() => {
  if (route.name === 'gateways') return '网关管理'
  if (route.name === 'market') return 'API 市场'
  if (route.name === 'groups') return 'API 分组'
  if (route.name === 'apis') return 'API 管理'
  if (route.name === 'upstreams') return 'Upstream'
  if (route.name === 'consumers') return 'Consumers'
  if (route.name === 'plugins') return 'Plugins'
  if (route.name === 'space-members') return '空间成员'
  return '空间管理'
})

const roleLabel = computed(() => {
  const map: Record<string, string> = {
    system_admin: '系统管理员',
    space_admin: '空间管理员',
    member: '成员',
  }
  return map[user.value?.role || ''] || user.value?.role || ''
})

onMounted(async () => {
  try {
    await store.fetchMe()
    await store.loadSpaces()
  } catch {
    // interceptor handles 401
  }
})

function onLogout() {
  store.logout()
  router.push('/login')
}
</script>

<style scoped>
.layout {
  height: 100%;
  min-height: 100vh;
  background: transparent;
}
.aside {
  background: var(--app-sidebar);
  color: var(--app-sidebar-text);
  display: flex;
  flex-direction: column;
  border-right: 1px solid #d7e4fb;
}
.brand {
  padding: 20px 16px;
  font-weight: 700;
  font-size: 15px;
  line-height: 1.4;
  color: #1f2a37;
  border-bottom: 1px solid #d7e4fb;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
}
.header-title {
  font-size: 18px;
  font-weight: 600;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.username {
  color: #334155;
}
.main {
  padding: 20px;
  background: transparent;
}
:deep(.el-container) {
  background: transparent;
}
:deep(.el-menu) {
  border-right: none;
  background: transparent;
}
:deep(.el-menu-item:hover) {
  background: rgba(59, 109, 255, 0.08) !important;
}
:deep(.el-menu-item.is-active) {
  background: rgba(59, 109, 255, 0.14) !important;
  font-weight: 600;
}
</style>
