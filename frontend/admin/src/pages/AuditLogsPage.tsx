import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ScrollText, Loader2, AlertCircle, CheckCircle, XCircle, Filter } from 'lucide-react'
import { auditApi } from '@/api/admin'
import { formatDate, cn } from '@/lib/utils'

const ACTIONS = [
  '', 'auth_login_success', 'auth_login_failed', 'auth_account_locked',
  'auth_2fa_enabled', 'auth_2fa_disabled', 'auth_email_verified',
  'auth_password_reset', 'admin_role_created', 'auth_session_revoked',
]

export function AuditLogsPage() {
  const [userId, setUserId] = useState('')
  const [action, setAction] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [successFilter, setSuccessFilter] = useState('')
  const [submitted, setSubmitted] = useState(false)

  const buildFilter = () => ({
    user_id: userId || undefined,
    action: action || undefined,
    start_date: startDate || undefined,
    end_date: endDate || undefined,
    success: successFilter === '' ? undefined : successFilter === 'true',
    limit: 100,
  })

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['audit-logs', userId, action, startDate, endDate, successFilter],
    queryFn: () => auditApi.search(buildFilter()),
    enabled: submitted,
  })

  const handleSearch = () => {
    setSubmitted(true)
    refetch()
  }

  return (
    <div className="space-y-6 fade-in">
      <div>
        <h2 className="text-xl font-semibold text-white">Audit Logs</h2>
        <p className="text-sm text-slate-500 mt-0.5">Búsqueda avanzada del registro de auditoría</p>
      </div>

      {/* Filters */}
      <div className="bg-[#161b27] border border-slate-800/60 rounded-xl p-5">
        <div className="flex items-center gap-2 mb-4">
          <Filter className="w-4 h-4 text-indigo-400" />
          <h3 className="text-sm font-semibold text-slate-200">Filtros</h3>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <div>
            <label className="block text-xs text-slate-500 mb-1">User ID</label>
            <input
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              placeholder="uuid del usuario..."
              className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Acción</label>
            <select
              value={action}
              onChange={(e) => setAction(e.target.value)}
              className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              {ACTIONS.map((a) => (
                <option key={a} value={a}>{a || 'Todas las acciones'}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Estado</label>
            <select
              value={successFilter}
              onChange={(e) => setSuccessFilter(e.target.value)}
              className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              <option value="">Todos</option>
              <option value="true">Exitosos</option>
              <option value="false">Fallidos</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Fecha Inicio</label>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-xs text-slate-500 mb-1">Fecha Fin</label>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>
          <div className="flex items-end">
            <button
              onClick={handleSearch}
              className="w-full px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-medium rounded-lg transition-all"
            >
              Buscar
            </button>
          </div>
        </div>
      </div>

      {/* Results */}
      {isLoading && (
        <div className="flex items-center justify-center p-12">
          <Loader2 className="w-6 h-6 text-indigo-400 animate-spin" />
        </div>
      )}
      {isError && (
        <div className="flex items-center gap-2 p-5 text-sm text-rose-400 bg-[#161b27] border border-slate-800/60 rounded-xl">
          <AlertCircle className="w-4 h-4" />
          Error al cargar los logs. Verifica que el servicio esté disponible.
        </div>
      )}
      {submitted && data && (
        <div className="bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden">
          <div className="flex items-center justify-between px-5 py-3 border-b border-slate-800/60">
            <div className="flex items-center gap-2">
              <ScrollText className="w-4 h-4 text-indigo-400" />
              <span className="text-sm font-medium text-slate-200">Resultados</span>
            </div>
            <span className="text-xs text-slate-600">{data.length} entradas</span>
          </div>
          {data.length === 0 ? (
            <div className="p-10 text-center text-sm text-slate-600">No se encontraron logs con los filtros aplicados.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="border-b border-slate-800/60">
                    {['Fecha', 'Acción', 'Usuario', 'IP', 'País', 'Estado', 'Error'].map((h) => (
                      <th key={h} className="px-4 py-3 text-left text-slate-500 font-medium whitespace-nowrap">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/40">
                  {data.map((log) => (
                    <tr key={log.id} className="hover:bg-slate-800/20 transition-colors">
                      <td className="px-4 py-3 text-slate-500 font-mono whitespace-nowrap">{formatDate(log.created_at)}</td>
                      <td className="px-4 py-3 font-mono text-slate-300 whitespace-nowrap">{log.action}</td>
                      <td className="px-4 py-3 text-slate-400 max-w-32 truncate">{log.user_id || '—'}</td>
                      <td className="px-4 py-3 font-mono text-slate-500 whitespace-nowrap">{log.ip_address}</td>
                      <td className="px-4 py-3 text-slate-500">{log.country || '—'}</td>
                      <td className="px-4 py-3">
                        <span className={cn(
                          'inline-flex items-center gap-1 px-2 py-0.5 rounded-full border text-xs',
                          log.success
                            ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400'
                            : 'bg-rose-500/10 border-rose-500/20 text-rose-400',
                        )}>
                          {log.success
                            ? <CheckCircle className="w-3 h-3" />
                            : <XCircle className="w-3 h-3" />}
                          {log.success ? 'OK' : 'Fallo'}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-slate-600 max-w-40 truncate">{log.error_msg || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {!submitted && (
        <div className="text-center py-12 text-slate-600">
          <ScrollText className="w-10 h-10 mx-auto mb-3 opacity-30" />
          <p className="text-sm">Aplica filtros y presiona "Buscar" para ver los logs</p>
        </div>
      )}
    </div>
  )
}
