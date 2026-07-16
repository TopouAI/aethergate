"use client";

import { Button, Card, Table } from "@heroui/react";
import { Box, Boxes, ChevronRight, Layers3, Plus, ServerCog } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Dialog, Field, inputClass, MetricGrid, PageHeader, SearchBar, selectClass, StateBadge, type APIState } from "@/components/foundation/resource-ui";
import { createWorkspace, listWorkspaces } from "@/lib/control-plane";
import { foundationOrganizationId, seedWorkspaces } from "@/lib/foundation-data";
import type { Workspace, WorkspaceEnvironment } from "@/types/foundation";

function slugify(value: string) { return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, ""); }

export function WorkspacesPage() {
  const [items, setItems] = useState(seedWorkspaces);
  const [query, setQuery] = useState("");
  const [environment, setEnvironment] = useState<"all" | WorkspaceEnvironment>("all");
  const [selectedId, setSelectedId] = useState(seedWorkspaces[0]?.id ?? "");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [name, setName] = useState("");
  const [newEnvironment, setNewEnvironment] = useState<WorkspaceEnvironment>("production");
  const [apiState, setAPIState] = useState<APIState>("connecting");

  useEffect(() => {
    const controller = new AbortController();
    listWorkspaces(foundationOrganizationId, controller.signal).then(({ data }) => { setItems(data); setSelectedId((current) => data.some((item) => item.id === current) ? current : data[0]?.id ?? ""); setAPIState("connected"); }).catch((error: unknown) => { if (!(error instanceof DOMException && error.name === "AbortError")) setAPIState("fallback"); });
    return () => controller.abort();
  }, []);

  const filtered = useMemo(() => { const needle = query.trim().toLowerCase(); return items.filter((item) => (environment === "all" || item.environment === environment) && (!needle || [item.name, item.slug, item.environment].some((value) => value.toLowerCase().includes(needle)))); }, [environment, items, query]);
  const selected = items.find((item) => item.id === selectedId) ?? null;
  const projectCount = items.reduce((sum, item) => sum + item.projects, 0);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    setAPIState("saving");
    try {
      const { data } = await createWorkspace({ organizationId: foundationOrganizationId, name: trimmed, slug: slugify(trimmed), environment: newEnvironment });
      setItems((current) => [data, ...current]); setSelectedId(data.id); setAPIState("connected");
    } catch {
      const fallback: Workspace = { id: `ws_${crypto.randomUUID().slice(0, 8)}`, organizationId: foundationOrganizationId, name: trimmed, slug: slugify(trimmed), status: "active", environment: newEnvironment, projects: 0, createdAt: new Date().toISOString() };
      setItems((current) => [fallback, ...current]); setSelectedId(fallback.id); setAPIState("fallback");
    }
    setName(""); setDialogOpen(false);
  }

  return <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
    <PageHeader eyebrow="Enterprise control plane" title="Workspaces" description="Partition teams and environments while keeping projects, budgets, model access, and policy inheritance in one boundary." icon={Box} apiState={apiState} action={<Button onPress={() => setDialogOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={14} /> Create workspace</Button>} />
    <MetricGrid items={[{ label: "Workspaces", value: items.length.toString(), hint: "Current organization", icon: Boxes }, { label: "Projects", value: projectCount.toString(), hint: "Across workspace scopes", icon: Layers3 }, { label: "Production", value: items.filter((item) => item.environment === "production").length.toString(), hint: "Protected environments", icon: ServerCog }, { label: "Shared", value: items.filter((item) => item.environment === "shared").length.toString(), hint: "Cross-team resources", icon: Box }]} />
    <SearchBar value={query} onChange={setQuery} placeholder="Search workspace name, slug, or environment..." trailing={<select value={environment} onChange={(event) => setEnvironment(event.target.value as "all" | WorkspaceEnvironment)} className={selectClass}><option value="all">All environments</option><option value="development">Development</option><option value="staging">Staging</option><option value="production">Production</option><option value="shared">Shared</option></select>} />
    <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_340px]" : "grid-cols-1"}`}>
      <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm font-medium text-slate-100">Workspace registry</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">{filtered.length} visible boundaries</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Workspace registry" className="min-w-[760px]"><Table.Header>{[["name", "Workspace"], ["environment", "Environment"], ["status", "Status"], ["projects", "Projects"], ["created", "Created"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} isRowHeader={id === "name"} className="bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase">{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((item) => <Table.Row key={item.id} id={item.id} onClick={() => setSelectedId(item.id)} className={`cursor-pointer border-t border-white/[0.045] transition hover:bg-white/[0.03] ${selectedId === item.id ? "bg-blue-400/[0.055]" : ""}`}><Table.Cell><p className="text-xs font-medium text-slate-200">{item.name}</p><p className="mt-0.5 font-mono text-[10px] text-slate-600">{item.slug}</p></Table.Cell><Table.Cell><StateBadge value={item.environment} /></Table.Cell><Table.Cell><StateBadge value={item.status} /></Table.Cell><Table.Cell className="font-mono text-xs text-slate-300">{item.projects}</Table.Cell><Table.Cell className="text-[11px] text-slate-500">{new Date(item.createdAt).toLocaleDateString("zh-CN")}</Table.Cell><Table.Cell><ChevronRight className="ml-auto text-slate-700" size={14} /></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>
      {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex items-center justify-between"><span className="rounded-xl bg-blue-400/10 p-2 text-blue-300"><Box size={17} /></span><StateBadge value={selected.environment} /></div><h2 className="mt-4 text-base font-semibold text-white">{selected.name}</h2><p className="mt-1 font-mono text-[10px] text-slate-600">{selected.id}</p></div><div className="space-y-5 p-5"><div className="grid grid-cols-2 gap-2">{[["Projects", selected.projects], ["Status", selected.status]].map(([label, value]) => <div key={label} className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><p className="text-[9px] tracking-wider text-slate-600 uppercase">{label}</p><p className="mt-1.5 text-sm font-medium capitalize text-slate-200">{value}</p></div>)}</div><dl className="space-y-3 text-[11px]">{[["Organization", selected.organizationId], ["Slug", selected.slug], ["Created", new Date(selected.createdAt).toLocaleString("zh-CN")]].map(([label, value]) => <div key={label} className="flex justify-between gap-4"><dt className="text-slate-600">{label}</dt><dd className="truncate text-slate-300">{value}</dd></div>)}</dl><div className="rounded-xl border border-blue-400/10 bg-blue-400/[0.035] p-3 text-[11px] leading-5 text-blue-100/70">Projects created here inherit organization roles, model policy, residency, and budget controls.</div></div></aside>}
    </section>
    {dialogOpen && <Dialog title="Create workspace" description="Create an environment and policy boundary in TopoAI." submitLabel="Create workspace" canSubmit={Boolean(name.trim())} onClose={() => setDialogOpen(false)} onSubmit={submit}><Field label="Workspace name"><input autoFocus value={name} onChange={(event) => setName(event.target.value)} placeholder="Platform Engineering" className={inputClass} /></Field><Field label="Environment"><select value={newEnvironment} onChange={(event) => setNewEnvironment(event.target.value as WorkspaceEnvironment)} className={selectClass}><option value="development">Development</option><option value="staging">Staging</option><option value="production">Production</option><option value="shared">Shared</option></select></Field></Dialog>}
  </div>;
}
