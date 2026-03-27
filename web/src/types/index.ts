// API 统一响应
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PaginatedData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 领域实体
export interface User {
  id: string
  username: string
  is_super: boolean
  role_id: string | null
  role_name: string
  created_at: string
  updated_at: string
}

export interface RolePermission {
  module: string
  can_read: boolean
  can_write: boolean
}

export interface RoleDetail {
  id: string
  name: string
  permissions: RolePermission[]
  environment_ids: string[]
}

export interface Role {
  id: string
  name: string
  description: string
  permissions: RolePermission[]
  environment_ids: string[]
  user_count: number
  created_at: string
  updated_at: string
}

export interface UserProfile {
  user_id: string
  username: string
  is_super: boolean
  role: RoleDetail | null
}

export interface Environment {
  id: string
  name: string
  key_prefix: string
  config_prefix: string
  gateway_prefix: string
  grpc_prefix: string
  description: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface ConfigRevision {
  id: string
  environment_id: string
  key: string
  value: string
  prev_value: string
  etcd_revision: number
  action: 'create' | 'update' | 'delete'
  operator: string
  comment: string
  created_at: string
  updated_at: string
}

export interface AuditLog {
  id: string
  user_id: string
  username: string
  action: string
  resource_type: string
  resource_key: string
  detail: string
  ip: string
  created_at: string
  updated_at: string
}

// KV
export interface KVItem {
  key: string
  value: string
  version: number
  create_revision: number
  mod_revision: number
}

// Auth
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user_id: string
  username: string
  is_super: boolean
  role: RoleDetail | null
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

// Config
export interface ConfigItem {
  key: string
  value: string
}

export interface ConfigCreateRequest {
  env: string
  key: string
  value: string
  comment: string
}

export interface ConfigUpdateRequest {
  env: string
  key: string
  value: string
  comment: string
}

export interface ConfigRollbackRequest {
  env: string
  key: string
  revision_id: string
}

export interface ImportResult {
  total: number
  success: number
  failed: string[]
}

// Environment
export interface EnvironmentCreateRequest {
  name: string
  key_prefix: string
  config_prefix?: string
  gateway_prefix?: string
  grpc_prefix?: string
  description?: string
  sort_order?: number
}

// User
export interface UserCreateRequest {
  username: string
  password: string
  role_id: string
}

export interface UserUpdateRequest {
  role_id: string
}

// Role
export interface RoleCreateRequest {
  name: string
  description: string
  permissions: RolePermission[]
  environment_ids: string[]
}

export interface RoleUpdateRequest {
  name: string
  description: string
  permissions: RolePermission[]
  environment_ids: string[]
}

// Cluster
export interface ClusterStatus {
  cluster_id: string
  members: ClusterMember[]
  leader: string
}

export interface ClusterMember {
  id: string
  name: string
  peer_urls: string[]
  client_urls: string[]
  is_learner: boolean
}

export interface ClusterMetrics {
  cluster_id: string
  leader_name: string
  db_size: number
  member_count: number
  version: string
  health: Record<string, boolean>
}

// Audit filter
export interface AuditLogFilter {
  user_id?: string
  action?: string
  resource_type?: string
  start_time?: string
  end_time?: string
  page?: number
  page_size?: number
}

// Watch SSE event
export interface WatchEvent {
  type: 'PUT' | 'DELETE' | 'COMPACTED'
  key: string
  value?: string
  revision: number
}

// Gateway
export interface ServiceInstance {
  id: string
  service_name: string
  host: string
  port: number
  weight: number
  version: string
  status: 'up' | 'down'
  registered_at: string
  metadata: Record<string, string>
}

export interface ServiceGroup {
  service_name: string
  instance_count: number
  healthy_count: number
  unhealthy_count: number
  instances: ServiceInstance[]
}

// gRPC Service
export interface GrpcInstance {
  service_name: string
  version: string
  address: string
  env: string
  weight: number
  tags: string[]
  status: 'up' | 'down'
  register_time: number
  instance_id: string
  properties: Record<string, string>
}

export interface GrpcServiceGroup {
  service_name: string
  instance_count: number
  healthy_count: number
  unhealthy_count: number
  instances: GrpcInstance[]
}
