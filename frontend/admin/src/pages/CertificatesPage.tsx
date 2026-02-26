import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Key, Download, Loader2, AlertCircle, Copy, CheckCircle } from 'lucide-react'
import { m2mApi } from '@/api/admin'
import { downloadText } from '@/lib/utils'
import toast from 'react-hot-toast'
import type { M2MCertificate } from '@/types/api'

type Mode = 'issue' | 'sign'

export function CertificatesPage() {
  const [mode, setMode] = useState<Mode>('issue')
  const [clientId, setClientId] = useState('')
  const [organization, setOrganization] = useState('')
  const [validityDays, setValidityDays] = useState('365')
  const [csrPem, setCsrPem] = useState('')
  const [result, setResult] = useState<M2MCertificate | null>(null)
  const [copiedField, setCopiedField] = useState<string | null>(null)

  const issueMut = useMutation({
    mutationFn: () => m2mApi.issueCertificate({
      client_id: clientId,
      organization: organization || undefined,
      validity_days: parseInt(validityDays),
    }),
    onSuccess: (data) => { setResult(data); toast.success('Certificado emitido') },
    onError: () => toast.error('Error al emitir certificado'),
  })

  const signMut = useMutation({
    mutationFn: () => m2mApi.signCSR({ csr_pem: csrPem, validity_days: parseInt(validityDays) }),
    onSuccess: (data) => { setResult(data); toast.success('CSR firmado correctamente') },
    onError: () => toast.error('Error al firmar el CSR'),
  })

  const isPending = issueMut.isPending || signMut.isPending

  const handleSubmit = () => {
    setResult(null)
    if (mode === 'issue') issueMut.mutate()
    else signMut.mutate()
  }

  const copyToClipboard = (text: string, field: string) => {
    navigator.clipboard.writeText(text)
    setCopiedField(field)
    setTimeout(() => setCopiedField(null), 2000)
    toast.success('Copiado al portapapeles')
  }

  return (
    <div className="space-y-6 fade-in">
      <div>
        <h2 className="text-xl font-semibold text-white">M2M / Certificados</h2>
        <p className="text-sm text-slate-500 mt-0.5">Autenticación Machine-to-Machine con mTLS</p>
      </div>

      {/* Mode selector */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {[
          { id: 'issue' as Mode, label: 'Emitir Certificado', desc: 'Genera un certificado completo (clave + cert) para un cliente' },
          { id: 'sign' as Mode, label: 'Firmar CSR', desc: 'Firma un CSR existente (Zero-Knowledge — la clave privada nunca sale del cliente)' },
        ].map(({ id, label, desc }) => (
          <button
            key={id}
            onClick={() => { setMode(id); setResult(null) }}
            className={`flex flex-col gap-1.5 p-4 rounded-xl border text-left transition-all ${
              mode === id
                ? 'bg-indigo-600/10 border-indigo-500/30'
                : 'bg-[#161b27] border-slate-800/60 hover:border-slate-700/60'
            }`}
          >
            <p className="text-sm font-semibold text-white">{label}</p>
            <p className="text-xs text-slate-500">{desc}</p>
          </button>
        ))}
      </div>

      {/* Form */}
      <div className="bg-[#161b27] border border-slate-800/60 rounded-xl p-5">
        <div className="flex items-center gap-2 mb-5">
          <Key className="w-4 h-4 text-indigo-400" />
          <h3 className="text-sm font-semibold text-slate-200">
            {mode === 'issue' ? 'Emitir Nuevo Certificado' : 'Firmar CSR Existente'}
          </h3>
        </div>

        <div className="space-y-4 max-w-md">
          {mode === 'issue' ? (
            <>
              <div>
                <label className="block text-xs text-slate-400 mb-1.5">Client ID *</label>
                <input
                  value={clientId}
                  onChange={(e) => setClientId(e.target.value)}
                  placeholder="ej: api-gateway, billing-service..."
                  className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-400 mb-1.5">Organización</label>
                <input
                  value={organization}
                  onChange={(e) => setOrganization(e.target.value)}
                  placeholder="ej: Vertercloud Inc."
                  className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
            </>
          ) : (
            <div>
              <label className="block text-xs text-slate-400 mb-1.5">CSR (PEM format) *</label>
              <textarea
                value={csrPem}
                onChange={(e) => setCsrPem(e.target.value)}
                placeholder="-----BEGIN CERTIFICATE REQUEST-----&#10;...&#10;-----END CERTIFICATE REQUEST-----"
                rows={8}
                className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-xs text-slate-300 placeholder-slate-600 font-mono focus:outline-none focus:ring-1 focus:ring-indigo-500 resize-none"
              />
            </div>
          )}

          <div>
            <label className="block text-xs text-slate-400 mb-1.5">Validez (días)</label>
            <input
              type="number"
              value={validityDays}
              onChange={(e) => setValidityDays(e.target.value)}
              className="w-full bg-[#1e2433] border border-slate-700/60 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>

          <button
            onClick={handleSubmit}
            disabled={isPending || (mode === 'issue' ? !clientId : !csrPem)}
            className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-60 text-white text-sm font-medium rounded-lg transition-all"
          >
            {isPending ? (
              <><Loader2 className="w-4 h-4 animate-spin" />Procesando...</>
            ) : (
              <><Key className="w-4 h-4" />{mode === 'issue' ? 'Emitir Certificado' : 'Firmar CSR'}</>
            )}
          </button>
        </div>
      </div>

      {/* Result */}
      {result && (
        <div className="bg-[#161b27] border border-slate-800/60 rounded-xl overflow-hidden fade-in">
          <div className="flex items-center gap-2 px-5 py-4 border-b border-slate-800/60">
            <CheckCircle className="w-4 h-4 text-emerald-400" />
            <span className="text-sm font-semibold text-slate-200">Certificado Listo</span>
            <span className="ml-auto text-xs text-slate-600">
              Expira: {result.expires_at ? new Date(result.expires_at).toLocaleDateString() : '—'}
            </span>
          </div>
          <div className="p-5 space-y-4">
            {/* Certificate PEM */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-xs font-medium text-slate-400">Certificado (PEM)</label>
                <div className="flex gap-2">
                  <button
                    onClick={() => copyToClipboard(result.certificate_pem, 'cert')}
                    className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-300 transition-colors"
                  >
                    {copiedField === 'cert' ? <CheckCircle className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    Copiar
                  </button>
                  <button
                    onClick={() => downloadText(result.certificate_pem, `${result.client_id || 'certificate'}.crt.pem`)}
                    className="flex items-center gap-1 text-xs text-indigo-400 hover:text-indigo-300 transition-colors"
                  >
                    <Download className="w-3 h-3" /> Descargar
                  </button>
                </div>
              </div>
              <pre className="text-xs text-slate-400 bg-[#0f1117] rounded-lg p-3 overflow-auto max-h-36 font-mono">
                {result.certificate_pem}
              </pre>
            </div>

            {/* Private Key PEM (only when issued, not when signed) */}
            {result.private_key_pem && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-1.5">
                    <AlertCircle className="w-3.5 h-3.5 text-amber-400" />
                    <label className="text-xs font-medium text-amber-400">Clave Privada (PEM) — Guárdala de inmediato</label>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => copyToClipboard(result.private_key_pem!, 'key')}
                      className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-300 transition-colors"
                    >
                      {copiedField === 'key' ? <CheckCircle className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                      Copiar
                    </button>
                    <button
                      onClick={() => downloadText(result.private_key_pem!, `${result.client_id || 'private'}.key.pem`)}
                      className="flex items-center gap-1 text-xs text-amber-400 hover:text-amber-300 transition-colors"
                    >
                      <Download className="w-3 h-3" /> Descargar
                    </button>
                  </div>
                </div>
                <pre className="text-xs text-slate-400 bg-[#0f1117] rounded-lg p-3 overflow-auto max-h-36 font-mono border border-amber-500/20">
                  {result.private_key_pem}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Info */}
      <div className="bg-[#161b27] border border-slate-800/60 rounded-xl p-5">
        <h3 className="text-sm font-semibold text-slate-200 mb-3">Flujo Zero-Knowledge (CSR)</h3>
        <ol className="space-y-2 text-xs text-slate-500">
          <li className="flex gap-2"><span className="text-indigo-400 font-bold shrink-0">1.</span>El servicio cliente genera su propio par de claves RSA/ECDSA localmente.</li>
          <li className="flex gap-2"><span className="text-indigo-400 font-bold shrink-0">2.</span>Crea un CSR (Certificate Signing Request) con sus datos y lo envía al Admin Console.</li>
          <li className="flex gap-2"><span className="text-indigo-400 font-bold shrink-0">3.</span>El Auth Service firma el CSR con su CA privada y retorna solo el certificado firmado.</li>
          <li className="flex gap-2"><span className="text-indigo-400 font-bold shrink-0">4.</span>La clave privada nunca sale del servidor del cliente. Máxima seguridad.</li>
        </ol>
      </div>
    </div>
  )
}
