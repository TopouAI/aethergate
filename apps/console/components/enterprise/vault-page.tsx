"use client";

import { Button, Card, Table } from "@heroui/react";
import { AlertTriangle, Check, ChevronRight, Clock3, Copy, History, KeyRound, LockKeyhole, Plus, RefreshCcw, ShieldCheck, ShieldX } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Dialog, Field, inputClass, MetricGrid, PageHeader, SearchBar, selectClass, StateBadge, type APIState } from "@/components/foundation/resource-ui";
import { createVaultSecret, disableVaultSecret, listVaultAccessEvents, listVaultSecrets, rotateVaultSecret } from "@/lib/control-plane";
import { foundationOrganizationId } from "@/lib/foundation-data";
import { seedVaultAccessEvents, seedVaultSecrets } from "@/lib/vault-data";
import type { VaultAccessEvent, VaultScopeType, VaultSecret, VaultSecretKind } from "@/types/vault";

type Workspace = "secrets" | "access";
const operatorEmail = "holden@topoai.dev";
const kinds: VaultSecretKind[] = ["provider_api_key", "webhook_signing_secret", "integration_token", "smtp_password", "object_storage_key", "database_credential", "generic"];
const scopeTypes: VaultScopeType[] = ["provider", "webhook", "notification", "reporting", "gateway", "integration", "organization"];

export function VaultPage() {
  const [secrets, setSecrets] = useState<VaultSecret[]>(seedVaultSecrets);
  const [accessEvents, setAccessEvents] = useState<VaultAccessEvent[]>(seedVaultAccessEvents);
  const [workspace, setWorkspace] = useState<Workspace>("secrets");
  const [selectedId, setSelectedId] = useState(seedVaultSecrets[0]?.id ?? "");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [kindFilter, setKindFilter] = useState("all");
  const [rotationFilter, setRotationFilter] = useState("all");
  const [apiState, setAPIState] = useState<APIState>("connecting");
  const [notice, setNotice] = useState("");
  const [currentTime, setCurrentTime] = useState<number | null>(null);
  const [copied, setCopied] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [rotateOpen, setRotateOpen] = useState(false);
  const [disableOpen, setDisableOpen] = useState(false);
  const [name, setName] = useState("");
  const [kind, setKind] = useState<VaultSecretKind>("provider_api_key");
  const [scopeType, setScopeType] = useState<VaultScopeType>("provider");
  const [scopeId, setScopeId] = useState("");
  const [secretValue, setSecretValue] = useState("");
  const [rotationDays, setRotationDays] = useState("90");
  const [expiresAt, setExpiresAt] = useState("");
  const [rotateValue, setRotateValue] = useState("");
  const [rotateReason, setRotateReason] = useState("Scheduled credential rotation");
  const [disableReason, setDisableReason] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      listVaultSecrets({ organizationId: foundationOrganizationId }, controller.signal),
      listVaultAccessEvents({ organizationId: foundationOrganizationId }, controller.signal),
    ])
      .then(([secretResponse, accessResponse]) => {
        setCurrentTime(Date.now());
        setSecrets(secretResponse.data);
        setAccessEvents(accessResponse.data);
        setSelectedId((current) => secretResponse.data.some((item) => item.id === current) ? current : secretResponse.data[0]?.id ?? "");
        setAPIState("connected");
      })
      .catch((error: unknown) => {
        if (!(error instanceof DOMException && error.name === "AbortError")) {
          setCurrentTime(Date.now());
          setAPIState("fallback");
        }
      });
    return () => controller.abort();
  }, []);

  const now = currentTime ?? 0;
  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return secrets.filter((secret) => {
      const due = new Date(secret.rotationDueAt).getTime();
      const rotationMatches = rotationFilter === "all" ||
        (rotationFilter === "overdue" && secret.status === "active" && due < now) ||
        (rotationFilter === "due" && secret.status === "active" && due >= now && due <= now + 30 * 86400000) ||
        (rotationFilter === "healthy" && secret.status === "active" && due > now + 30 * 86400000);
      return (status === "all" || secret.status === status) &&
        (kindFilter === "all" || secret.kind === kindFilter) && rotationMatches &&
        (!needle || [secret.name, secret.kind, secret.scopeType, secret.scopeId, secret.reference, secret.fingerprint].some((value) => value.toLowerCase().includes(needle)));
    });
  }, [kindFilter, now, query, rotationFilter, secrets, status]);

  const selected = secrets.find((item) => item.id === selectedId) ?? null;
  const overdue = secrets.filter((item) => item.status === "active" && new Date(item.rotationDueAt).getTime() < now).length;
  const dueSoon = secrets.filter((item) => {
    const due = new Date(item.rotationDueAt).getTime();
    return item.status === "active" && due >= now && due <= now + 30 * 86400000;
  }).length;

  async function submitCreate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAPIState("saving");
    setNotice("");
    try {
      const response = await createVaultSecret({
        organizationId: foundationOrganizationId, name, kind, scopeType, scopeId, secretValue,
        rotationIntervalDays: Number(rotationDays), expiresAt: expiresAt ? new Date(expiresAt).toISOString() : "",
        createdBy: operatorEmail, requestId: `ui_${crypto.randomUUID()}`, sourceIp: "",
      });
      setSecrets((current) => [response.data, ...current]);
      setSelectedId(response.data.id);
      setCreateOpen(false);
      setName(""); setScopeId(""); setExpiresAt("");
      setNotice("Secret encrypted server-side. The browser received metadata only.");
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Secret was not stored. No local fallback is permitted for secret material.");
      setAPIState("fallback");
    } finally {
      setSecretValue("");
    }
  }

  async function submitRotate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    setAPIState("saving");
    setNotice("");
    try {
      const response = await rotateVaultSecret(selected.id, {
        organizationId: foundationOrganizationId, secretValue: rotateValue, reason: rotateReason,
        rotatedBy: operatorEmail, requestId: `ui_${crypto.randomUUID()}`, sourceIp: "",
      });
      setSecrets((current) => current.map((item) => item.id === response.data.id ? response.data : item));
      setRotateOpen(false);
      setNotice(`Version ${response.data.currentVersion} encrypted; prior material is superseded.`);
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Secret rotation failed; existing material was not changed.");
      setAPIState("fallback");
    } finally {
      setRotateValue("");
    }
  }

  async function submitDisable(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    setAPIState("saving");
    try {
      const response = await disableVaultSecret(selected.id, {
        organizationId: foundationOrganizationId, reason: disableReason, disabledBy: operatorEmail,
        requestId: `ui_${crypto.randomUUID()}`, sourceIp: "",
      });
      setSecrets((current) => current.map((item) => item.id === response.data.id ? response.data : item));
      setDisableOpen(false);
      setDisableReason("");
      setNotice("Secret disabled. Internal worker resolution will now be denied and recorded.");
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Secret could not be disabled.");
      setAPIState("fallback");
    }
  }

  async function copyReference() {
    if (!selected) return;
    await navigator.clipboard.writeText(selected.reference);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  }

  return <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
    <PageHeader eyebrow="Secrets governance" title="Enterprise Vault" description="Store provider and integration credentials as versioned envelope-encrypted records; the Console receives only masked metadata and immutable access evidence." icon={LockKeyhole} apiState={apiState} action={<Button onPress={() => setCreateOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={14}/>Store secret</Button>}/>
    {notice && <div className="rounded-xl border border-amber-400/15 bg-amber-400/[0.04] px-4 py-3 text-xs text-amber-200">{notice}</div>}
    <MetricGrid items={[
      { label: "Secret records", value: secrets.length.toString(), hint: `${secrets.filter((item) => item.status === "active").length} active`, icon: KeyRound },
      { label: "Overdue rotation", value: overdue.toString(), hint: "Requires security review", icon: AlertTriangle },
      { label: "Due in 30 days", value: dueSoon.toString(), hint: "Planned rotation window", icon: Clock3 },
      { label: "Access evidence", value: accessEvents.length.toString(), hint: "Append-only events", icon: History },
    ]}/>

    <nav className="glass-panel flex flex-wrap gap-2 p-2" aria-label="Vault workspaces">
      {([['secrets', 'Secret registry', LockKeyhole], ['access', 'Access evidence', History]] as const).map(([id, label, Icon]) => <button key={id} type="button" onClick={() => setWorkspace(id)} className={`flex items-center gap-2 rounded-xl px-3 py-2 text-xs transition ${workspace === id ? "bg-blue-500/15 text-blue-200 ring-1 ring-inset ring-blue-400/20" : "text-slate-500 hover:bg-white/[0.035] hover:text-slate-200"}`}><Icon size={13}/>{label}</button>)}
    </nav>

    {workspace === "secrets" && <>
      <SearchBar value={query} onChange={setQuery} placeholder="Search name, scope, reference, or fingerprint..." trailing={<div className="grid gap-2 sm:grid-cols-3"><select value={status} onChange={(event) => setStatus(event.target.value)} className={selectClass}><option value="all">All states</option><option value="active">Active</option><option value="disabled">Disabled</option></select><select value={kindFilter} onChange={(event) => setKindFilter(event.target.value)} className={selectClass}><option value="all">All kinds</option>{kinds.map((item) => <option key={item} value={item}>{labelValue(item)}</option>)}</select><select value={rotationFilter} onChange={(event) => setRotationFilter(event.target.value)} className={selectClass}><option value="all">All rotation states</option><option value="overdue">Overdue</option><option value="due">Due in 30 days</option><option value="healthy">Healthy</option></select></div>}/>
      <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_400px]" : "grid-cols-1"}`}>
        <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">Encrypted secret registry</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">{filtered.length} metadata records · ciphertext and data keys never enter this API response</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Vault secret registry" className="min-w-[1040px]"><Table.Header>{[["secret","Secret"],["kind","Kind"],["scope","Scope"],["status","Status"],["version","Version"],["rotation","Rotation"],["fingerprint","Fingerprint"],["open",""]].map(([id,label]) => <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((item) => <Table.Row key={item.id} id={item.id} onClick={() => setSelectedId(item.id)} className={`cursor-pointer border-t border-white/[0.045] hover:bg-white/[0.03] ${selectedId === item.id ? "bg-blue-400/[0.055]" : ""}`}><Table.Cell><p className="text-xs font-medium text-slate-200">{item.name}</p><p className="mt-1 font-mono text-[10px] text-slate-600">{item.maskedValue}</p></Table.Cell><Table.Cell className="text-[11px] text-slate-400">{labelValue(item.kind)}</Table.Cell><Table.Cell><p className="text-[11px] capitalize text-slate-400">{item.scopeType}</p><p className="max-w-44 truncate font-mono text-[9px] text-slate-700">{item.scopeId}</p></Table.Cell><Table.Cell><StateBadge value={item.status}/></Table.Cell><Table.Cell className="font-mono text-xs text-slate-300">v{item.currentVersion}</Table.Cell><Table.Cell><RotationBadge secret={item} now={currentTime}/></Table.Cell><Table.Cell className="font-mono text-[10px] text-slate-500">{item.fingerprint}</Table.Cell><Table.Cell><ChevronRight size={14} className="ml-auto text-slate-700"/></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>
        {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex justify-between"><span className="rounded-xl bg-blue-400/10 p-2 text-blue-300"><LockKeyhole size={17}/></span><StateBadge value={selected.status}/></div><h2 className="mt-4 text-base font-semibold text-white">{selected.name}</h2><p className="mt-1 font-mono text-[10px] text-slate-600">{selected.maskedValue}</p></div><div className="space-y-5 p-5"><div className="rounded-xl border border-emerald-400/15 bg-emerald-400/[0.03] p-4"><p className="flex items-center gap-2 text-xs text-emerald-200"><ShieldCheck size={13}/>Envelope encrypted</p><p className="mt-1 text-[10px] leading-5 text-slate-600">AES-256-GCM secret data + separately wrapped random data key. Tenant, secret ID, and version are authenticated context.</p></div><dl className="space-y-3 text-[11px]">{[["Kind",labelValue(selected.kind)],["Scope",`${selected.scopeType}/${selected.scopeId}`],["Version",`v${selected.currentVersion}`],["Fingerprint",selected.fingerprint],["Last rotated",new Date(selected.lastRotatedAt).toLocaleString("zh-CN")],["Rotation due",new Date(selected.rotationDueAt).toLocaleString("zh-CN")],["Created by",selected.createdBy]].map(([label,value]) => <div key={label} className="flex justify-between gap-4"><dt className="text-slate-600">{label}</dt><dd className="max-w-60 truncate text-right text-slate-300">{value}</dd></div>)}</dl><div><p className="mb-2 text-[9px] uppercase text-slate-600">Secret reference</p><button type="button" onClick={copyReference} className="flex w-full items-center justify-between gap-3 rounded-xl border border-white/[0.055] bg-black/20 p-3 text-left"><span className="truncate font-mono text-[10px] text-blue-200/70">{selected.reference}</span>{copied ? <Check size={12} className="text-emerald-300"/> : <Copy size={12} className="text-slate-600"/>}</button></div>{selected.status === "active" ? <div className="grid grid-cols-2 gap-2"><Button onPress={() => setRotateOpen(true)} className="h-9 gap-2 bg-blue-500 text-xs text-white"><RefreshCcw size={12}/>Rotate</Button><Button onPress={() => setDisableOpen(true)} variant="tertiary" className="h-9 gap-2 border border-rose-400/15 text-xs text-rose-300"><ShieldX size={12}/>Disable</Button></div> : <div className="rounded-xl border border-rose-400/12 bg-rose-400/[0.03] p-3 text-[10px] leading-5 text-rose-100/60">Disabled by {selected.disabledBy}: {selected.disabledReason}</div>}<p className="text-[9px] leading-4 text-slate-700">No browser or public HTTP endpoint can resolve this reference. Only internal workers with an explicit actor, workload, purpose, request ID, and access record may decrypt it.</p></div></aside>}
      </section>
    </>}

    {workspace === "access" && <Card className="glass-panel overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">Immutable access evidence</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Management and internal-resolution attempts are recorded without secret values.</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Vault access evidence" className="min-w-[1040px]"><Table.Header>{[["secret","Secret"],["actor","Actor / workload"],["purpose","Purpose"],["outcome","Outcome"],["request","Request"],["source","Source IP"],["time","Created"]].map(([id,label]) => <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{accessEvents.map((event) => <Table.Row key={event.id} id={event.id} className="border-t border-white/[0.045]"><Table.Cell><p className="text-xs text-slate-200">{event.secretName}</p><p className="font-mono text-[9px] text-slate-700">{event.secretId} · v{event.secretVersion}</p></Table.Cell><Table.Cell><p className="text-[11px] text-slate-300">{event.actor}</p><p className="font-mono text-[9px] text-slate-700">{event.workload}</p></Table.Cell><Table.Cell className="max-w-72 text-[11px] text-slate-400">{event.purpose}</Table.Cell><Table.Cell><AccessBadge value={event.outcome}/></Table.Cell><Table.Cell className="font-mono text-[10px] text-slate-500">{event.requestId || "—"}</Table.Cell><Table.Cell className="font-mono text-[10px] text-slate-500">{event.sourceIp || "internal"}</Table.Cell><Table.Cell className="text-[10px] text-slate-500">{new Date(event.createdAt).toLocaleString("zh-CN")}</Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>}

    {createOpen && <Dialog title="Store encrypted secret" description="The value is sent once over the control-plane connection, envelope-encrypted, and never returned." submitLabel="Encrypt and store" canSubmit={Boolean(name.trim() && scopeId.trim() && secretValue.length >= 8 && Number(rotationDays) >= 1 && Number(rotationDays) <= 365)} onClose={() => { setCreateOpen(false); setSecretValue(""); }} onSubmit={submitCreate}><Field label="Secret name"><input autoFocus value={name} onChange={(event) => setName(event.target.value)} className={inputClass} placeholder="OpenAI Production API Key"/></Field><div className="grid gap-4 sm:grid-cols-2"><Field label="Kind"><select value={kind} onChange={(event) => setKind(event.target.value as VaultSecretKind)} className={selectClass}>{kinds.map((item) => <option key={item} value={item}>{labelValue(item)}</option>)}</select></Field><Field label="Scope type"><select value={scopeType} onChange={(event) => setScopeType(event.target.value as VaultScopeType)} className={selectClass}>{scopeTypes.map((item) => <option key={item}>{labelValue(item)}</option>)}</select></Field><Field label="Scope ID"><input value={scopeId} onChange={(event) => setScopeId(event.target.value)} className={inputClass} placeholder="provider_openai_primary"/></Field><Field label="Rotation interval days"><input type="number" min="1" max="365" value={rotationDays} onChange={(event) => setRotationDays(event.target.value)} className={inputClass}/></Field></div><Field label="Secret value"><input type="password" autoComplete="new-password" value={secretValue} onChange={(event) => setSecretValue(event.target.value)} className={inputClass} placeholder="Value is never displayed again"/></Field><Field label="Optional expiry"><input type="datetime-local" value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} className={inputClass}/></Field><div className="rounded-xl border border-amber-400/12 bg-amber-400/[0.03] p-3 text-[10px] leading-5 text-amber-100/65">There is intentionally no client-side fallback. If encryption or persistence fails, AetherGate reports the error and discards this form value.</div></Dialog>}
    {rotateOpen && selected && <Dialog title={`Rotate ${selected.name}`} description={`Create encrypted version ${selected.currentVersion + 1}; the current version becomes superseded after an atomic store update.`} submitLabel="Rotate secret" canSubmit={rotateValue.length >= 8 && Boolean(rotateReason.trim())} onClose={() => { setRotateOpen(false); setRotateValue(""); }} onSubmit={submitRotate}><Field label="New secret value"><input autoFocus type="password" autoComplete="new-password" value={rotateValue} onChange={(event) => setRotateValue(event.target.value)} className={inputClass}/></Field><Field label="Rotation reason"><input value={rotateReason} onChange={(event) => setRotateReason(event.target.value)} className={inputClass}/></Field></Dialog>}
    {disableOpen && selected && <Dialog title={`Disable ${selected.name}`} description="Internal resolution will be denied. This is intentionally irreversible through the current API." submitLabel="Disable secret" canSubmit={disableReason.trim().length >= 4} onClose={() => setDisableOpen(false)} onSubmit={submitDisable}><Field label="Disable reason"><input autoFocus value={disableReason} onChange={(event) => setDisableReason(event.target.value)} className={inputClass} placeholder="Provider account retired"/></Field><div className="rounded-xl border border-rose-400/12 bg-rose-400/[0.03] p-3 text-[10px] leading-5 text-rose-100/65">Disabling may make provider routes or external delivery connectors unavailable. Verify fallback coverage first.</div></Dialog>}
  </div>;
}

function labelValue(value: string) { return value.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase()); }

function RotationBadge({ secret, now }: { secret: VaultSecret; now: number | null }) {
  if (secret.status !== "active") return <span className="text-[10px] text-slate-700">Not scheduled</span>;
  if (now === null) return <span className="text-[10px] text-slate-600">Scheduled</span>;
  const days = Math.ceil((new Date(secret.rotationDueAt).getTime() - now) / 86400000);
  const style = days < 0 ? "bg-rose-400/10 text-rose-300 ring-rose-400/20" : days <= 30 ? "bg-amber-400/10 text-amber-300 ring-amber-400/20" : "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20";
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] ring-1 ring-inset ${style}`}>{days < 0 ? `${Math.abs(days)}d overdue` : `${days}d`}</span>;
}

function AccessBadge({ value }: { value: VaultAccessEvent["outcome"] }) {
  const style = value === "success" ? "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20" : "bg-rose-400/10 text-rose-300 ring-rose-400/20";
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] capitalize ring-1 ring-inset ${style}`}>{value}</span>;
}
