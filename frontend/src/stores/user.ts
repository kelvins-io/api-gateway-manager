import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, Space } from '@/types'
import * as authApi from '@/api/auth'
import * as spaceApi from '@/api/space'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<User | null>(null)
  const spaces = ref<Space[]>([])
  const currentSpaceId = ref<number | null>(
    localStorage.getItem('currentSpaceId') ? Number(localStorage.getItem('currentSpaceId')) : null,
  )

  const isLoggedIn = computed(() => !!token.value)
  const isSystemAdmin = computed(() => user.value?.role === 'system_admin')
  const currentSpace = computed(() => spaces.value.find((s) => s.id === currentSpaceId.value) || null)

  function setAuth(t: string, u: User) {
    token.value = t
    user.value = u
    localStorage.setItem('token', t)
    localStorage.setItem('user', JSON.stringify(u))
  }

  function loadFromStorage() {
    const raw = localStorage.getItem('user')
    if (raw) {
      try {
        user.value = JSON.parse(raw)
      } catch {
        user.value = null
      }
    }
  }

  async function login(username: string, password: string) {
    const data = await authApi.login(username, password)
    setAuth(data.token, data.user)
  }

  async function register(username: string, password: string) {
    const data = await authApi.register(username, password)
    setAuth(data.token, data.user)
  }

  async function fetchMe() {
    const me = await authApi.fetchMe()
    user.value = me
    localStorage.setItem('user', JSON.stringify(me))
  }

  function logout() {
    token.value = ''
    user.value = null
    spaces.value = []
    currentSpaceId.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    localStorage.removeItem('currentSpaceId')
  }

  async function loadSpaces() {
    spaces.value = (await spaceApi.listSpaces()) || []
    if (spaces.value.length && !spaces.value.find((s) => s.id === currentSpaceId.value)) {
      setCurrentSpace(spaces.value[0].id)
    }
  }

  function setCurrentSpace(id: number) {
    currentSpaceId.value = id
    localStorage.setItem('currentSpaceId', String(id))
  }

  loadFromStorage()

  return {
    token,
    user,
    spaces,
    currentSpaceId,
    currentSpace,
    isLoggedIn,
    isSystemAdmin,
    login,
    register,
    fetchMe,
    logout,
    loadSpaces,
    setCurrentSpace,
  }
})
