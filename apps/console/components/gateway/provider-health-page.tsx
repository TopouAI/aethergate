"use client";

import { Button, Card, Table } from "@heroui/react";
import { Activity, ChevronRight, CircleGauge, Pause, Play, Plus, RadioTower, RefreshCcw, ServerCog, ShieldCheck, TriangleAlert } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Dialog, Field, inputClass, MetricGrid, PageHeader, SearchBar, selectClass, StateBadge, type APIState } from "@/components/foundation/resource-ui";
import {
  createProvider, listProviderHealthEvents, listProviderHealthProbes, listProviders,
  queueProviderHealthProbe, recordProviderHealth, setProviderMaintenance,
} from "@/lib/control-plane";
import { foundationOrganizationId } from "@/lib/foundation-data";
import { seedProviderHealthEvents, seedProviderHealthProbes } from "@/lib/provider-health-data";
import { seedProviders } from "@/lib/provider-data";
import type { ProviderConnection, ProviderHealthEvent, ProviderHealthProbe, ProviderStatus } from "@/types/provider";

export function ProviderHealthPage() {
  const [providers, setProviders] = useState<ProviderConnection[]>(seedProviders);
  const [events, setEvents] = useState<ProviderHealthEvent[]>(seedProviderHealthEvents);
  const [probes, setProbes] = useState<ProviderHealthProbe[]>(seedProviderHealthProbes);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"all" | ProviderStatus>("all");
  const [selectedId, setSelectedId] = useState(seedProviders[0]?.id ?? "");
  const [apiState, setAPIState] = useState<APIState>("connecting");
  const [notice, setNotice] = useState("");
  const [registerOpen, setRegisterOpen] = useState(false);
  const [telemetryOpen, setTelemetryOpen] = useState(false);
  const [maintenanceOpen, setMaintenanceOpen] = useState(false);

  const [name, setName] = useState("");
  const [providerFamily, setProviderFamily] = useState("OpenAI");
  const [baseUrl, setBaseUrl] = useState("https://api.openai.com/v1");
  const [requestCount, setRequestCount] = useState("1000");
  const [errorCount, setErrorCount] = useState("5");
  const [averageLatency, setAverageLatency] = useState("850");
  const [p95Latency, setP95Latency] = useState("1400");
  const [maintenanceUntil, setMaintenanceUntil] = useState(() => new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString().slice(0, 16));
  const [maintenanceReason, setMaintenanceReason] = useState("Planned provider maintenance");

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      listProviders({ organizationId: foundationOrganizationId }, controller.signal),
      listProviderHealthEvents({ organizationId: foundationOrganizationId }, controller.signal),
      listProviderHealthProbes({ organizationId: foundationOrganizationId }, controller.signal),
    ]).then(([providerResponse, eventResponse, probeResponse]) => {
      setProviders(providerResponse.data);
      setEvents(eventResponse.data);
      setProbes(probeResponse.data);
      setSelectedId((current) => providerResponse.data.some((item) => item.id === current) ? current : providerResponse.data[0]?.id ?? "");
      setAPIState("connected");
    }).catch((error: unknown) => {
      if (!(error instanceof DOMException && error.name === "AbortError")) setAPIState("fallback");
    });
    return () => controller.abort();
  }, []);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return providers.filter((item) => (status === "all" || item.status === status)
      && (!needle || [item.name, item.provider, item.baseUrl, item.healthReason].some((value) => value.toLowerCase().includes(needle))));
  }, [providers, query, status]);
  const selected = providers.find((item) => item.id === selectedId) ?? null;
  const selectedEvents = events.filter((item) => !selected || item.providerId === selected.id);
  const selectedProbes = probes.filter((item) => !selected || item.providerId === selected.id);

  async function register(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAPIState("saving");
    try {
      const response = await createProvider({ organizationId: foundationOrganizationId, name, provider: providerFamily, baseUrl });
      setProviders((current) => [response.data, ...current]);
      setSelectedId(response.data.id);
      setRegisterOpen(false);
      setName("");
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Provider could not be registered.");
      setAPIState("fallback");
    }
  }

  async function queueProbe() {
    if (!selected) return;
    try {
      const response = await queueProviderHealthProbe(selected.id, { organizationId: foundationOrganizationId, region: "automatic", model: "provider-default", requestedBy: "holden@topoai.dev" });
      setProbes((current) => [response.data, ...current]);
      setNotice("Active probe accepted by the provider-health worker queue.");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Probe could not be queued.");
    }
  }

  async function recordTelemetry(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) return;
    try {
      const response = await recordProviderHealth(selected.id, {
        organizationId: foundationOrganizationId, source: "passive_telemetry", success: Number(errorCount) === 0,
        requestCount: Number(requestCount), errorCount: Number(errorCount), averageLatencyMs: Number(averageLatency),
        p95LatencyMs: Number(p95Latency), httpStatus: null, probeId: null, message: "Console telemetry simulator",
      });
      setEvents((current) => [response.data, ...current]);
      setProviders((current) => current.map((item) => item.id === selected.id ? {
        ...item, status: response.data.status, routingEligible: response.data.routingEligible,
        healthSource: response.data.source, healthReason: response.data.reason, errorRate: response.data.errorRate,
        requestCount24h: response.data.requestCount, averageLatencyMs: response.data.averageLatencyMs,
        p95LatencyMs: response.data.p95LatencyMs, successRate: 100 - response.data.errorRate,
        consecutiveFailures: response.data.consecutiveFailures, lastCheckedAt: response.data.observedAt,
        lastTransitionAt: response.data.transition ? response.data.observedAt : item.lastTransitionAt,
      } : item));
      setTelemetryOpen(false);
      setNotice(`Health recomputed as ${response.data.status}; routing eligible: ${response.data.routingEligible}.`);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Telemetry could not be recorded.");
    }
  }

  async function changeMaintenance(enabled: boolean, event?: React.FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    if (!selected) return;
    try {
      const response = await setProviderMaintenance(selected.id, {
        organizationId: foundationOrganizationId, enabled,
        until: enabled ? new Date(maintenanceUntil).toISOString() : null,
        reason: enabled ? maintenanceReason : "",
      });
      setProviders((current) => current.map((item) => item.id === response.data.id ? response.data : item));
      setMaintenanceOpen(false);
      setNotice(enabled ? "Maintenance enabled; provider removed from routing." : "Maintenance ended; fresh health evidence is required before routing.");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Maintenance state could not be changed.");
    }
  }

  return <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
    <PageHeader eyebrow="Gateway reliability" title="Providers & health" description="Combine active probes and passive traffic telemetry into debounced provider states that directly control routing eligibility." icon={RadioTower} apiState={apiState} action={<Button onPress={() => setRegisterOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={14} />Register provider</Button>} />
    {notice && <div className="rounded-xl border border-amber-400/15 bg-amber-400/[0.04] px-4 py-3 text-xs text-amber-200">{notice}</div>}
    <MetricGrid items={[
      { label: "Connections", value: providers.length.toString(), hint: `${new Set(providers.map((item) => item.provider)).size} provider families`, icon: RadioTower },
      { label: "Routable", value: providers.filter((item) => item.routingEligible).length.toString(), hint: "Healthy + credentials + no maintenance", icon: ShieldCheck },
      { label: "Degraded/offline", value: providers.filter((item) => item.status === "degraded" || item.status === "offline").length.toString(), hint: "Excluded or at risk", icon: TriangleAlert },
      { label: "Probe queue", value: probes.filter((item) => item.status === "queued" || item.status === "running").length.toString(), hint: `${events.filter((item) => item.transition).length} recorded transitions`, icon: Activity },
    ]} />
    <SearchBar value={query} onChange={setQuery} placeholder="Search connection, provider, URL, or health reason..." trailing={<select value={status} onChange={(event) => setStatus(event.target.value as "all" | ProviderStatus)} className={selectClass}><option value="all">All health states</option><option value="healthy">Healthy</option><option value="degraded">Degraded</option><option value="offline">Offline</option><option value="maintenance">Maintenance</option><option value="configuring">Configuring</option></select>} />

    <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_390px]" : "grid-cols-1"}`}>
      <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">Provider health registry</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Routing eligibility is derived, never inferred from a green badge alone.</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Provider health registry" className="min-w-[1120px]"><Table.Header>{[["connection", "Connection"], ["status", "Health"], ["routing", "Routing"], ["source", "Evidence"], ["traffic", "24h traffic"], ["error", "Error rate"], ["latency", "Avg / P95"], ["checked", "Observed"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((item) => <Table.Row key={item.id} id={item.id} onClick={() => setSelectedId(item.id)} className={`cursor-pointer border-t border-white/[0.045] hover:bg-white/[0.03] ${selectedId === item.id ? "bg-blue-400/[0.055]" : ""}`}><Table.Cell><p className="text-xs font-medium text-slate-200">{item.name}</p><p className="max-w-64 truncate text-[10px] text-slate-600">{item.provider} · {item.baseUrl}</p></Table.Cell><Table.Cell><StateBadge value={item.status} /></Table.Cell><Table.Cell><span className={`rounded-full px-2 py-1 text-[10px] ring-1 ring-inset ${item.routingEligible ? "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20" : "bg-rose-400/10 text-rose-300 ring-rose-400/20"}`}>{item.routingEligible ? "Eligible" : "Excluded"}</span></Table.Cell><Table.Cell><p className="text-[11px] text-slate-400">{item.healthSource.replaceAll("_", " ")}</p><p className="text-[9px] text-slate-700">{item.consecutiveFailures} consecutive failures</p></Table.Cell><Table.Cell className="font-mono text-[11px] text-slate-400">{item.requestCount24h.toLocaleString()}</Table.Cell><Table.Cell className="font-mono text-[11px] text-slate-400">{item.errorRate.toFixed(3)}%</Table.Cell><Table.Cell><p className="font-mono text-[11px] text-slate-300">{item.averageLatencyMs} / {item.p95LatencyMs}ms</p></Table.Cell><Table.Cell className="text-[11px] text-slate-500">{item.lastCheckedAt ? new Date(item.lastCheckedAt).toLocaleString("zh-CN") : "Pending"}</Table.Cell><Table.Cell><ChevronRight size={14} className="ml-auto text-slate-700" /></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>

      {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex justify-between"><span className="rounded-xl bg-blue-400/10 p-2 text-blue-300"><ServerCog size={17} /></span><StateBadge value={selected.status} /></div><h2 className="mt-4 text-base font-semibold text-white">{selected.name}</h2><p className="mt-1 text-[10px] text-slate-600">{selected.provider}</p></div><div className="space-y-5 p-5"><div className={`rounded-xl border p-4 ${selected.routingEligible ? "border-emerald-400/15 bg-emerald-400/[0.035]" : "border-rose-400/15 bg-rose-400/[0.035]"}`}><p className="text-xs font-medium text-slate-200">{selected.routingEligible ? "Routing eligible" : "Excluded from routing"}</p><p className="mt-1 text-[10px] leading-5 text-slate-500">{selected.healthReason}</p></div><div className="grid grid-cols-2 gap-2">{[["Requests", selected.requestCount24h.toLocaleString()], ["Error", `${selected.errorRate.toFixed(3)}%`], ["Average", `${selected.averageLatencyMs}ms`], ["P95", `${selected.p95LatencyMs}ms`]].map(([label, value]) => <div key={label} className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><p className="text-[9px] uppercase text-slate-600">{label}</p><p className="mt-1.5 font-mono text-sm text-slate-200">{value}</p></div>)}</div><dl className="space-y-3 text-[11px]">{[["Credentials", selected.credentialState], ["Evidence", selected.healthSource.replaceAll("_", " ")], ["Failures", selected.consecutiveFailures], ["Maintenance", selected.maintenanceUntil ? new Date(selected.maintenanceUntil).toLocaleString("zh-CN") : "None"]].map(([label, value]) => <div key={label} className="flex justify-between gap-4"><dt className="text-slate-600">{label}</dt><dd className="max-w-56 truncate text-right capitalize text-slate-300">{value}</dd></div>)}</dl><div className="grid grid-cols-2 gap-2"><Button onPress={queueProbe} isDisabled={selected.credentialState !== "configured" || selected.status === "maintenance"} variant="tertiary" className="h-9 gap-2 border border-white/8 text-xs text-slate-300"><RefreshCcw size={13} />Queue probe</Button><Button onPress={() => setTelemetryOpen(true)} variant="tertiary" className="h-9 gap-2 border border-white/8 text-xs text-slate-300"><CircleGauge size={13} />Telemetry</Button>{selected.status === "maintenance" ? <Button onPress={() => changeMaintenance(false)} className="col-span-2 h-9 gap-2 bg-blue-500 text-xs text-white"><Play size={13} />End maintenance</Button> : <Button onPress={() => setMaintenanceOpen(true)} variant="tertiary" className="col-span-2 h-9 gap-2 border border-amber-400/15 text-xs text-amber-200"><Pause size={13} />Schedule maintenance</Button>}</div></div></aside>}
    </section>

    <section className="grid gap-4 xl:grid-cols-2"><Card className="glass-panel px-0 py-0"><Card.Header className="p-4"><Card.Title className="text-sm text-slate-100">Health evidence</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Active and passive observations with status transition evidence.</Card.Description></Card.Header><Card.Content className="space-y-2 border-t border-white/[0.055] p-4">{selectedEvents.slice(0, 8).map((event) => <div key={event.id} className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><div className="flex items-center justify-between"><div className="flex items-center gap-2"><StateBadge value={event.status} /><span className="text-[9px] uppercase text-slate-600">{event.source.replaceAll("_", " ")}</span></div><span className="text-[10px] text-slate-600">{new Date(event.observedAt).toLocaleString("zh-CN")}</span></div><p className="mt-2 text-[11px] text-slate-400">{event.reason}</p><p className="mt-1 font-mono text-[9px] text-slate-700">{event.requestCount} requests · {event.errorRate.toFixed(3)}% errors · P95 {event.p95LatencyMs}ms{event.transition ? ` · ${event.previousStatus} → ${event.status}` : ""}</p></div>)}</Card.Content></Card><Card className="glass-panel px-0 py-0"><Card.Header className="p-4"><Card.Title className="text-sm text-slate-100">Active probe queue</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Worker-owned network checks; control plane stores intent and evidence.</Card.Description></Card.Header><Card.Content className="space-y-2 border-t border-white/[0.055] p-4">{selectedProbes.slice(0, 8).map((probe) => <div key={probe.id} className="flex items-center justify-between rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><div><p className="text-xs text-slate-300">{probe.model} · {probe.region}</p><p className="mt-1 font-mono text-[9px] text-slate-700">{probe.id} · requested by {probe.requestedBy}</p></div><div className="text-right"><StateBadge value={probe.status} /><p className="mt-1 text-[9px] text-slate-700">{new Date(probe.requestedAt).toLocaleString("zh-CN")}</p></div></div>)}</Card.Content></Card></section>

    {registerOpen && <Dialog title="Register provider" description="Create non-secret provider metadata in a non-routable configuring state." submitLabel="Register provider" canSubmit={Boolean(name.trim() && providerFamily.trim() && /^https?:\/\//.test(baseUrl))} onClose={() => setRegisterOpen(false)} onSubmit={register}><Field label="Connection name"><input autoFocus value={name} onChange={(event) => setName(event.target.value)} className={inputClass} /></Field><div className="grid gap-4 sm:grid-cols-2"><Field label="Provider family"><input value={providerFamily} onChange={(event) => setProviderFamily(event.target.value)} className={inputClass} /></Field><Field label="Base URL"><input type="url" value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} className={inputClass} /></Field></div></Dialog>}
    {telemetryOpen && <Dialog title="Record passive telemetry" description="Simulate an aggregated traffic observation and recompute routing eligibility." submitLabel="Record observation" canSubmit={Number(requestCount) > 0 && Number(errorCount) >= 0 && Number(errorCount) <= Number(requestCount)} onClose={() => setTelemetryOpen(false)} onSubmit={recordTelemetry}><div className="grid gap-4 sm:grid-cols-2"><Field label="Request count"><input type="number" min="1" value={requestCount} onChange={(event) => setRequestCount(event.target.value)} className={inputClass} /></Field><Field label="Error count"><input type="number" min="0" value={errorCount} onChange={(event) => setErrorCount(event.target.value)} className={inputClass} /></Field><Field label="Average latency ms"><input type="number" min="0" value={averageLatency} onChange={(event) => setAverageLatency(event.target.value)} className={inputClass} /></Field><Field label="P95 latency ms"><input type="number" min="0" value={p95Latency} onChange={(event) => setP95Latency(event.target.value)} className={inputClass} /></Field></div></Dialog>}
    {maintenanceOpen && <Dialog title="Schedule maintenance" description="Immediately remove this provider from routing until maintenance ends and fresh evidence arrives." submitLabel="Enable maintenance" canSubmit={Boolean(maintenanceUntil && maintenanceReason.trim())} onClose={() => setMaintenanceOpen(false)} onSubmit={(event) => changeMaintenance(true, event)}><Field label="Maintenance until"><input type="datetime-local" value={maintenanceUntil} onChange={(event) => setMaintenanceUntil(event.target.value)} className={inputClass} /></Field><Field label="Reason"><input value={maintenanceReason} onChange={(event) => setMaintenanceReason(event.target.value)} className={inputClass} /></Field></Dialog>}
  </div>;
}
