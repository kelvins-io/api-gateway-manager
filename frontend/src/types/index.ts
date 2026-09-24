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
  created_at?: string
}

export interface SpaceMember {
  id: number
  space_id: number
  user_id: number
  role: string
  user?: User
}

export interface Gateway {
  id: number
  name: string
  admin_api: string
  network_zone: string
}

export interface ApiGroup {
  id: number
  space_id: number
  gateway_id: number
  name: string
  gateway?: Gateway
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
  status: string
  current_version: string
  kong_service_id?: string
  kong_route_id?: string
  group?: ApiGroup
}

export interface ApiVersion {
  id: number
  api_id: number
  version: string
  config_snapshot: Record<string, unknown>
  published_at: string
}
