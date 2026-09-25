<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-brand">API GATEWAY</div>
      <h1>注册</h1>
      <p class="auth-hint">首个注册用户将自动成为系统管理员</p>
      <el-form :model="form" @submit.prevent="onSubmit" label-position="top">
        <el-form-item label="用户名" required>
          <el-input v-model="form.username" placeholder="用户名" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input
            v-model="form.password"
            type="password"
            show-password
            placeholder="至少 6 个字符"
            autocomplete="new-password"
          />
        </el-form-item>
        <el-form-item label="确认密码" required>
          <el-input
            v-model="form.confirm"
            type="password"
            show-password
            placeholder="再次输入密码"
            autocomplete="new-password"
          />
        </el-form-item>
        <el-button type="primary" class="auth-submit" native-type="submit" :loading="loading">
          注册
        </el-button>
      </el-form>
      <div class="auth-footer">
        已有账号？
        <router-link to="/login">去登录</router-link>
      </div>
    </div>
    <a class="contact" href="mailto:1225807604@qq.com">联系我们：1225807604@qq.com</a>
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
  --bg: #f4f6fb;
  --panel: #ffffff;
  --line: #e6eaf2;
  --text: #1f2a37;
  --muted: #6b7280;
  --brand: #3b6dff;

  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  position: relative;
  color: var(--text);
  background:
    radial-gradient(1200px 600px at 10% -10%, rgba(59, 109, 255, 0.18), transparent 55%),
    radial-gradient(900px 500px at 100% 0%, rgba(17, 24, 39, 0.08), transparent 50%),
    var(--bg);
}

.auth-card {
  width: 100%;
  max-width: 400px;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 16px;
  padding: 28px 28px 24px;
  box-shadow: 0 12px 40px rgba(17, 24, 39, 0.06);
}

.auth-brand {
  color: var(--brand);
  font-weight: 700;
  letter-spacing: 0.04em;
  margin-bottom: 12px;
}

.auth-card h1 {
  margin: 0 0 6px;
  font-size: 24px;
}

.auth-hint {
  margin: 0 0 20px;
  color: var(--muted);
  font-size: 14px;
}

.auth-submit {
  width: 100%;
}

.auth-footer {
  margin-top: 16px;
  text-align: center;
  color: var(--muted);
  font-size: 14px;
}

.auth-footer a {
  color: var(--brand);
  text-decoration: none;
}

.contact {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--muted);
  text-decoration: none;
}

.contact:hover {
  color: var(--brand);
}
</style>
