import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Shield, Plus, Loader2, Check, AlertCircle, ChevronRight } from 'lucide-react'
import { rbacApi } from '@/api/admin'
import type { CreatePermissionRequest, CreateRoleRequest } from '@/types/api'
import { cn } from '@/lib/utils'
import toast from 'react-hot-toast'

function Modal({
  title,
  onClose,
  children,
}: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 fade-in">
      <div className="bg-[#1e2433] border border-slate-700/60 rounded-2xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-700/60">
          <h3 className="text-sm font-semibold text-slate-200">{title}</h3>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 text-lg leading-none">×</button>
        </div>
        <div className="p-5">{children}</div>
      </div>
    </div>
  )
}

type Tab = 'roles' | 'permissions'

export function RBACPage() {
  const [tab, setTab] = useState<Tab>('roles')
  const [showRoleModal, setShowRoleModal] = useState(false)
  const [showPermModal, setShowPermModal] = useState(false)
  const [roleName, setRoleName] = useState('')
  const [roleDesc, setRoleDesc] = useState('')
  const [permName, setPermName] = useState('')
  const [permResource, setPermResource] = useState('')
  const [permAction, setPermAction] = useState('')
  const [permDesc, setPermDesc] = useState('')

  const qc = useQueryClient()

  const rolesQ = useQuery({ queryKey: ['roles'], queryFn: rbacApi.listRoles })
  const permsQ = useQuery({ queryKey: ['permissions'], queryFn: rbacApi.listPermissions })

  const createRoleMut = useMutation({
    mutationFn: (data: CreateRoleRequest) => rbacApi.createRole(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['roles'] })
      toast.success('Rol creado correctamente')
      setShowRoleModal(false)
      setRoleName('')
      setRoleDesc('')
    },
    onError: () => toast.error('Error al crear el rol'),
  })

  const createPermMut = useMutation({
    mutationFn: (data: CreatePermissionRequest) => rbacApi.createPermission(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['permissions'] })
      toast.success('Permiso creado correctamente')
      setShowPermModal(false)
      setPermName('')
      setPermResource('')
      setPermAction('')
      setPermDesc('')
    },
    onError: () => toast.error('Error al crear el permiso'),
  })

  return (
    <div className="space-y-6 fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-white">Roles & Permisos</h2>
          <p className="text-sm text-slate-500 mt-0.5">Control de acceso basado en roles (RBAC)</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setShowRoleModal(true)}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-all"
          >
            <Plus className="w-3.5 h-3.5" /> Nuevo Rol
          </button>
          <button
            onClick={() => setShowPermModal(true)}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-medium bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg transition-all"
          >
            <Plus className="w-3.5 h-3.5" /> Nuevo Permiso
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-[#161b27] border border-slate-800/60 rounded-xl p-1 w-fit">
        {(['roles', 'permissions'] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cn(
              'px-4 py-1.5 text-xs font-medium rounded-lg capitalize transition-all',
              tab === t
                ? 'bg-indigo-600 text-white shadow'
                : 'text-slate-400 hover:text-slate-200',
            )}
          >
            {t === 'roles' ? 'Roles' : 'Permisos'}
          </button>
        ))}
      </div>

      {/* Roles Tab */}
      {tab === 'roles' && (
        <div className="bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden">
          <div className="px-5 py-3 border-b border-slate-800/60 flex items-center gap-2">
            <Shield className="w-4 h-4 text-indigo-400" />
            <span className="text-sm font-medium text-slate-200">Roles</span>
            {rolesQ.data && (
              <span className="ml-auto text-xs text-slate-600">{rolesQ.data.length} roles</span>
            )}
          </div>
          {rolesQ.isLoading && (
            <div className="flex items-center justify-center p-10">
              <Loader2 className="w-5 h-5 text-indigo-400 animate-spin" />
            </div>
          )}
          {rolesQ.isError && (
            <div className="flex items-center gap-2 p-5 text-sm text-rose-400">
              <AlertCircle className="w-4 h-4" />
              Error al cargar roles
            </div>
          )}
          {rolesQ.data?.length === 0 && (
            <div className="p-10 text-center text-sm text-slate-600">
              No hay roles creados aún.
            </div>
          )}
          <div className="divide-y divide-slate-800/40">
            {rolesQ.data?.map((role) => (
              <div key={role.id} className="flex items-center gap-4 px-5 py-3.5 hover:bg-slate-800/20 transition-colors">
                <div className="w-8 h-8 rounded-full bg-indigo-600/10 border border-indigo-500/20 flex items-center justify-center shrink-0">
                  <Shield className="w-4 h-4 text-indigo-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-slate-200">{role.name}</p>
                  {role.description && <p className="text-xs text-slate-500">{role.description}</p>}
                </div>
                <div className="flex items-center gap-1.5">
                  {(role.permissions ?? []).length > 0 && (
                    <span className="text-xs bg-slate-800 text-slate-400 px-2 py-0.5 rounded-md">
                      {role.permissions.length} permisos
                    </span>
                  )}
                  <ChevronRight className="w-4 h-4 text-slate-700" />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Permissions Tab */}
      {tab === 'permissions' && (
        <div className="bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden">
          <div className="px-5 py-3 border-b border-slate-800/60 flex items-center gap-2">
            <Check className="w-4 h-4 text-emerald-400" />
            <span className="text-sm font-medium text-slate-200">Permisos</span>
            {permsQ.data && (
              <span className="ml-auto text-xs text-slate-600">{permsQ.data.length} permisos</span>
            )}
          </div>
          {permsQ.isLoading && (
            <div className="flex items-center justify-center p-10">
              <Loader2 className="w-5 h-5 text-indigo-400 animate-spin" />
            </div>
          )}
          {permsQ.data?.length === 0 && (
            <div className="p-10 text-center text-sm text-slate-600">
              No hay permisos creados aún.
            </div>
          )}
          <div className="divide-y divide-slate-800/40">
            {permsQ.data?.map((perm) => (
              <div key={perm.id} className="flex items-center gap-4 px-5 py-3.5 hover:bg-slate-800/20 transition-colors">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-slate-200">{perm.name}</p>
                  {perm.description && <p className="text-xs text-slate-500">{perm.description}</p>}
                </div>
                <div className="flex items-center gap-1.5 text-xs">
                  <span className="bg-slate-800 text-slate-400 px-2 py-0.5 rounded font-mono">{perm.resource}</span>
                  <span className="text-slate-600">:</span>
                  <span className="bg-indigo-900/30 text-indigo-400 px-2 py-0.5 rounded font-mono">{perm.action}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Create Role Modal */}
      {showRoleModal && (
        <Modal title="Crear Nuevo Rol" onClose={() => setShowRoleModal(false)}>
          <div className="space-y-4">
            <div>
              <label className="block text-xs text-slate-400 mb-1">Nombre del Rol *</label>
              <input
                value={roleName}
                onChange={(e) => setRoleName(e.target.value)}
                placeholder="ej: editor, billing-admin..."
                className="w-full bg-[#161b27] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Descripción</label>
              <input
                value={roleDesc}
                onChange={(e) => setRoleDesc(e.target.value)}
                placeholder="Descripción del rol..."
                className="w-full bg-[#161b27] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <button
              onClick={() => createRoleMut.mutate({ name: roleName, description: roleDesc })}
              disabled={!roleName || createRoleMut.isPending}
              className="w-full flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60 text-white text-sm font-medium py-2 rounded-lg transition-all"
            >
              {createRoleMut.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
              Crear Rol
            </button>
          </div>
        </Modal>
      )}

      {/* Create Permission Modal */}
      {showPermModal && (
        <Modal title="Crear Nuevo Permiso" onClose={() => setShowPermModal(false)}>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-slate-400 mb-1">Recurso *</label>
                <input
                  value={permResource}
                  onChange={(e) => setPermResource(e.target.value)}
                  placeholder="ej: users"
                  className="w-full bg-[#161b27] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-400 mb-1">Acción *</label>
                <input
                  value={permAction}
                  onChange={(e) => setPermAction(e.target.value)}
                  placeholder="ej: write"
                  className="w-full bg-[#161b27] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Nombre del Permiso *</label>
              <input
                value={permName}
                onChange={(e) => setPermName(e.target.value)}
                placeholder="ej: users:write"
                className="w-full bg-[#161b27] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Descripción</label>
              <input
                value={permDesc}
                onChange={(e) => setPermDesc(e.target.value)}
                placeholder="Descripción..."
                className="w-full bg-[#161b27] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <button
              onClick={() =>
                createPermMut.mutate({ name: permName, resource: permResource, action: permAction, description: permDesc })
              }
              disabled={!permName || !permResource || !permAction || createPermMut.isPending}
              className="w-full flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60 text-white text-sm font-medium py-2 rounded-lg transition-all"
            >
              {createPermMut.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
              Crear Permiso
            </button>
          </div>
        </Modal>
      )}
    </div>
  )
}
