"use client";

import { Button, Card, Table } from "@heroui/react";
import { CheckCircle2, ChevronRight, Copy, Gauge, KeyRound, Plus, Search, ShieldCheck, ShieldX, X } from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { createAPIKey as createAPIKeyRequest, listAPIKeys as listAPIKeysRequest, revokeAPIKey } from "@/lib/control-plane";
import { apiKeys as seedAPIKeys } from "@/lib/enterprise-data";
import type { APIKeyRecord, APIKeyStatus } from "@/types/enterprise";

type StatusFilter = "all" | APIKeyStatus;

const statusStyles: Record<APIKeyStatus, string> = {
  active: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20",
  expired: "bg-amber-400/10 text-amber-300 ring-amber-400/20",
  revoked: "bg-rose-400/10 text-rose-300 ring-rose-400/20",
};

function KeyStatusBadge({ status }: { status: APIKeyStatus }) {
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] font-medium capitalize ring-1 ring-inset ${statusStyles[status]}`}>{status}</span>;
}

function CreateKeyDialog({ onClose, onCreate }: { onClose: () => void; onCreate: (record: APIKeyRecord, secret: string) => void }) {
  const [name, setName] = useState("");
  const [project, setProject] = useState("Engineering Copilot");
  const [model, setModel] = useState("claude-sonnet-4");
  const [rpm, setRpm] = useState("300");
  const [expires, setExpires] = useState("180");

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!name.trim()) return;
    const random = crypto.randomUUID().replaceAll("-", "");
    const secret = `ag_live_${random}`;
    const expiresAt = expires === "never" ? null : new Date(Date.now() + Number(expires) * 86_400_000).toISOString().slice(0, 10);
    onCreate({
      id: `key_${random.slice(0, 12).toUpperCase()}`,
      name: name.trim(),
      prefix: `${secret.slice(0, 12)}`,
      project,
      status: "active",
      models: [model],
      rpm: Number(rpm),
      tpm: Number(rpm) * 2_000,
      spendUsd: 0,
      createdBy: "holden@topoai.dev",
      createdAt: new Date().toISOString().slice(0, 10),
      lastUsedAt: null,
      expiresAt,
    }, secret);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#02040a]/80 p-4 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label="Create API key">
      <form onSubmit={submit} className="w-full max-w-lg rounded-2xl border border-white/10 bg-[#0a0f18] shadow-2xl shadow-black/60">
        <div className="flex items-start justify-between border-b border-white/[0.06] p-5"><div><h2 className="text-base font-semibold text-white">Create API key</h2><p className="mt-1 text-xs text-slate-500">Scope a development key to one project, model policy, and limit.</p></div><button type="button" onClick={onClose} className="rounded-lg p-2 text-slate-500 hover:bg-white/5 hover:text-white"><X size={15} /></button></div>
        <div className="space-y-4 p-5">
          <label className="block"><span className="mb-1.5 block text-[11px] font-medium text-slate-400">Key name</span><input autoFocus value={name} onChange={(event) => setName(event.target.value)} placeholder="Production application" className="h-10 w-full rounded-xl border border-white/8 bg-black/20 px-3 text-xs text-slate-200 outline-none focus:border-blue-400/40" /></label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label><span className="mb-1.5 block text-[11px] font-medium text-slate-400">Project</span><select value={project} onChange={(event) => setProject(event.target.value)} className="h-10 w-full rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option>Engineering Copilot</option><option>Customer Support</option><option>Finance Analyst</option><option>Contract Intelligence</option></select></label>
            <label><span className="mb-1.5 block text-[11px] font-medium text-slate-400">Allowed model</span><select value={model} onChange={(event) => setModel(event.target.value)} className="h-10 w-full rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option>claude-sonnet-4</option><option>gpt-5-mini</option><option>gemini-2.5-pro</option><option>deepseek-v3</option></select></label>
            <label><span className="mb-1.5 block text-[11px] font-medium text-slate-400">RPM limit</span><select value={rpm} onChange={(event) => setRpm(event.target.value)} className="h-10 w-full rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option value="60">60 RPM</option><option value="120">120 RPM</option><option value="300">300 RPM</option><option value="600">600 RPM</option><option value="900">900 RPM</option></select></label>
            <label><span className="mb-1.5 block text-[11px] font-medium text-slate-400">Expiration</span><select value={expires} onChange={(event) => setExpires(event.target.value)} className="h-10 w-full rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option value="30">30 days</option><option value="90">90 days</option><option value="180">180 days</option><option value="365">1 year</option><option value="never">No expiration</option></select></label>
          </div>
          <div className="rounded-xl border border-blue-400/12 bg-blue-400/[0.035] p-3 text-[11px] leading-5 text-blue-100/70">The secret is generated in this browser and shown once. Production issuance will store only an Argon2id hash and create an immutable audit event.</div>
        </div>
        <div className="flex justify-end gap-2 border-t border-white/[0.06] p-4"><Button type="button" onPress={onClose} variant="tertiary" className="h-9 border border-white/8 bg-white/[0.025] text-xs text-slate-300">Cancel</Button><Button type="submit" isDisabled={!name.trim()} className="h-9 gap-2 bg-blue-500 text-xs text-white"><KeyRound size={13} /> Create key</Button></div>
      </form>
    </div>
  );
}

function SecretDialog({ secret, onClose }: { secret: string; onClose: () => void }) {
  const [copied, setCopied] = useState(false);
  async function copy() {
    await navigator.clipboard.writeText(secret);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  }
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-[#02040a]/85 p-4 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label="API key secret">
      <div className="w-full max-w-lg rounded-2xl border border-emerald-400/15 bg-[#0a0f18] shadow-2xl shadow-black/60">
        <div className="p-6 text-center"><span className="mx-auto flex size-11 items-center justify-center rounded-full bg-emerald-400/10 text-emerald-300 ring-1 ring-inset ring-emerald-400/20"><ShieldCheck size={20} /></span><h2 className="mt-4 text-base font-semibold text-white">API key created</h2><p className="mt-2 text-xs leading-5 text-slate-500">Copy this secret now. For security it will not be shown again.</p><div className="mt-5 break-all rounded-xl border border-white/8 bg-black/30 p-4 text-left font-mono text-xs leading-5 text-slate-200">{secret}</div><div className="mt-4 flex gap-2"><Button onPress={copy} className="h-9 flex-1 gap-2 bg-blue-500 text-xs text-white">{copied ? <CheckCircle2 size={13} /> : <Copy size={13} />} {copied ? "Copied" : "Copy secret"}</Button><Button onPress={onClose} variant="tertiary" className="h-9 flex-1 border border-white/8 bg-white/[0.025] text-xs text-slate-300">I saved it</Button></div></div>
      </div>
    </div>
  );
}

export function APIKeysPage() {
  const [keys, setKeys] = useState(seedAPIKeys);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [selectedId, setSelectedId] = useState(seedAPIKeys[0]?.id ?? "");
  const [createOpen, setCreateOpen] = useState(false);
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const [apiState, setAPIState] = useState<"connecting" | "connected" | "fallback" | "saving">("connecting");
  void apiState;

  useEffect(() => {
    const controller = new AbortController();
    listAPIKeysRequest({ organizationId: "org_topoai" }, controller.signal)
      .then(({ data }) => {
        setKeys(data);
        setSelectedId((current) => data.some((item) => item.id === current) ? current : (data[0]?.id ?? ""));
        setAPIState("connected");
      })
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setAPIState("fallback");
      });
    return () => controller.abort();
  }, []);

  const filtered = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return keys.filter((key) => (status === "all" || key.status === status) && (!normalized || [key.name, key.prefix, key.project, key.createdBy, ...key.models].some((value) => value.toLowerCase().includes(normalized))));
  }, [keys, query, status]);
  const selected = keys.find((key) => key.id === selectedId) ?? null;
  const activeCount = keys.filter((key) => key.status === "active").length;
  const totalSpend = keys.reduce((sum, key) => sum + key.spendUsd, 0);

  function createKey(record: APIKeyRecord, secret: string) {
    setKeys((current) => [record, ...current]);
    setSelectedId(record.id);
    setCreateOpen(false);
    setAPIState("saving");
    void createAPIKeyRequest({
      organizationId: "org_topoai",
      name: record.name,
      project: record.project,
      models: record.models,
      rpm: record.rpm,
      tpm: record.tpm,
      createdBy: record.createdBy,
      expiresAt: record.expiresAt,
    }).then(({ data, secret: issuedSecret }) => {
      setKeys((current) => current.map((item) => item.id === record.id ? data : item));
      setSelectedId(data.id);
      setNewSecret(issuedSecret);
      setAPIState("connected");
    }).catch(() => {
      setKeys((current) => current.filter((item) => item.id !== record.id));
      setAPIState("fallback");
    });
    void secret;
  }

  function revokeSelected() {
    if (!selected || selected.status !== "active") return;
    setKeys((current) => current.map((key) => key.id === selected.id ? { ...key, status: "revoked" } : key));
    setAPIState("saving");
    void revokeAPIKey(selected.id).then(({ data }) => {
      setKeys((current) => current.map((key) => key.id === selected.id ? data : key));
      setAPIState("connected");
    }).catch(() => {
      setKeys((current) => current.map((key) => key.id === selected.id ? { ...key, status: "active" } : key));
      setAPIState("fallback");
    });
  }

  return (
    <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between"><div><div className="mb-2 flex items-center gap-2 text-[11px] font-medium tracking-[0.12em] text-blue-300/80 uppercase"><KeyRound size={13} /> Gateway credentials</div><h1 className="text-2xl font-semibold tracking-[-0.035em] text-white sm:text-3xl">API keys</h1><p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">Issue project-scoped gateway credentials with model access, rate limits, expiration, rotation, and usage evidence.</p></div><Button onPress={() => setCreateOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={14} /> Create API key</Button></section>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">{[
        { label: "Active keys", value: activeCount.toString(), hint: `${keys.length} total credentials`, icon: KeyRound },
        { label: "Monthly spend", value: `$${totalSpend.toLocaleString(undefined, { maximumFractionDigits: 0 })}`, hint: "Attributed by virtual key", icon: ShieldCheck },
        { label: "Highest RPM", value: Math.max(...keys.map((key) => key.rpm)).toLocaleString(), hint: "Per-key configured limit", icon: Gauge },
        { label: "Revoked / expired", value: (keys.length - activeCount).toString(), hint: "Retained for audit", icon: ShieldX },
      ].map((metric) => { const Icon = metric.icon; return <Card key={metric.label} className="glass-panel px-0 py-0"><Card.Content className="flex items-start justify-between p-4"><div><p className="text-[10px] font-medium tracking-wider text-slate-600 uppercase">{metric.label}</p><p className="mt-2 text-xl font-semibold text-white">{metric.value}</p><p className="mt-1 text-[10px] text-slate-600">{metric.hint}</p></div><span className="rounded-lg bg-blue-400/8 p-2 text-blue-300 ring-1 ring-inset ring-blue-400/12"><Icon size={15} /></span></Card.Content></Card>; })}</section>

      <Card className="glass-panel px-0 py-0"><Card.Content className="flex flex-col gap-2 p-3 lg:flex-row"><label className="relative flex-1"><Search className="absolute top-1/2 left-3 -translate-y-1/2 text-slate-600" size={14} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search key, project, model, or creator..." className="h-10 w-full rounded-xl border border-white/8 bg-black/20 pr-3 pl-9 text-xs text-slate-200 outline-none focus:border-blue-400/40" /></label><select value={status} onChange={(event) => setStatus(event.target.value as StatusFilter)} className="h-10 rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option value="all">All statuses</option><option value="active">Active</option><option value="expired">Expired</option><option value="revoked">Revoked</option></select></Card.Content></Card>

      <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_380px]" : "grid-cols-1"}`}>
        <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm font-medium text-slate-100">Credential registry</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">{filtered.length} visible keys · secrets are never stored in this table</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="API key registry" className="min-w-[1020px]"><Table.Header>{[["key", "Key"], ["status", "Status"], ["project", "Project"], ["models", "Model access"], ["limits", "Limits"], ["spend", "Spend"], ["activity", "Last used"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} isRowHeader={id === "key"} className={`bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase ${["limits", "spend"].includes(id) ? "text-right" : ""}`}>{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((key) => <Table.Row key={key.id} id={key.id} onClick={() => setSelectedId(key.id)} className={`cursor-pointer border-t border-white/[0.045] transition hover:bg-white/[0.03] ${selectedId === key.id ? "bg-blue-400/[0.055]" : ""}`}><Table.Cell><p className="max-w-56 truncate text-xs font-medium text-slate-200">{key.name}</p><p className="mt-0.5 font-mono text-[10px] text-slate-600">{key.prefix}••••••••</p></Table.Cell><Table.Cell><KeyStatusBadge status={key.status} /></Table.Cell><Table.Cell><p className="max-w-44 truncate text-xs text-slate-300">{key.project}</p><p className="text-[10px] text-slate-600">{key.createdBy}</p></Table.Cell><Table.Cell><p className="max-w-44 truncate text-xs text-slate-300">{key.models.join(", ")}</p><p className="text-[10px] text-slate-600">{key.models.length} allowed</p></Table.Cell><Table.Cell className="text-right"><p className="font-mono text-[11px] text-slate-300">{key.rpm.toLocaleString()} RPM</p><p className="font-mono text-[10px] text-slate-600">{key.tpm.toLocaleString()} TPM</p></Table.Cell><Table.Cell className="text-right font-mono text-[11px] text-slate-300">${key.spendUsd.toFixed(2)}</Table.Cell><Table.Cell><p className="text-[11px] text-slate-400">{key.lastUsedAt ? new Intl.DateTimeFormat("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hour12: false }).format(new Date(key.lastUsedAt)) : "Never"}</p><p className="text-[10px] text-slate-600">Expires {key.expiresAt ?? "never"}</p></Table.Cell><Table.Cell><ChevronRight className="ml-auto text-slate-700" size={14} /></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>

        {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex items-center justify-between"><span className="rounded-xl bg-blue-400/10 p-2 text-blue-300"><KeyRound size={17} /></span><KeyStatusBadge status={selected.status} /></div><h2 className="mt-4 text-base font-semibold text-white">{selected.name}</h2><p className="mt-1 font-mono text-[10px] text-slate-600">{selected.prefix}••••••••••••</p></div><div className="space-y-5 p-5"><div className="grid grid-cols-2 gap-2">{[["RPM", selected.rpm.toLocaleString()], ["TPM", selected.tpm.toLocaleString()], ["Spend", `$${selected.spendUsd.toFixed(2)}`], ["Models", selected.models.length]].map(([label, value]) => <div key={label} className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><p className="text-[9px] tracking-wider text-slate-600 uppercase">{label}</p><p className="mt-1.5 text-sm font-medium text-slate-200">{value}</p></div>)}</div><section><p className="mb-2 text-[10px] font-medium tracking-wider text-slate-500 uppercase">Allowed models</p><div className="flex flex-wrap gap-1.5">{selected.models.map((model) => <span key={model} className="rounded-lg bg-white/[0.035] px-2 py-1.5 font-mono text-[10px] text-slate-300 ring-1 ring-inset ring-white/[0.06]">{model}</span>)}</div></section><dl className="space-y-3 text-[11px]">{[["Project", selected.project], ["Created by", selected.createdBy], ["Created", selected.createdAt], ["Expires", selected.expiresAt ?? "Never"]].map(([label, value]) => <div key={label} className="flex justify-between gap-4"><dt className="text-slate-600">{label}</dt><dd className="truncate text-slate-300">{value}</dd></div>)}</dl>{selected.status === "active" ? <Button onPress={revokeSelected} variant="tertiary" className="h-9 w-full gap-2 border border-rose-400/15 bg-rose-400/[0.045] text-xs text-rose-300"><ShieldX size={13} /> Revoke key</Button> : <div className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-3 text-center text-[11px] text-slate-500">This credential can no longer authenticate gateway requests.</div>}</div></aside>}
      </section>

      {createOpen && <CreateKeyDialog onClose={() => setCreateOpen(false)} onCreate={createKey} />}
      {newSecret && <SecretDialog secret={newSecret} onClose={() => setNewSecret(null)} />}
    </div>
  );
}
