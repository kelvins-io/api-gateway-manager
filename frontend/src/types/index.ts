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

export interface ApiItem {
  id: number
  group_id: number
  name: string
  path: string
  methods: string
  upstream_url: string
  strip_path: boolean
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
