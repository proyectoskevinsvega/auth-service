// API Types matching the backend domain models

export interface User {
  id: string
  tenant_id: string
  username: string
  email: string
  roles: string[]
  permissions: string[]
  attributes?: Record<string, unknown>
  email_verified: boolean
  two_fa_enabled: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Session {
  id: string
  user_id: string
  tenant_id: string
  ip_address: string
  user_agent: string
  country: string
  created_at: string
  last_used_at: string
  expires_at: string
}

export interface AuditLogEntry {
  id: string
  tenant_id: string
  user_id: string
  action: string
  ip_address: string
  user_agent: string
  country: string
  success: boolean
  error_msg?: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface Role {
  id: string
  tenant_id: string
  name: string
  description?: string
  permissions: string[]
  created_at: string
}

export interface Permission {
  id: string
  tenant_id: string
  name: string
  description?: string
  resource: string
  action: string
  created_at: string
}

// Compliance Report Types
export interface GDPRDataExport {
  user: User
  audit_logs: AuditLogEntry[]
  active_sessions: Session[]
  exported_at: string
}

export interface SOC2AuditReport {
  tenant_id: string
  period: string
  summary: Record<string, unknown>
  admin_logs: AuditLogEntry[]
  generated_at: string
}

export interface HIPAAReport {
  tenant_id: string
  security_events: AuditLogEntry[]
  access_logs: AuditLogEntry[]
  generated_at: string
}

// Request / Response types
export interface LoginRequest {
  identifier: string
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  user: User
}

export interface CreateRoleRequest {
  name: string
  description?: string
}

export interface CreatePermissionRequest {
  name: string
  description?: string
  resource: string
  action: string
}

export interface AuditSearchFilter {
  user_id?: string
  action?: string
  start_date?: string
  end_date?: string
  success?: boolean
  limit?: number
  offset?: number
}

export interface HealthResponse {
  status: string
  timestamp: string
}

export interface ErrorResponse {
  error: string
  code: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}

export interface M2MCertificate {
  client_id: string
  certificate_pem: string
  private_key_pem?: string
  expires_at: string
}

export interface IssueM2MCertificateRequest {
  client_id: string
  organization?: string
  validity_days?: number
}

export interface SignCSRRequest {
  csr_pem: string
  validity_days?: number
}
