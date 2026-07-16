"use client";

import { Button, Card, Table } from "@heroui/react";
import {
  CheckCircle2,
  ChevronRight,
  Clock3,
  Copy,
  Download,
  Filter,
  Search,
  ServerCog,
  ShieldAlert,
  Sparkles,
  X,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { listRequests as listRequestsRequest } from "@/lib/control-plane";
import { requests as seedRequests } from "@/lib/mock-data";
import type { LlmRequest, RequestStatus } from "@/types/observability";

type StatusFilter = "all" | RequestStatus;

const statusMeta: Record<RequestStatus, { label: string; className: string }> = {
  success: { label: "Success", className: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20" },
  error: { label: "Error", className: "bg-rose-400/10 text-rose-300 ring-rose-400/20" },
  rate_limited: { label: "Rate limited", className: "bg-amber-400/10 text-amber-300 ring-amber-400/20" },
};

function StatusBadge({ status }: { status: RequestStatus }) {
  const meta = statusMeta[status];
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] font-medium ring-1 ring-inset ${meta.className}`}>{meta.label}</span>;
}

function formatDate(timestamp: string) {
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(new Date(timestamp));
}

function RequestDetail({ request, onClose }: { request: LlmRequest; onClose: () => void }) {
  const [copied, setCopied] = useState(false);

  async function copyId() {
    await navigator.clipboard.writeText(request.id);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  }

  return (
    <aside className="glass-panel sticky top-[76px] flex max-h-[calc(100vh-100px)] min-h-[620px] flex-col overflow-hidden 2xl:min-h-0">
      <div className="flex items-start justify-between border-b border-white/[0.06] p-4">
        <div className="min-w-0">
          <div className="mb-2 flex items-center gap-2">
            <StatusBadge status={request.status} />
            {request.cached && <span className="rounded-full bg-blue-400/10 px-2 py-1 text-[10px] font-medium text-blue-300 ring-1 ring-inset ring-blue-400/15">Cached</span>}
          </div>
          <p className="truncate font-mono text-xs text-slate-300">{request.id}</p>
          <p className="mt-1 text-[10px] text-slate-600">{formatDate(request.timestamp)}</p>
        </div>
        <button type="button" onClick={onClose} className="rounded-lg p-2 text-slate-500 transition hover:bg-white/5 hover:text-white" aria-label="Close request detail">
          <X size={15} />
        </button>
      </div>

      <div className="flex-1 space-y-5 overflow-y-auto p-4">
        <section className="grid grid-cols-2 gap-2">
          {[
            ["Model", request.model],
            ["Provider", request.provider],
            ["Latency", `${request.latencyMs.toLocaleString()} ms`],
            ["Cost", `$${request.costUsd.toFixed(4)}`],
            ["Input", `${request.inputTokens.toLocaleString()} tokens`],
            ["Output", `${request.outputTokens.toLocaleString()} tokens`],
          ].map(([label, value]) => (
            <div key={label} className="rounded-xl border border-white/[0.055] bg-white/[0.02] p-3">
              <p className="text-[9px] font-medium tracking-wider text-slate-600 uppercase">{label}</p>
              <p className="mt-1.5 truncate text-[11px] text-slate-300" title={value}>{value}</p>
            </div>
          ))}
        </section>

        <section>
          <p className="mb-2 text-[10px] font-medium tracking-wider text-slate-500 uppercase">Prompt</p>
          <div className="rounded-xl border border-white/[0.06] bg-black/20 p-3 text-xs leading-5 text-slate-300">{request.prompt}</div>
        </section>

        <section>
          <p className="mb-2 text-[10px] font-medium tracking-wider text-slate-500 uppercase">Response</p>
          <div className={`rounded-xl border p-3 text-xs leading-5 ${request.status === "success" ? "border-white/[0.06] bg-black/20 text-slate-300" : "border-rose-400/15 bg-rose-400/[0.045] text-rose-200"}`}>
            {request.response}
          </div>
        </section>

        <section>
          <p className="mb-2 text-[10px] font-medium tracking-wider text-slate-500 uppercase">Context</p>
          <dl className="space-y-2 rounded-xl border border-white/[0.055] bg-white/[0.015] p-3 text-[11px]">
            <div className="flex justify-between gap-4"><dt className="text-slate-600">Project</dt><dd className="truncate text-slate-300">{request.project}</dd></div>
            <div className="flex justify-between gap-4"><dt className="text-slate-600">User</dt><dd className="truncate text-slate-300">{request.user}</dd></div>
            <div className="flex justify-between gap-4"><dt className="text-slate-600">Cache</dt><dd className="text-slate-300">{request.cached ? "Hit" : "Miss"}</dd></div>
          </dl>
        </section>
      </div>

      <div className="flex gap-2 border-t border-white/[0.06] p-4">
        <Button onPress={copyId} variant="tertiary" className="h-9 flex-1 gap-2 border border-white/8 bg-white/[0.025] text-xs text-slate-300">
          {copied ? <CheckCircle2 size={13} /> : <Copy size={13} />} {copied ? "Copied" : "Copy ID"}
        </Button>
        <Button className="h-9 flex-1 gap-2 bg-blue-500 text-xs text-white"><Sparkles size={13} /> Open trace</Button>
      </div>
    </aside>
  );
}

export function RequestExplorer() {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [project, setProject] = useState("all");
  const [requestRows, setRequestRows] = useState(seedRequests);
  const [selectedId, setSelectedId] = useState<string | null>(seedRequests[0]?.id ?? null);
  const [apiState, setAPIState] = useState<"connecting" | "connected" | "fallback">("connecting");
  const requests = requestRows;
  void apiState;

  useEffect(() => {
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      listRequestsRequest({ q: query, status, project }, controller.signal)
        .then(({ data }) => {
          setRequestRows(data);
          setSelectedId((current) => data.some((item) => item.id === current) ? current : (data[0]?.id ?? null));
          setAPIState("connected");
        })
        .catch((error: unknown) => {
          if (error instanceof DOMException && error.name === "AbortError") return;
          setAPIState("fallback");
        });
    }, 180);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [project, query, status]);

  const projects = useMemo(() => Array.from(new Set(requestRows.map((request) => request.project))).sort(), [requestRows]);
  const filteredRequests = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return requestRows.filter((request) => {
      const matchesQuery = !normalized || [request.id, request.model, request.provider, request.project, request.user, request.prompt].some((value) => value.toLowerCase().includes(normalized));
      return matchesQuery && (status === "all" || request.status === status) && (project === "all" || request.project === project);
    });
  }, [project, query, requestRows, status]);

  const selectedRequest = requestRows.find((request) => request.id === selectedId) ?? null;
  const activeFilterCount = Number(status !== "all") + Number(project !== "all") + Number(Boolean(query.trim()));

  function clearFilters() {
    setQuery("");
    setStatus("all");
    setProject("all");
  }

  return (
    <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2 text-[11px] font-medium tracking-[0.12em] text-blue-300/80 uppercase"><ServerCog size={13} /> Observability</div>
          <h1 className="text-2xl font-semibold tracking-[-0.035em] text-white sm:text-3xl">Requests</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">Search and debug model traffic across projects, providers, users, and gateway policies.</p>
        </div>
        <div className="flex gap-2">
          <Button variant="tertiary" className="h-9 gap-2 border border-white/9 bg-white/[0.025] px-3 text-xs text-slate-300"><Download size={14} /> Export</Button>
          <Button className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Sparkles size={14} /> Saved views</Button>
        </div>
      </section>

      <Card className="glass-panel px-0 py-0">
        <Card.Content className="p-3">
          <div className="flex flex-col gap-2 lg:flex-row lg:items-center">
            <label className="relative min-w-0 flex-1">
              <span className="sr-only">Search requests</span>
              <Search className="absolute top-1/2 left-3 -translate-y-1/2 text-slate-600" size={14} />
              <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search request ID, model, user, project, or prompt..." className="h-10 w-full rounded-xl border border-white/8 bg-black/20 pr-3 pl-9 text-xs text-slate-200 outline-none transition placeholder:text-slate-650 focus:border-blue-400/40 focus:ring-2 focus:ring-blue-400/10" />
            </label>
            <div className="flex flex-wrap items-center gap-2">
              <label className="relative">
                <span className="sr-only">Filter by status</span>
                <select value={status} onChange={(event) => setStatus(event.target.value as StatusFilter)} className="h-10 appearance-none rounded-xl border border-white/8 bg-[#0a0f18] pr-8 pl-3 text-xs text-slate-300 outline-none focus:border-blue-400/40">
                  <option value="all">All statuses</option><option value="success">Success</option><option value="error">Error</option><option value="rate_limited">Rate limited</option>
                </select>
                <Filter className="pointer-events-none absolute top-1/2 right-2.5 -translate-y-1/2 text-slate-600" size={12} />
              </label>
              <label className="relative min-w-44 flex-1 sm:flex-none">
                <span className="sr-only">Filter by project</span>
                <select value={project} onChange={(event) => setProject(event.target.value)} className="h-10 w-full appearance-none rounded-xl border border-white/8 bg-[#0a0f18] pr-8 pl-3 text-xs text-slate-300 outline-none focus:border-blue-400/40">
                  <option value="all">All projects</option>{projects.map((item) => <option key={item} value={item}>{item}</option>)}
                </select>
                <ChevronRight className="pointer-events-none absolute top-1/2 right-2.5 -translate-y-1/2 rotate-90 text-slate-600" size={12} />
              </label>
              {activeFilterCount > 0 && <button type="button" onClick={clearFilters} className="h-10 px-2 text-[11px] text-slate-500 transition hover:text-white">Clear {activeFilterCount}</button>}
            </div>
          </div>
        </Card.Content>
      </Card>

      <section className={`grid min-w-0 gap-4 ${selectedRequest ? "2xl:grid-cols-[minmax(0,1fr)_390px]" : "grid-cols-1"}`}>
        <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0">
          <Card.Header className="flex-row items-center justify-between border-b border-white/[0.055] p-4">
            <div><Card.Title className="text-sm font-medium text-slate-100">Request stream</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">{filteredRequests.length} results in this development dataset</Card.Description></div>
            <div className="flex items-center gap-1.5 text-[10px] text-emerald-300"><span className="size-1.5 animate-pulse rounded-full bg-emerald-300" /> Live</div>
          </Card.Header>
          <Card.Content className="p-0">
            <Table className="rounded-none border-0 bg-transparent">
              <Table.ScrollContainer>
                <Table.Content aria-label="LLM request explorer" className="min-w-[1020px]">
                  <Table.Header>
                    {[["time", "Time"], ["model", "Model / Provider"], ["context", "Project / User"], ["status", "Status"], ["tokens", "Tokens"], ["latency", "Latency"], ["cost", "Cost"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} isRowHeader={id === "time"} className={`bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase ${["tokens", "latency", "cost"].includes(id) ? "text-right" : ""}`}>{label}</Table.Column>)}
                  </Table.Header>
                  <Table.Body>
                    {filteredRequests.map((request) => (
                      <Table.Row key={request.id} id={request.id} onClick={() => setSelectedId(request.id)} className={`cursor-pointer border-t border-white/[0.045] transition hover:bg-white/[0.03] ${selectedId === request.id ? "bg-blue-400/[0.055]" : ""}`}>
                        <Table.Cell className="font-mono text-[10px] text-slate-500">{formatDate(request.timestamp)}</Table.Cell>
                        <Table.Cell><p className="text-xs font-medium text-slate-200">{request.model}</p><p className="text-[10px] text-slate-600">{request.provider}</p></Table.Cell>
                        <Table.Cell><p className="max-w-48 truncate text-xs text-slate-300">{request.project}</p><p className="max-w-48 truncate text-[10px] text-slate-600">{request.user}</p></Table.Cell>
                        <Table.Cell><StatusBadge status={request.status} /></Table.Cell>
                        <Table.Cell className="text-right font-mono text-[11px] text-slate-400">{(request.inputTokens + request.outputTokens).toLocaleString()}</Table.Cell>
                        <Table.Cell className="text-right font-mono text-[11px] text-slate-400">{request.latencyMs.toLocaleString()}ms</Table.Cell>
                        <Table.Cell className="text-right font-mono text-[11px] text-slate-300">${request.costUsd.toFixed(4)}</Table.Cell>
                        <Table.Cell className="text-right"><ChevronRight className="ml-auto text-slate-700" size={14} /></Table.Cell>
                      </Table.Row>
                    ))}
                  </Table.Body>
                </Table.Content>
              </Table.ScrollContainer>
            </Table>
            {filteredRequests.length === 0 && <div className="flex min-h-64 flex-col items-center justify-center gap-3 border-t border-white/[0.045] text-center"><ShieldAlert className="text-slate-700" size={24} /><div><p className="text-sm text-slate-300">No matching requests</p><p className="mt-1 text-xs text-slate-600">Adjust or clear the current filters.</p></div><Button onPress={clearFilters} variant="tertiary" className="h-8 border border-white/8 bg-white/[0.025] text-xs text-slate-300">Clear filters</Button></div>}
          </Card.Content>
          <Card.Footer className="flex-row justify-between border-t border-white/[0.055] p-3 text-[10px] text-slate-600"><span>Showing {filteredRequests.length} of {requests.length}</span><span className="flex items-center gap-1"><Clock3 size={11} /> Refreshed just now</span></Card.Footer>
        </Card>

        {selectedRequest && <RequestDetail request={selectedRequest} onClose={() => setSelectedId(null)} />}
      </section>
    </div>
  );
}
