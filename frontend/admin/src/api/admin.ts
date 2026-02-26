import { apiClient } from './client'
import type {
  AuditLogEntry,
  AuditSearchFilter,
  CreatePermissionRequest,
  CreateRoleRequest,
  GDPRDataExport,
  HIPAAReport,
  IssueM2MCertificateRequest,
  M2MCertificate,
  Permission,
  Role,
  SignCSRRequest,
  SOC2AuditReport,
  User,
} from '@/types/api'

// --- Users ---
export const usersApi = {
  forcePasswordReset: async (userId: string): Promise<void> => {
    await apiClient.post(`/admin/users/${userId}/force-reset`)
  },
  assignRole: async (userId: string, roleId: string): Promise<void> => {
    await apiClient.post(`/admin/users/${userId}/roles`, { role_id: roleId })
  },
}

// --- RBAC ---
export const rbacApi = {
  listRoles: async (): Promise<Role[]> => {
    const res = await apiClient.get<Role[]>('/admin/roles')
    return res.data
  },
  createRole: async (data: CreateRoleRequest): Promise<Role> => {
    const res = await apiClient.post<Role>('/admin/roles', data)
    return res.data
  },
  listPermissions: async (): Promise<Permission[]> => {
    const res = await apiClient.get<Permission[]>('/admin/permissions')
    return res.data
  },
  createPermission: async (data: CreatePermissionRequest): Promise<Permission> => {
    const res = await apiClient.post<Permission>('/admin/permissions', data)
    return res.data
  },
  addPermissionToRole: async (roleId: string, permissionId: string): Promise<void> => {
    await apiClient.post(`/admin/roles/${roleId}/permissions`, { permission_id: permissionId })
  },
}

// --- Compliance ---
export const complianceApi = {
  generateGDPR: async (userId: string): Promise<GDPRDataExport> => {
    const res = await apiClient.get<GDPRDataExport>(`/admin/compliance/gdpr/${userId}`)
    return res.data
  },
  generateSOC2: async (startDate: string, endDate: string): Promise<SOC2AuditReport> => {
    const res = await apiClient.get<SOC2AuditReport>('/admin/compliance/soc2', {
      params: { start_date: startDate, end_date: endDate },
    })
    return res.data
  },
  generateHIPAA: async (startDate: string, endDate: string): Promise<HIPAAReport> => {
    const res = await apiClient.get<HIPAAReport>('/admin/compliance/hipaa', {
      params: { start_date: startDate, end_date: endDate },
    })
    return res.data
  },
}

// --- Audit Logs ---
export const auditApi = {
  search: async (filter: AuditSearchFilter): Promise<AuditLogEntry[]> => {
    const res = await apiClient.get<AuditLogEntry[]>('/admin/audit-logs', { params: filter })
    return res.data
  },
}

// --- M2M / Certificates ---
export const m2mApi = {
  issueCertificate: async (data: IssueM2MCertificateRequest): Promise<M2MCertificate> => {
    const res = await apiClient.post<M2MCertificate>('/admin/m2m/certificates', data)
    return res.data
  },
  signCSR: async (data: SignCSRRequest): Promise<M2MCertificate> => {
    const res = await apiClient.post<M2MCertificate>('/admin/m2m/certificates/sign', data)
    return res.data
  },
}

// --- Health ---
export const healthApi = {
  check: async (): Promise<{ status: string }> => {
    const res = await apiClient.get('/health')
    return res.data
  },
}

// --- Auth introspection (re-use for listings) ---
export const adminSessionsApi = {
  getMyUser: async (): Promise<User> => {
    const res = await apiClient.get<User>('/auth/me')
    return res.data
  },
}
