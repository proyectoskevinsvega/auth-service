import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { FileText, Download, Loader2, AlertCircle, CheckCircle, User, Shield, Activity } from 'lucide-react'
import { complianceApi } from '@/api/admin'
import { downloadJson, formatDate } from '@/lib/utils'
import toast from 'react-hot-toast'
import type { GDPRDataExport, HIPAAReport, SOC2AuditReport } from '@/types/api'

type ReportType = 'gdpr' | 'soc2' | 'hipaa'

function InputField({ label, ...props }: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  return (
    <div>
      <label className="block text-xs text-slate-400 mb-1.5">{label}</label>
      <input
        {...props}
        className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
      />
    </div>
  )
}

export function CompliancePage() {
  const [activeReport, setActiveReport] = useState<ReportType>('gdpr')
  const [gdprUserId, setGdprUserId] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [result, setResult] = useState<GDPRDataExport | SOC2AuditReport | HIPAAReport | null>(null)

  const gdprMut = useMutation({
    mutationFn: () => complianceApi.generateGDPR(gdprUserId),
    onSuccess: (data) => { setResult(data); toast.success('Reporte GDPR generado') },
    onError: () => toast.error('Error al generar reporte GDPR'),
  })

  const soc2Mut = useMutation({
    mutationFn: () => complianceApi.generateSOC2(startDate, endDate),
    onSuccess: (data) => { setResult(data); toast.success('Reporte SOC2 generado') },
    onError: () => toast.error('Error al generar reporte SOC2'),
  })

  const hipaaMut = useMutation({
    mutationFn: () => complianceApi.generateHIPAA(startDate, endDate),
    onSuccess: (data) => { setResult(data); toast.success('Reporte HIPAA generado') },
    onError: () => toast.error('Error al generar reporte HIPAA'),
  })

  const isPending = gdprMut.isPending || soc2Mut.isPending || hipaaMut.isPending

  const handleGenerate = () => {
    setResult(null)
    if (activeReport === 'gdpr') gdprMut.mutate()
    if (activeReport === 'soc2') soc2Mut.mutate()
    if (activeReport === 'hipaa') hipaaMut.mutate()
  }

  const handleDownload = () => {
    if (!result) return
    downloadJson(result, `report-${activeReport}-${Date.now()}.json`)
  }

  const reports = [
    {
      id: 'gdpr' as ReportType,
      label: 'GDPR',
      desc: 'Portabilidad de Datos',
      icon: User,
      color: 'text-blue-400',
    },
    {
      id: 'soc2' as ReportType,
      label: 'SOC2',
      desc: 'Auditoría de Seguridad',
      icon: Shield,
      color: 'text-amber-400',
    },
    {
      id: 'hipaa' as ReportType,
      label: 'HIPAA',
      desc: 'Integridad y Acceso',
      icon: Activity,
      color: 'text-emerald-400',
    },
  ]

  return (
    <div className="space-y-6 fade-in">
      <div>
        <h2 className="text-xl font-semibold text-white">Compliance & Auditoría</h2>
        <p className="text-sm text-slate-500 mt-0.5">Generación de reportes de cumplimiento normativo</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Report selector */}
        {reports.map(({ id, label, desc, icon: Icon, color }) => (
          <button
            key={id}
            onClick={() => { setActiveReport(id); setResult(null) }}
            className={`flex flex-col gap-2 p-4 rounded-xl border text-left transition-all ${
              activeReport === id
                ? 'bg-indigo-600/10 border-indigo-500/30'
                : 'bg-[#161b27] border-slate-800/60 hover:border-slate-700/60'
            }`}
          >
            <div className="flex items-center gap-3">
              <div className={`p-2 rounded-lg bg-slate-800/60`}>
                <Icon className={`w-5 h-5 ${color}`} />
              </div>
              <div>
                <p className="text-sm font-semibold text-white">{label}</p>
                <p className="text-xs text-slate-500">{desc}</p>
              </div>
            </div>
          </button>
        ))}
      </div>

      {/* Form panel */}
      <div className="bg-[#161b27] border border-slate-800/60 rounded-xl p-5">
        <div className="flex items-center gap-2 mb-5">
          <FileText className="w-4 h-4 text-indigo-400" />
          <h3 className="text-sm font-semibold text-slate-200">
            Parámetros — Reporte {activeReport.toUpperCase()}
          </h3>
        </div>

        <div className="space-y-4 max-w-md">
          {activeReport === 'gdpr' ? (
            <InputField
              label="ID del Usuario *"
              placeholder="uuid del usuario..."
              value={gdprUserId}
              onChange={(e) => setGdprUserId(e.target.value)}
            />
          ) : (
            <>
              <InputField
                label="Fecha Inicio *"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
              />
              <InputField
                label="Fecha Fin *"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
              />
            </>
          )}

          <button
            onClick={handleGenerate}
            disabled={isPending || (activeReport === 'gdpr' ? !gdprUserId : !startDate || !endDate)}
            className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60 text-white text-sm font-medium rounded-lg transition-all"
          >
            {isPending ? (
              <><Loader2 className="w-4 h-4 animate-spin" />Generando...</>
            ) : (
              <><FileText className="w-4 h-4" />Generar Reporte</>
            )}
          </button>
        </div>
      </div>

      {/* Result panel */}
      {result && (
        <div className="bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden fade-in">
          <div className="flex items-center justify-between px-5 py-4 border-b border-slate-800/60">
            <div className="flex items-center gap-2">
              <CheckCircle className="w-4 h-4 text-emerald-400" />
              <span className="text-sm font-semibold text-slate-200">
                Reporte {activeReport.toUpperCase()} Generado
              </span>
              {'generated_at' in result && result.generated_at && (
                <span className="text-xs text-slate-600">{formatDate(result.generated_at)}</span>
              )}
              {'exported_at' in result && result.exported_at && (
                <span className="text-xs text-slate-600">{formatDate(result.exported_at)}</span>
              )}
            </div>
            <button
              onClick={handleDownload}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-emerald-600/10 hover:bg-emerald-600/20 text-emerald-400 border border-emerald-500/20 rounded-lg transition-all"
            >
              <Download className="w-3.5 h-3.5" />
              Descargar JSON
            </button>
          </div>
          <div className="p-5">
            <pre className="text-xs text-slate-400 bg-[#0f1117] rounded-lg p-4 overflow-auto max-h-96 font-mono leading-relaxed">
              {JSON.stringify(result, null, 2)}
            </pre>
          </div>
        </div>
      )}

      {/* Compliance info */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {[
          { label: 'GDPR', text: 'Reglamento General de Protección de Datos. Derecho al acceso y la portabilidad de datos personales.' },
          { label: 'SOC2', text: 'System and Organization Controls. Auditoría de controles de seguridad y disponibilidad del servicio.' },
          { label: 'HIPAA', text: 'Health Insurance Portability. Monitoreo de integridad de datos y control de acceso a información sensible.' },
        ].map(({ label, text }) => (
          <div key={label} className="bg-[#161b27] border border-slate-800/60 rounded-xl p-4">
            <div className="flex items-center gap-2 mb-2">
              <AlertCircle className="w-3.5 h-3.5 text-slate-500" />
              <span className="text-xs font-semibold text-slate-400">{label}</span>
            </div>
            <p className="text-xs text-slate-600 leading-relaxed">{text}</p>
          </div>
        ))}
      </div>
    </div>
  )
}
