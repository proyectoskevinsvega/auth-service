import { useQuery } from '@tanstack/react-query'
import {
  Users,
  Shield,
  Activity,
  AlertTriangle,
  CheckCircle,
  XCircle,
  TrendingUp,
  Server,
} from 'lucide-react'
import { healthApi } from '@/api/admin'
import { useAuthStore } from '@/store/auth'
import { cn } from '@/lib/utils'

function StatCard({
  label,
  value,
  sub,
  icon: Icon,
  color = 'indigo',
}: {
  label: string
  value: string | number
  sub?: string
  icon: React.ElementType
  color?: 'indigo' | 'emerald' | 'amber' | 'rose'
}) {
  const colors = {
    indigo: 'bg-indigo-600/10 border-indigo-500/20 text-indigo-400',
    emerald: 'bg-emerald-600/10 border-emerald-500/20 text-emerald-400',
    amber: 'bg-amber-600/10 border-amber-500/20 text-amber-400',
    rose: 'bg-rose-600/10 border-rose-500/20 text-rose-400',
  }
  return (
    <div className="bg-[#161b27] border border-slate-800/60 rounded-xl p-5 hover:border-slate-700/60 transition-all">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-medium text-slate-500 mb-1">{label}</p>
          <p className="text-2xl font-bold text-white">{value}</p>
          {sub && <p className="text-xs text-slate-500 mt-1">{sub}</p>}
        </div>
        <div className={cn('p-2.5 rounded-lg border', colors[color])}>
          <Icon className="w-5 h-5" />
        </div>
      </div>
    </div>
  )
}

function HealthBadge({ status }: { status: string }) {
  const ok = status === 'ok' || status === 'healthy'
  return (
    <span className={cn(
      'flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full border',
      ok
        ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400'
        : 'bg-rose-500/10 border-rose-500/20 text-rose-400',
    )}>
      {ok ? <CheckCircle className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
      {ok ? 'Servicio Operativo' : 'Servicio Degradado'}
    </span>
  )
}

const sampleActivity = [
  { action: 'auth_login_success', user: 'user@acme.com', time: '14:02', success: true },
  { action: 'auth_login_failed', user: 'hacker@test.com', time: '13:58', success: false },
  { action: 'auth_2fa_enabled', user: 'alice@acme.com', time: '13:41', success: true },
  { action: 'admin_role_created', user: 'admin@verter.io', time: '13:20', success: true },
  { action: 'auth_account_locked', user: 'bob@corp.com', time: '12:55', success: false },
  { action: 'auth_password_reset', user: 'charlie@co.io', time: '12:40', success: true },
]

export function DashboardPage() {
  const user = useAuthStore((s) => s.user)

  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: healthApi.check,
    retry: 1,
  })

  return (
    <div className="space-y-6 fade-in">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-white">Dashboard</h2>
          <p className="text-sm text-slate-500 mt-0.5">
            Bienvenido, <span className="text-slate-300">{user?.username ?? 'Admin'}</span>
          </p>
        </div>
        {health && <HealthBadge status={health.status} />}
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
        <StatCard label="Estado del Servicio" value={health?.status === 'ok' ? 'Online' : 'Verificando...'} sub="HTTP REST + gRPC" icon={Server} color="emerald" />
        <StatCard label="API Version" value="v1.7" sub="Go 1.23 · PostgreSQL 16" icon={TrendingUp} color="indigo" />
        <StatCard label="Roles Activos" value="—" sub="Cargando..." icon={Shield} color="amber" />
        <StatCard label="Módulos" value="7" sub="RBAC, M2M, Compliance..." icon={Users} color="indigo" />
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent Audit Activity */}
        <div className="lg:col-span-2 bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden">
          <div className="flex items-center justify-between px-5 py-4 border-b border-slate-800/60">
            <div className="flex items-center gap-2">
              <Activity className="w-4 h-4 text-indigo-400" />
              <h3 className="text-sm font-semibold text-slate-200">Actividad Reciente</h3>
            </div>
            <span className="text-xs text-slate-600">Últimas 24h</span>
          </div>
          <div className="divide-y divide-slate-800/40">
            {sampleActivity.map((item, i) => (
              <div key={i} className="flex items-center gap-3 px-5 py-3 hover:bg-slate-800/20 transition-colors">
                <div className={cn(
                  'w-1.5 h-1.5 rounded-full shrink-0',
                  item.success ? 'bg-emerald-400' : 'bg-rose-400',
                )} />
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-mono text-slate-300 truncate">{item.action}</p>
                  <p className="text-xs text-slate-600 truncate">{item.user}</p>
                </div>
                <div className="flex items-center gap-1.5 shrink-0">
                  {item.success
                    ? <CheckCircle className="w-3.5 h-3.5 text-emerald-500" />
                    : <XCircle className="w-3.5 h-3.5 text-rose-500" />
                  }
                  <span className="text-xs text-slate-600 font-mono">{item.time}</span>
                </div>
              </div>
            ))}
          </div>
          <div className="px-5 py-3 border-t border-slate-800/60">
            <a href="/audit-logs" className="text-xs text-indigo-400 hover:text-indigo-300 transition-colors">
              Ver todos los logs →
            </a>
          </div>
        </div>

        {/* Quick Actions Panel */}
        <div className="bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden">
          <div className="flex items-center gap-2 px-5 py-4 border-b border-slate-800/60">
            <AlertTriangle className="w-4 h-4 text-amber-400" />
            <h3 className="text-sm font-semibold text-slate-200">Módulos Admin</h3>
          </div>
          <div className="p-3 space-y-2">
            {[
              { label: 'Gestión de Usuarios', href: '/users', desc: 'Ver y administrar usuarios' },
              { label: 'Roles & Permisos', href: '/rbac', desc: 'Configurar RBAC' },
              { label: 'Generar Reporte GDPR', href: '/compliance', desc: 'Exportar datos de usuario' },
              { label: 'Reporte SOC2 / HIPAA', href: '/compliance', desc: 'Auditoría de seguridad' },
              { label: 'Emitir Certificado M2M', href: '/certificates', desc: 'Autenticación inter-servicios' },
              { label: 'Audit Logs', href: '/audit-logs', desc: 'Búsqueda avanzada' },
            ].map((item) => (
              <a
                key={item.label}
                href={item.href}
                className="flex flex-col px-3 py-2.5 rounded-lg hover:bg-slate-800/40 hover:border-slate-700/50 border border-transparent transition-all group"
              >
                <span className="text-xs font-medium text-slate-300 group-hover:text-indigo-400 transition-colors">{item.label}</span>
                <span className="text-xs text-slate-600 mt-0.5">{item.desc}</span>
              </a>
            ))}
          </div>
        </div>
      </div>

      {/* Service Info */}
      <div className="bg-[#161b27] border border-slate-800/60 rounded-xl p-5">
        <h3 className="text-sm font-semibold text-slate-200 mb-3">Stack del Microservicio</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { label: 'Runtime', value: 'Go 1.23' },
            { label: 'Database', value: 'PostgreSQL 16' },
            { label: 'Cache', value: 'Redis 7' },
            { label: 'Tokens', value: 'RSA-256 JWT' },
            { label: 'Passwords', value: 'Argon2id' },
            { label: 'gRPC', value: 'mTLS + CSR' },
            { label: 'Auditoría', value: 'Compliance Ready' },
            { label: 'Multitenancy', value: 'User Pools' },
          ].map((item) => (
            <div key={item.label}>
              <p className="text-xs text-slate-600">{item.label}</p>
              <p className="text-xs font-medium text-slate-300 mt-0.5">{item.value}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
