import { defineConfig, loadEnv, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import type { IncomingMessage, ServerResponse } from 'node:http'

// 与 frontend/docker/normalize-base-path.sh 保持一致。
function normalizeBasePath(input: string | undefined): string {
  const value = (input ?? '').trim()
  if (!value || value === '/') return '/'
  let path = value.startsWith('/') ? value : `/${value}`
  if (path.length > 1 && path.endsWith('/')) path = path.slice(0, -1)
  if (!/^\/[A-Za-z0-9/_-]+$/.test(path) || path.includes('//')) {
    throw new Error(`无效的 VITE_BASE_PATH: ${input}`)
  }
  return `${path}/`
}

function redirectRootToBase(base: string): Plugin {
  const noslash = base.replace(/\/$/, '')
  const redirect = (req: IncomingMessage, res: ServerResponse, next: () => void) => {
    const path = req.url?.split('?')[0] ?? ''
    if (path !== '/' && path !== noslash) {
      next()
      return
    }
    const query = req.url?.includes('?') ? req.url.slice(req.url.indexOf('?')) : ''
    res.statusCode = path === '/' ? 302 : 301
    res.setHeader('Location', `${base}${query}`)
    res.end()
  }
  return {
    name: 'redirect-root-to-base',
    configureServer(server) {
      if (base !== '/') server.middlewares.use(redirect)
    },
    configurePreviewServer(server) {
      if (base !== '/') server.middlewares.use(redirect)
    },
  }
}

export default defineConfig(({ mode }) => {
  const fileEnv = loadEnv(mode, process.cwd(), 'VITE_')
  const base = normalizeBasePath(process.env.VITE_BASE_PATH || fileEnv.VITE_BASE_PATH)

  return {
    base,
    plugins: [vue(), redirectRootToBase(base)],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: 5373,
      host: true,
      allowedHosts: true,
      proxy: {
        '/api': {
          target: 'http://localhost:10000',
          changeOrigin: true,
        },
      },
    },
  }
})
