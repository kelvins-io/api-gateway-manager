export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface User {
  id: number
  username: string
  role: string
  created_at?: string
}

export interface Space {
  id: number
  name: string
  description: string
  prefix: string
  owner_id: number
  status: string
  member_status?: string
  group_count?: number
  created_at?: string
}

export interface SpaceMember {
  id: number
  space_id: number
  user_id: number
  role: string
  status: string
  user?: User
}

export interface Gateway {
  id: number
  name: string
  admin_api: string
  domain: string
  network_zone: string
  shared?: boolean
}

export interface ApiGroup {
  id: number
  space_id: number
  gateway_id: number
  name: string
  gateway?: Gateway
  space?: { id?: number; name?: string; prefix: string }
  api_count?: number
}

export interface UpstreamTarget {
  id?: number
  upstream_id?: number
  target: string
  weight: number
}

export interface UpstreamHealthSide {
  type?: string
  http_path?: string
  timeout?: number
  concurrency?: number
  https_verify_certificate?: boolean
  healthy_interval?: number
  healthy_successes?: number
  healthy_http_statuses?: number[]
  unhealthy_interval?: number
  unhealthy_http_failures?: number
  unhealthy_tcp_failures?: number
  unhealthy_timeouts?: number
  unhealthy_http_statuses?: number[]
}

export interface UpstreamItem {
  id: number
  space_id: number
  name: string
  algorithm: string
  slots: number
  hash_on?: string
  hash_fallback?: string
  hash_on_header?: string
  hash_fallback_header?: string
  hash_on_cookie?: string
  hash_on_cookie_path?: string
  hash_on_query_arg?: string
  hash_fallback_query_arg?: string
  hash_on_uri_capture?: string
  hash_fallback_uri_capture?: string
  healthchecks?: {
    active?: UpstreamHealthSide
    passive?: UpstreamHealthSide
    threshold?: number
  }
  targets?: UpstreamTarget[]
}

export interface ConsumerCredential {
  id?: number
  consumer_id?: number
  plugin: string
  config: Record<string, string>
}

export interface ConsumerItem {
  id: number
  space_id: number
  username: string
  custom_id: string
  created_at?: string
  credentials?: ConsumerCredential[]
  apis?: ApiItem[]
  space?: { id: number; name: string }
}

export interface PluginItem {
  id: number
  space_id: number
  name: string
  plugin: string
  config: Record<string, unknown>
  enabled: boolean
  api_count?: number
}

export interface ApiItem {
  id: number
  group_id: number
  name: string
  access_path: string
  access_methods: string
  access_protocols: string
  access_hosts?: string
  access_headers?: Record<string, string[]>
  upstream_url?: string
  service_protocol: string
  service_host_kind: string
  service_host: string
  service_upstream_id?: number
  service_port: number
  service_path: string
  service_retries: number
  service_connect_timeout: number
  service_write_timeout: number
  service_read_timeout: number
  access_strip_path: boolean
  request_buffering?: boolean
  response_buffering?: boolean
  auth_enabled: boolean
  auth_plugin: string
  auth_config?: Record<string, unknown>
  status: string
  current_version: string
  shared?: boolean
  kong_service_id?: string
  kong_route_id?: string
  group?: ApiGroup
  plugins?: PluginItem[]
  consumers?: ConsumerItem[]
}

export interface ApiVersion {
  id: number
  api_id: number
  version: string
  config_snapshot: Record<string, unknown>
  published_at: string
}
