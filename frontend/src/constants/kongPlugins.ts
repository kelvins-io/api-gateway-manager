/** Kong Gateway 3.4.2 OSS bundled plugins, grouped like Kong Manager. */

export interface PluginCategory {
  /** Bilingual label, e.g. "认证 / Authentication" */
  label: string
  plugins: string[]
}

export const pluginCategories: PluginCategory[] = [
  {
    label: '认证 / Authentication',
    plugins: ['basic-auth', 'hmac-auth', 'jwt', 'key-auth', 'ldap-auth', 'oauth2', 'session'],
  },
  {
    label: '安全 / Security',
    plugins: ['acme', 'bot-detection', 'cors', 'ip-restriction'],
  },
  {
    label: '流量控制 / Traffic Control',
    plugins: [
      'acl',
      'proxy-cache',
      'rate-limiting',
      'request-size-limiting',
      'request-termination',
      'response-ratelimiting',
    ],
  },
  {
    label: '无服务器 / Serverless',
    plugins: ['aws-lambda', 'azure-functions', 'pre-function', 'post-function'],
  },
  {
    label: '分析与监控 / Analytics & Monitoring',
    plugins: ['datadog', 'opentelemetry', 'prometheus', 'statsd', 'zipkin'],
  },
  {
    label: '转换 / Transformations',
    plugins: [
      'correlation-id',
      'grpc-gateway',
      'grpc-web',
      'request-transformer',
      'response-transformer',
    ],
  },
  {
    label: '日志 / Logging',
    plugins: ['file-log', 'http-log', 'loggly', 'syslog', 'tcp-log', 'udp-log'],
  },
]

/** Plugins with dedicated config forms in PluginsView. */
export const formPlugins = new Set([
  'rate-limiting',
  'cors',
  'key-auth',
  'acl',
  'ip-restriction',
  'request-size-limiting',
  'jwt',
  'basic-auth',
  'hmac-auth',
  'request-termination',
  'correlation-id',
  'request-transformer',
  'response-transformer',
])

export function hasPluginForm(plugin: string) {
  return formPlugins.has(plugin)
}
