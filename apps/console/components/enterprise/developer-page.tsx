"use client";

import { Button, Card } from "@heroui/react";
import { Activity, BookOpenText, CheckCircle2, Database, ExternalLink, FolderGit2, KeyRound, RefreshCcw, ServerCog, ShieldCheck, TerminalSquare, XCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { MetricGrid, PageHeader, StateBadge, type APIState } from "@/components/foundation/resource-ui";
import { getLiteLLMIntegrationStatus, verifyLiteLLMIntegration } from "@/lib/control-plane";
import type { IntegrationProbe, LiteLLMIntegrationStatus } from "@/types/integration";

const emptyStatus: LiteLLMIntegrationStatus = { configured: false, baseUrl: "", masterKeyConfigured: false, overall: "not_configured", liveness: null, readiness: null, checkedAt: null };

export function DeveloperPage() {
  const [status, setStatus] = useState<LiteLLMIntegrationStatus>(emptyStatus);
  const [apiState, setAPIState] = useState<APIState>("connecting");
  const [notice, setNotice] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    getLiteLLMIntegrationStatus(controller.signal).then((response) => {
      setStatus(response.data);
      setAPIState("connected");
    }).catch((error: unknown) => {
      if (!(error instanceof DOMException && error.name === "AbortError")) setAPIState("fallback");
    });
    return () => controller.abort();
  }, []);

  async function verify() {
    setAPIState("saving");
    setNotice("");
    try {
      const response = await verifyLiteLLMIntegration();
      setStatus(response.data);
      setNotice(response.data.overall === "ready" ? "LiteLLM liveness and readiness probes passed." : "LiteLLM responded, but the gateway is not fully ready. Review probe evidence below.");
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "LiteLLM verification could not run.");
      setAPIState("fallback");
    }
  }

  return <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
    <PageHeader eyebrow="Developer platform" title="Integration diagnostics" description="Verify the LiteLLM data-plane boundary and keep deployment source, runtime secrets, and persistent data in their correct locations." icon={BookOpenText} apiState={apiState} action={<Button onPress={verify} isDisabled={!status.configured} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><RefreshCcw size={14}/>Verify LiteLLM</Button>}/>
    {notice && <div className="rounded-xl border border-amber-400/15 bg-amber-400/[0.04] px-4 py-3 text-xs text-amber-200">{notice}</div>}
    <MetricGrid items={[
      { label: "Integration", value: status.configured ? "Configured" : "Pending", hint: status.baseUrl || "LITELLM_BASE_URL is not set", icon: ServerCog },
      { label: "Master key", value: status.masterKeyConfigured ? "Server-side" : "Missing", hint: "Value is never returned to the browser", icon: KeyRound },
      { label: "Liveness", value: probeLabel(status.liveness), hint: probeHint(status.liveness), icon: Activity },
      { label: "Readiness", value: probeLabel(status.readiness), hint: probeHint(status.readiness), icon: Database },
    ]}/>

    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_440px]">
      <Card className="glass-panel px-0 py-0"><Card.Header className="flex-row items-start justify-between border-b border-white/[0.055] p-5"><div><Card.Title className="text-sm text-slate-100">LiteLLM data plane</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">AetherGate probes public health endpoints only; it never reads or mutates LiteLLM internal tables.</Card.Description></div><StateBadge value={status.overall}/></Card.Header><Card.Content className="space-y-4 p-5"><div className="rounded-xl border border-white/[0.055] bg-black/20 p-4"><p className="text-[9px] uppercase text-slate-600">Internal base URL</p><p className="mt-2 break-all font-mono text-xs text-blue-200/70">{status.baseUrl || "Set LITELLM_BASE_URL in the Go API server environment"}</p></div><div className="grid gap-3 md:grid-cols-2"><ProbeCard title="Liveness" probe={status.liveness}/><ProbeCard title="Readiness" probe={status.readiness}/></div><div className="rounded-xl border border-emerald-400/12 bg-emerald-400/[0.03] p-4"><p className="flex items-center gap-2 text-xs text-emerald-200"><ShieldCheck size={13}/>Credential-safe diagnostic</p><p className="mt-2 text-[10px] leading-5 text-slate-500">The Master Key is attached only by the Go server. Redirects are rejected so authorization cannot be forwarded to another host; response bodies are bounded and never relayed to the Console.</p></div></Card.Content></Card>

      <Card className="glass-panel px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-5"><Card.Title className="text-sm text-slate-100">Stack placement</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Keep reviewed deployment source and the server runtime copy separate.</Card.Description></Card.Header><Card.Content className="space-y-4 p-5"><Location icon={FolderGit2} label="Repository source" value="deploy/compose/core/" detail="Copy only reviewed Compose, config templates, scripts, and documentation. Do not add an extra nested stack directory."/><Location icon={TerminalSquare} label="Server runtime" value="/opt/aethergate" detail="Recommended canonical location for Compose, server-only .env, generated backend environment, scripts, and mounts."/><Location icon={ExternalLink} label="Existing runtime alias" value="/opt/aethergate-litellm-stack" detail="May remain during migration; converge to one documented runtime directory after backup and verification."/><div className="rounded-xl border border-rose-400/12 bg-rose-400/[0.03] p-3 text-[10px] leading-5 text-rose-100/65">Never copy `.env`, real provider keys, LiteLLM Master/Salt keys, PostgreSQL data, Redis data, logs, or backups into Git.</div></Card.Content></Card>
    </div>

    <Card className="glass-panel px-0 py-0"><Card.Header className="p-5"><Card.Title className="text-sm text-slate-100">Integration gate</Card.Title></Card.Header><Card.Content className="grid gap-3 border-t border-white/[0.055] p-5 md:grid-cols-2 xl:grid-cols-4">{[
      ["1", "Import safe stack source", "Follow docs/deployment/stack-import.zh-CN.md and review every copied file."],
      ["2", "Pin and configure", "Set LITELLM_BASE_URL and server-only Master Key; keep both out of NEXT_PUBLIC variables."],
      ["3", "Verify service health", "Run liveness/readiness from the same network as AetherGate and inspect Compose health/logs."],
      ["4", "Exercise real traffic", "Test streaming, cancellation, virtual keys, routing, usage attribution, and provider failures before promotion."],
    ].map(([step,title,detail]) => <div key={step} className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-4"><span className="flex size-7 items-center justify-center rounded-lg bg-blue-400/10 font-mono text-[10px] text-blue-300">{step}</span><p className="mt-3 text-xs font-medium text-slate-200">{title}</p><p className="mt-2 text-[10px] leading-5 text-slate-600">{detail}</p></div>)}</Card.Content></Card>
  </div>;
}

function ProbeCard({ title, probe }: { title: string; probe: IntegrationProbe | null }) {
  const Icon = probe?.healthy ? CheckCircle2 : XCircle;
  return <div className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-4"><div className="flex items-center justify-between"><p className="text-xs text-slate-300">{title}</p><Icon size={14} className={probe?.healthy ? "text-emerald-300" : "text-slate-700"}/></div><p className="mt-3 font-mono text-[10px] text-slate-500">{probe?.path ?? "Not probed"}</p><p className="mt-1 text-[10px] text-slate-600">{probe ? `HTTP ${probe.statusCode || "—"} · ${probe.latencyMs} ms${probe.errorCode ? ` · ${probe.errorCode}` : ""}` : "Run verification after configuration."}</p></div>;
}

function Location({ icon: Icon, label, value, detail }: { icon: typeof FolderGit2; label: string; value: string; detail: string }) {
  return <div className="flex gap-3 rounded-xl border border-white/[0.055] bg-white/[0.018] p-4"><span className="mt-0.5 text-blue-300"><Icon size={15}/></span><div className="min-w-0"><p className="text-[9px] uppercase text-slate-600">{label}</p><p className="mt-1 break-all font-mono text-xs text-slate-200">{value}</p><p className="mt-2 text-[10px] leading-5 text-slate-600">{detail}</p></div></div>;
}

function probeLabel(probe: IntegrationProbe | null) { return probe ? (probe.healthy ? "Healthy" : "Failed") : "Not checked"; }
function probeHint(probe: IntegrationProbe | null) { return probe ? `${probe.latencyMs} ms · HTTP ${probe.statusCode || "—"}` : "Live verification required"; }
