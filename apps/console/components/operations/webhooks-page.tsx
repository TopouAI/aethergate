"use client";

import { Button, Card, Table } from "@heroui/react";
import { Check, ChevronRight, Clock3, Copy, FlaskConical, History, Pause, Play, Plus, RefreshCcw, ShieldCheck, Webhook } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Dialog, Field, inputClass, MetricGrid, PageHeader, SearchBar, selectClass, StateBadge, type APIState } from "@/components/foundation/resource-ui";
import {
  createWebhook, disableWebhook, enableWebhook, listWebhookDeliveries, listWebhooks,
  queueWebhookTest, replayWebhookDelivery, retryWebhookDelivery,
} from "@/lib/control-plane";
import { foundationOrganizationId } from "@/lib/foundation-data";
import { seedWebhookDeliveries, seedWebhooks, webhookEventOptions } from "@/lib/webhook-data";
import type { WebhookDelivery, WebhookEndpoint, WebhookEventType, WebhookStatus } from "@/types/webhook";

export function WebhooksPage() {
  const [webhooks, setWebhooks] = useState<WebhookEndpoint[]>(seedWebhooks);
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>(seedWebhookDeliveries);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"all" | WebhookStatus>("all");
  const [selectedId, setSelectedId] = useState(seedWebhooks[0]?.id ?? "");
  const [apiState, setAPIState] = useState<APIState>("connecting");
  const [notice, setNotice] = useState("");
  const [open, setOpen] = useState(false);
  const [revealedSecret, setRevealedSecret] = useState("");

  const [name, setName] = useState("");
  const [destination, setDestination] = useState("https://");
  const [events, setEvents] = useState<WebhookEventType[]>(["request.completed", "request.failed"]);
  const [sampleRate, setSampleRate] = useState("100");
  const [includeData, setIncludeData] = useState(true);
  const [filterKey, setFilterKey] = useState("environment");
  const [filterValue, setFilterValue] = useState("production");
  const [maxAttempts, setMaxAttempts] = useState("5");
  const [timeoutSeconds, setTimeoutSeconds] = useState("10");

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      listWebhooks({ organizationId: foundationOrganizationId }, controller.signal),
      listWebhookDeliveries({ organizationId: foundationOrganizationId }, controller.signal),
    ]).then(([endpointResponse, deliveryResponse]) => {
      setWebhooks(endpointResponse.data);
      setDeliveries(deliveryResponse.data);
      setSelectedId((current) => endpointResponse.data.some((item) => item.id === current) ? current : endpointResponse.data[0]?.id ?? "");
      setAPIState("connected");
    }).catch((error: unknown) => {
      if (!(error instanceof DOMException && error.name === "AbortError")) setAPIState("fallback");
    });
    return () => controller.abort();
  }, []);

  const filtered = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return webhooks.filter((endpoint) => (status === "all" || endpoint.status === status)
      && (!normalized || [endpoint.name, endpoint.destination, ...endpoint.events].some((value) => value.toLowerCase().includes(normalized))));
  }, [query, status, webhooks]);
  const selected = webhooks.find((endpoint) => endpoint.id === selectedId) ?? null;
  const selectedDeliveries = deliveries.filter((delivery) => !selected || delivery.webhookId === selected.id);

  function toggleEvent(event: WebhookEventType) {
    setEvents((current) => current.includes(event) ? current.filter((item) => item !== event) : [...current, event]);
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAPIState("saving");
    setNotice("");
    try {
      const response = await createWebhook({
        organizationId: foundationOrganizationId, name, destination, events, sampleRate: Number(sampleRate), includeData,
        propertyFilters: filterKey.trim() && filterValue.trim() ? [{ key: filterKey, value: filterValue }] : [],
        maxAttempts: Number(maxAttempts), timeoutSeconds: Number(timeoutSeconds),
      });
      setWebhooks((current) => [response.data, ...current]);
      setSelectedId(response.data.id);
      setRevealedSecret(response.signingSecret);
      setOpen(false);
      setName("");
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Webhook could not be created.");
      setAPIState("fallback");
    }
  }

  async function changeStatus(enabled: boolean) {
    if (!selected) return;
    setAPIState("saving");
    try {
      const response = enabled ? await enableWebhook(selected.id, foundationOrganizationId) : await disableWebhook(selected.id, foundationOrganizationId);
      setWebhooks((current) => current.map((item) => item.id === response.data.id ? response.data : item));
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Webhook state could not be changed.");
      setAPIState("fallback");
    }
  }

  async function queueTest() {
    if (!selected) return;
    try {
      const response = await queueWebhookTest(selected.id, foundationOrganizationId, selected.events[0]);
      setDeliveries((current) => [response.data, ...current]);
      setNotice("Test delivery accepted by the webhook worker queue.");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Test delivery could not be queued.");
    }
  }

  async function actOnDelivery(delivery: WebhookDelivery, action: "retry" | "replay") {
    try {
      const response = action === "retry"
        ? await retryWebhookDelivery(delivery.id, foundationOrganizationId)
        : await replayWebhookDelivery(delivery.id, foundationOrganizationId);
      setDeliveries((current) => [response.data, ...current]);
      setNotice(`${action === "retry" ? "Retry" : "Replay"} accepted by the webhook worker queue.`);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Delivery action could not be queued.");
    }
  }

  async function copySecret() {
    await navigator.clipboard.writeText(revealedSecret);
    setNotice("Signing secret copied. Store it now; it will not be shown again.");
  }

  return <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
    <PageHeader eyebrow="Event integrations" title="Webhooks" description="Subscribe enterprise systems to signed request, alert, budget, key, and provider events with delivery evidence and replay controls." icon={Webhook} apiState={apiState} action={<Button onPress={() => setOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={14} />Add webhook</Button>} />

    {notice && <div className="rounded-xl border border-amber-400/15 bg-amber-400/[0.04] px-4 py-3 text-xs text-amber-200">{notice}</div>}
    {revealedSecret && <section className="rounded-2xl border border-blue-400/20 bg-blue-400/[0.045] p-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between"><div><p className="flex items-center gap-2 text-xs font-medium text-blue-200"><ShieldCheck size={14} />Signing secret — shown once</p><p className="mt-2 break-all font-mono text-xs text-slate-300">{revealedSecret}</p><p className="mt-1 text-[10px] text-slate-600">Use HMAC-SHA256 verification and reject stale event timestamps or duplicate event IDs.</p></div><div className="flex gap-2"><Button onPress={copySecret} variant="tertiary" className="h-9 gap-2 border border-white/8 text-xs text-slate-300"><Copy size={13} />Copy</Button><Button onPress={() => setRevealedSecret("")} className="h-9 bg-blue-500 text-xs text-white">I stored it</Button></div></div>
    </section>}

    <MetricGrid items={[
      { label: "Endpoints", value: webhooks.length.toString(), hint: `${webhooks.filter((item) => item.status === "active").length} active`, icon: Webhook },
      { label: "Succeeded", value: deliveries.filter((item) => item.status === "succeeded").length.toString(), hint: "Visible delivery evidence", icon: Check },
      { label: "Needs retry", value: deliveries.filter((item) => item.status === "failed").length.toString(), hint: "Within retry policy", icon: RefreshCcw },
      { label: "Dead letter", value: deliveries.filter((item) => item.status === "dead_letter").length.toString(), hint: "Manual replay available", icon: History },
    ]} />

    <SearchBar value={query} onChange={setQuery} placeholder="Search webhook, URL, or subscribed event..." trailing={<select value={status} onChange={(event) => setStatus(event.target.value as "all" | WebhookStatus)} className={selectClass}><option value="all">All states</option><option value="active">Active</option><option value="disabled">Disabled</option></select>} />

    <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_390px]" : "grid-cols-1"}`}>
      <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">Webhook endpoints</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">{filtered.length} destinations · secrets never return in list responses</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Webhook registry" className="min-w-[1000px]"><Table.Header>{[["name", "Endpoint"], ["status", "Status"], ["events", "Subscriptions"], ["sample", "Sample"], ["delivery", "Delivery"], ["secret", "Signing key"], ["updated", "Last delivered"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((endpoint) => <Table.Row key={endpoint.id} id={endpoint.id} onClick={() => setSelectedId(endpoint.id)} className={`cursor-pointer border-t border-white/[0.045] hover:bg-white/[0.03] ${selectedId === endpoint.id ? "bg-blue-400/[0.055]" : ""}`}><Table.Cell><p className="text-xs font-medium text-slate-200">{endpoint.name}</p><p className="max-w-72 truncate font-mono text-[10px] text-slate-600">{endpoint.destination}</p></Table.Cell><Table.Cell><StateBadge value={endpoint.status} /></Table.Cell><Table.Cell><p className="text-xs text-slate-300">{endpoint.events.length} events</p><p className="max-w-56 truncate text-[10px] text-slate-600">{endpoint.events.join(", ")}</p></Table.Cell><Table.Cell className="font-mono text-xs text-slate-400">{endpoint.sampleRate}%</Table.Cell><Table.Cell><p className="font-mono text-xs text-emerald-300">{endpoint.successCount} ok</p><p className="font-mono text-[10px] text-rose-300">{endpoint.failureCount} failed</p></Table.Cell><Table.Cell className="font-mono text-[10px] text-slate-500">{endpoint.signingSecretPrefix}••••</Table.Cell><Table.Cell className="text-[11px] text-slate-500">{endpoint.lastDeliveredAt ? new Date(endpoint.lastDeliveredAt).toLocaleString("zh-CN") : "Never"}</Table.Cell><Table.Cell><ChevronRight size={14} className="ml-auto text-slate-700" /></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>

      {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex justify-between"><span className="rounded-xl bg-blue-400/10 p-2 text-blue-300"><Webhook size={17} /></span><StateBadge value={selected.status} /></div><h2 className="mt-4 text-base font-semibold text-white">{selected.name}</h2><p className="mt-1 break-all font-mono text-[10px] text-slate-600">{selected.destination}</p></div><div className="space-y-5 p-5"><section><p className="mb-2 text-[10px] font-medium tracking-wider text-slate-500 uppercase">Subscribed events</p><div className="flex flex-wrap gap-1.5">{selected.events.map((event) => <span key={event} className="rounded-md bg-blue-400/[0.07] px-2 py-1 font-mono text-[9px] text-blue-200 ring-1 ring-inset ring-blue-400/10">{event}</span>)}</div></section><dl className="space-y-3 text-[11px]">{[["Version", selected.version], ["Sample rate", `${selected.sampleRate}%`], ["Enhanced data", selected.includeData ? "Included" : "Excluded"], ["Attempts", selected.maxAttempts], ["Timeout", `${selected.timeoutSeconds}s`], ["Filters", selected.propertyFilters.map((item) => `${item.key}=${item.value}`).join(", ") || "None"]].map(([label, value]) => <div key={label} className="flex justify-between gap-4"><dt className="text-slate-600">{label}</dt><dd className="max-w-56 truncate text-right text-slate-300">{value}</dd></div>)}</dl><div className="grid grid-cols-2 gap-2"><Button onPress={queueTest} isDisabled={selected.status !== "active"} variant="tertiary" className="h-9 gap-2 border border-white/8 text-xs text-slate-300"><FlaskConical size={13} />Queue test</Button>{selected.status === "active" ? <Button onPress={() => changeStatus(false)} variant="tertiary" className="h-9 gap-2 border border-amber-400/15 text-xs text-amber-200"><Pause size={13} />Disable</Button> : <Button onPress={() => changeStatus(true)} className="h-9 gap-2 bg-blue-500 text-xs text-white"><Play size={13} />Enable</Button>}</div></div></aside>}
    </section>

    <Card className="glass-panel px-0 py-0"><Card.Header className="p-4"><Card.Title className="text-sm text-slate-100">Delivery history</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Idempotent event IDs, attempt evidence, retry schedule, and manual replay lineage.</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Webhook delivery history" className="min-w-[1080px]"><Table.Header>{[["event", "Event"], ["endpoint", "Endpoint"], ["status", "Status"], ["attempt", "Attempt"], ["response", "Response"], ["duration", "Duration"], ["time", "Created"], ["action", "Action"]].map(([id, label]) => <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{selectedDeliveries.map((delivery) => <Table.Row key={delivery.id} id={delivery.id} className="border-t border-white/[0.045]"><Table.Cell><p className="font-mono text-[11px] text-slate-300">{delivery.eventType}</p><p className="font-mono text-[9px] text-slate-700">{delivery.eventId}</p></Table.Cell><Table.Cell><p className="text-xs text-slate-400">{delivery.webhookName}</p><p className="text-[9px] uppercase text-slate-700">{delivery.trigger}</p></Table.Cell><Table.Cell><StateBadge value={delivery.status} /></Table.Cell><Table.Cell className="font-mono text-xs text-slate-400">{delivery.attempt}/{delivery.maxAttempts}</Table.Cell><Table.Cell><p className="font-mono text-xs text-slate-300">{delivery.responseStatus ?? "—"}</p><p className="max-w-64 truncate text-[10px] text-rose-300">{delivery.errorMessage}</p></Table.Cell><Table.Cell className="font-mono text-[11px] text-slate-500">{delivery.durationMs ? `${delivery.durationMs}ms` : "queued"}</Table.Cell><Table.Cell className="text-[11px] text-slate-500">{new Date(delivery.createdAt).toLocaleString("zh-CN")}</Table.Cell><Table.Cell>{delivery.status === "failed" && delivery.attempt < delivery.maxAttempts ? <Button onPress={() => actOnDelivery(delivery, "retry")} variant="tertiary" className="h-8 gap-1.5 border border-white/8 text-[10px] text-slate-300"><RefreshCcw size={11} />Retry</Button> : delivery.status !== "pending" && delivery.status !== "delivering" ? <Button onPress={() => actOnDelivery(delivery, "replay")} variant="tertiary" className="h-8 gap-1.5 border border-white/8 text-[10px] text-slate-300"><History size={11} />Replay</Button> : <span className="flex items-center gap-1 text-[10px] text-slate-600"><Clock3 size={11} />Queued</span>}</Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>

    {open && <Dialog title="Add webhook" description="Subscribe an HTTPS endpoint to signed AetherGate events." submitLabel="Create endpoint" canSubmit={Boolean(name.trim() && destination.trim() && events.length && Number(sampleRate) > 0)} onClose={() => setOpen(false)} onSubmit={submit}>
      <Field label="Webhook name"><input autoFocus value={name} onChange={(event) => setName(event.target.value)} className={inputClass} placeholder="Production event bus" /></Field>
      <Field label="Destination URL"><input value={destination} onChange={(event) => setDestination(event.target.value)} className={inputClass} placeholder="https://events.example.com/aethergate" /></Field>
      <Field label="Event subscriptions"><div className="grid gap-2 sm:grid-cols-2">{webhookEventOptions.map((option) => <button key={option.value} type="button" onClick={() => toggleEvent(option.value)} className={`flex items-center gap-2 rounded-xl border px-3 py-2 text-left text-[11px] ${events.includes(option.value) ? "border-blue-400/25 bg-blue-400/[0.07] text-blue-200" : "border-white/8 bg-black/15 text-slate-500"}`}><span className={`grid h-4 w-4 place-items-center rounded border ${events.includes(option.value) ? "border-blue-400/40 bg-blue-500/20" : "border-white/10"}`}>{events.includes(option.value) && <Check size={10} />}</span>{option.label}</button>)}</div></Field>
      <div className="grid gap-4 sm:grid-cols-3"><Field label="Sample rate %"><input type="number" min="0.1" max="100" step="0.1" value={sampleRate} onChange={(event) => setSampleRate(event.target.value)} className={inputClass} /></Field><Field label="Max attempts"><input type="number" min="1" max="10" value={maxAttempts} onChange={(event) => setMaxAttempts(event.target.value)} className={inputClass} /></Field><Field label="Timeout seconds"><input type="number" min="1" max="30" value={timeoutSeconds} onChange={(event) => setTimeoutSeconds(event.target.value)} className={inputClass} /></Field></div>
      <label className="flex items-center justify-between rounded-xl border border-white/8 bg-black/15 p-3"><span><span className="block text-[11px] font-medium text-slate-300">Include enhanced data</span><span className="mt-0.5 block text-[10px] text-slate-600">Cost, tokens, latency, response metadata, and artifact URLs.</span></span><input type="checkbox" checked={includeData} onChange={(event) => setIncludeData(event.target.checked)} className="h-4 w-4 accent-blue-500" /></label>
      <div className="grid gap-4 sm:grid-cols-2"><Field label="Property filter key"><input value={filterKey} onChange={(event) => setFilterKey(event.target.value)} className={inputClass} /></Field><Field label="Property filter value"><input value={filterValue} onChange={(event) => setFilterValue(event.target.value)} className={inputClass} /></Field></div>
    </Dialog>}
  </div>;
}
