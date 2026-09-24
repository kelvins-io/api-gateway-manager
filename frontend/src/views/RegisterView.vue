<template>
  <div class="auth-page">
    <el-card class="auth-card">
      <h2>注册</h2>
      <p class="hint">首个注册用户将自动成为系统管理员</p>
      <el-form :model="form" @submit.prevent="onSubmit">
        <el-form-item label="用户名">
          <el-input v-model="form.username" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-form-item label="确认密码">
          <el-input v-model="form.confirm" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width: 100%">
          注册
        </el-button>
      </el-form>
      <div class="footer">
        已有账号？
        <router-link to="/login">登录</router-link>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'

const store = useUserStore()
const router = useRouter()
const loading = ref(false)
const form = reactive({ username: '', password: '', confirm: '' })

async function onSubmit() {
  if (form.username.length < 3) {
    ElMessage.warning('用户名至少 3 个字符')
    return
  }
  if (form.password.length < 6) {
    ElMessage.warning('密码至少 6 个字符')
    return
  }
  if (form.password !== form.confirm) {
    ElMessage.warning('两次密码不一致')
    return
  }
  loading.value = true
  try {
    await store.register(form.username, form.password)
    ElMessage.success('注册成功')
    router.push('/')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2d3d 0%, #3a6073 100%);
}
.auth-card {
  width: 380px;
}
.auth-card h2 {
  margin: 0 0 8px;
  text-align: center;
}
.hint {
  margin: 0 0 16px;
  text-align: center;
  color: #909399;
  font-size: 13px;
}
.footer {
  margin-top: 16px;
  text-align: center;
  color: #909399;
}
.footer a {
  color: #409eff;
}
</style>
