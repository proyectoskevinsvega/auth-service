import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  Users,
  Shield,
  FileText,
  Key,
  ScrollText,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Zap,
  Menu,
} from 'lucide-react'
import { useAuthStore } from '@/store/auth'
import { authApi } from '@/api/auth'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/users', label: 'Usuarios', icon: Users },
  { to: '/rbac', label: 'Roles & Permisos', icon: Shield },
  { to: '/compliance', label: 'Compliance', icon: FileText },
  { to: '/certificates', label: 'M2M / Certs', icon: Key },
  { to: '/audit-logs', label: 'Audit Logs', icon: ScrollText },
]

export function AppLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = async () => {
    try { await authApi.logout() } catch { /* noop */ }
    logout()
    navigate('/login')
  }

  return (
    <div className="flex h-screen bg-[#0f1117] text-slate-100 overflow-hidden">
      {/* Sidebar */}
      <aside
        className={cn(
          'flex flex-col border-r border-slate-800/60 bg-[#161b27] transition-all duration-300 ease-in-out',
          collapsed ? 'w-16' : 'w-60',
        )}
      >
        {/* Logo */}
        <div className={cn('flex items-center gap-3 px-4 h-16 border-b border-slate-800/60', collapsed && 'justify-center px-2')}>
          <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-indigo-600 shrink-0">
            <Zap className="w-4 h-4 text-white" />
          </div>
          {!collapsed && (
            <div className="slide-in">
              <p className="text-sm font-semibold text-white leading-tight">Auth Admin</p>
              <p className="text-xs text-slate-500">Vertercloud</p>
            </div>
          )}
        </div>

        {/* Navigation */}
        <nav className="flex-1 flex flex-col gap-1 p-2 overflow-y-auto">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150',
                  isActive
                    ? 'bg-indigo-600/20 text-indigo-400 border border-indigo-500/20'
                    : 'text-slate-400 hover:bg-slate-800/60 hover:text-slate-200',
                  collapsed && 'justify-center px-2',
                )
              }
            >
              <Icon className="w-4 h-4 shrink-0" />
              {!collapsed && <span className="slide-in truncate">{label}</span>}
            </NavLink>
          ))}
        </nav>

        {/* Collapse toggle */}
        <div className="p-2 border-t border-slate-800/60">
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="flex items-center gap-3 w-full px-3 py-2 rounded-lg text-sm text-slate-500 hover:bg-slate-800/60 hover:text-slate-300 transition-all"
          >
            {collapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
            {!collapsed && <span className="text-xs">Colapsar</span>}
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex flex-col flex-1 overflow-hidden">
        {/* Header */}
        <header className="flex items-center justify-between h-16 px-6 border-b border-slate-800/60 bg-[#161b27]/50 backdrop-blur-sm shrink-0">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setCollapsed(!collapsed)}
              className="p-1.5 rounded-md text-slate-500 hover:text-slate-300 hover:bg-slate-800/60 transition-all lg:hidden"
            >
              <Menu className="w-4 h-4" />
            </button>
            <div>
              <h1 className="text-sm font-medium text-slate-200">
                Admin Console
              </h1>
              <p className="text-xs text-slate-500">Vertercloud Auth Service</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="text-right hidden sm:block">
              <p className="text-xs font-medium text-slate-300">{user?.username ?? 'Admin'}</p>
              <p className="text-xs text-slate-500">{user?.email}</p>
            </div>
            <div className="w-8 h-8 rounded-full bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center text-indigo-400 text-sm font-medium">
              {(user?.username?.[0] ?? 'A').toUpperCase()}
            </div>
            <button
              onClick={handleLogout}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-all"
            >
              <LogOut className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Salir</span>
            </button>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-y-auto p-6 fade-in">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
