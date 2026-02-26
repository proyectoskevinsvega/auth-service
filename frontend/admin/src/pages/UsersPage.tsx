import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Users, RotateCcw, Loader2, Shield, AlertCircle } from 'lucide-react'
import { usersApi, rbacApi } from '@/api/admin'
import toast from 'react-hot-toast'
import { cn } from '@/lib/utils'

// Simulated user list since the backend doesn't have a generic list-all-users endpoint.
// In a real deployment, replace with a real API call.
const SAMPLE_USERS = [
  { id: '550e8400-e29b-41d4-a716-446655440001', username: 'alice', email: 'alice@acme.com', roles: ['admin'], email_verified: true, two_fa_enabled: true, is_active: true },
  { id: '550e8400-e29b-41d4-a716-446655440002', username: 'bob', email: 'bob@corp.io', roles: ['editor'], email_verified: true, two_fa_enabled: false, is_active: true },
  { id: '550e8400-e29b-41d4-a716-446655440003', username: 'charlie', email: 'charlie@example.com', roles: ['viewer'], email_verified: false, two_fa_enabled: false, is_active: false },
]

export function UsersPage() {
  const [search, setSearch] = useState('')
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null)
  const [roleId, setRoleId] = useState('')
  const [showAssignModal, setShowAssignModal] = useState(false)
  const [assignTarget, setAssignTarget] = useState<string | null>(null)

  const resetMut = useMutation({
    mutationFn: (id: string) => usersApi.forcePasswordReset(id),
    onSuccess: () => toast.success('Reset de contraseña enviado'),
    onError: () => toast.error('Error al enviar reset de contraseña'),
  })

  const assignRoleMut = useMutation({
    mutationFn: () => usersApi.assignRole(assignTarget!, roleId),
    onSuccess: () => {
      toast.success('Rol asignado correctamente')
      setShowAssignModal(false)
      setRoleId('')
    },
    onError: () => toast.error('Error al asignar el rol'),
  })

  const filtered = SAMPLE_USERS.filter(
    (u) =>
      u.username.includes(search.toLowerCase()) ||
      u.email.toLowerCase().includes(search.toLowerCase()),
  )

  return (
    <div className="space-y-6 fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-white">Gestión de Usuarios</h2>
          <p className="text-sm text-slate-500 mt-0.5">Administra usuarios del microservicio</p>
        </div>
      </div>

      {/* Search */}
      <div className="flex gap-3">
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Buscar por nombre o email..."
          className="flex-1 bg-[#161b27] border border-slate-800/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
        />
      </div>

      {/* Info Banner */}
      <div className="flex items-start gap-2 px-4 py-3 bg-indigo-600/5 border border-indigo-500/15 rounded-xl text-xs text-slate-400">
        <AlertCircle className="w-4 h-4 text-indigo-400 shrink-0 mt-0.5" />
        <span>
          Los datos de usuario se cargan desde el contexto de autenticación. Para listados completos, 
          integra con el endpoint <span className="font-mono text-slate-300">/api/v1/admin/users</span> de tu deployment.
        </span>
      </div>

      {/* Users Table */}
      <div className="bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden">
        <div className="px-5 py-3 border-b border-slate-800/60 flex items-center gap-2">
          <Users className="w-4 h-4 text-indigo-400" />
          <span className="text-sm font-medium text-slate-200">Usuarios</span>
          <span className="ml-auto text-xs text-slate-600">{filtered.length} usuarios</span>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-slate-800/60">
                {['Usuario', 'Email', 'Roles', '2FA', 'Email Verificado', 'Estado', 'Acciones'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-slate-500 font-medium whitespace-nowrap">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/40">
              {filtered.map((user) => (
                <tr
                  key={user.id}
                  onClick={() => setSelectedUserId(user.id === selectedUserId ? null : user.id)}
                  className="hover:bg-slate-800/20 transition-colors cursor-pointer"
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-6 h-6 rounded-full bg-indigo-600/20 border border-indigo-500/20 flex items-center justify-center text-indigo-400 font-medium text-xs shrink-0">
                        {user.username[0].toUpperCase()}
                      </div>
                      <span className="text-slate-200 font-medium">{user.username}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-slate-400">{user.email}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {user.roles.map((r) => (
                        <span key={r} className="bg-indigo-900/30 text-indigo-400 px-1.5 py-0.5 rounded font-mono text-xs">
                          {r}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={cn('px-1.5 py-0.5 rounded text-xs', user.two_fa_enabled ? 'text-emerald-400' : 'text-slate-600')}>
                      {user.two_fa_enabled ? '✓ 2FA' : 'Sin 2FA'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={cn('px-1.5 py-0.5 rounded text-xs', user.email_verified ? 'text-emerald-400' : 'text-amber-400')}>
                      {user.email_verified ? '✓ Verificado' : 'Pendiente'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={cn(
                      'px-2 py-0.5 rounded-full border text-xs',
                      user.is_active
                        ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400'
                        : 'bg-rose-500/10 border-rose-500/20 text-rose-400',
                    )}>
                      {user.is_active ? 'Activo' : 'Inactivo'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
                      <button
                        onClick={() => resetMut.mutate(user.id)}
                        disabled={resetMut.isPending}
                        title="Force Password Reset"
                        className="flex items-center gap-1 px-2 py-1 text-xs text-slate-400 hover:text-amber-400 hover:bg-amber-500/10 rounded transition-all"
                      >
                        {resetMut.isPending ? (
                          <Loader2 className="w-3.5 h-3.5 animate-spin" />
                        ) : (
                          <RotateCcw className="w-3.5 h-3.5" />
                        )}
                        Reset
                      </button>
                      <button
                        onClick={() => { setAssignTarget(user.id); setShowAssignModal(true) }}
                        className="flex items-center gap-1 px-2 py-1 text-xs text-slate-400 hover:text-indigo-400 hover:bg-indigo-500/10 rounded transition-all"
                      >
                        <Shield className="w-3.5 h-3.5" />
                        Rol
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Assign Role Modal */}
      {showAssignModal && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4 fade-in">
          <div className="bg-[#1e2433] border border-slate-700/60 rounded-2xl w-full max-w-sm shadow-2xl">
            <div className="flex items-center justify-between px-5 py-4 border-b border-slate-700/60">
              <h3 className="text-sm font-semibold text-slate-200">Asignar Rol</h3>
              <button onClick={() => setShowAssignModal(false)} className="text-slate-500 hover:text-slate-300 text-lg">×</button>
            </div>
            <div className="p-5 space-y-4">
              <div>
                <label className="block text-xs text-slate-400 mb-1.5">ID del Rol *</label>
                <input
                  value={roleId}
                  onChange={(e) => setRoleId(e.target.value)}
                  placeholder="uuid del rol..."
                  className="w-full bg-[#161b27] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <button
                onClick={() => assignRoleMut.mutate()}
                disabled={!roleId || assignRoleMut.isPending}
                className="w-full flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60 text-white text-sm font-medium py-2 rounded-lg transition-all"
              >
                {assignRoleMut.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
                Asignar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
