"use client";

import { Button, Card, Table } from "@heroui/react";
import { Building2, Check, ChevronRight, CircleDollarSign, Globe2, Plus, Search, Users, X } from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";

import { createOrganization as createOrganizationRequest, listOrganizations } from "@/lib/control-plane";
import { organizations as seedOrganizations } from "@/lib/enterprise-data";
import type { Organization, OrganizationStatus } from "@/types/enterprise";

type StatusFilter = "all" | OrganizationStatus;

const statusStyles: Record<OrganizationStatus, string> = {
  active: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20",
  provisioning: "bg-blue-400/10 text-blue-300 ring-blue-400/20",
  suspended: "bg-rose-400/10 text-rose-300 ring-rose-400/20",
};

function OrganizationBadge({ status }: { status: OrganizationStatus }) {
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] font-medium capitalize ring-1 ring-inset ${statusStyles[status]}`}>{status}</span>;
}

function NewOrganizationDialog({ onClose, onCreate }: { onClose: () => void; onCreate: (organization: Organization) => void }) {
  const [name, setName] = useState("");
  const [region, setRegion] = useState("Singapore");
  const [plan, setPlan] = useState<Organization["plan"]>("Evaluation");

  function submit(event: FormEvent) {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    const slug = trimmed.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
    onCreate({
      id: `org_${crypto.randomUUID().slice(0, 8)}`,
      name: trimmed,
      slug: slug || `organization-${Date.now()}`,
      status: "provisioning",
      plan,
      region,
      workspaces: 0,
      projects: 0,
      members: 1,
      monthlyCostUsd: 0,
      budgetUsd: 0,
      requests: 0,
      owner: "holden@topoai.dev",
      createdAt: new Date().toISOString().slice(0, 10),
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#02040a]/80 p-4 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label="Create organization">
      <form onSubmit={submit} className="w-full max-w-lg rounded-2xl border border-white/10 bg-[#0a0f18] shadow-2xl shadow-black/60">
        <div className="flex items-start justify-between border-b border-white/[0.06] p-5">
          <div><h2 className="text-base font-semibold text-white">Create organization</h2><p className="mt-1 text-xs text-slate-500">Creates a local development record for the control-plane workflow.</p></div>
          <button type="button" onClick={onClose} className="rounded-lg p-2 text-slate-500 hover:bg-white/5 hover:text-white"><X size={15} /></button>
        </div>
        <div className="space-y-4 p-5">
          <label className="block"><span className="mb-1.5 block text-[11px] font-medium text-slate-400">Organization name</span><input autoFocus value={name} onChange={(event) => setName(event.target.value)} placeholder="Acme Corporation" className="h-10 w-full rounded-xl border border-white/8 bg-black/20 px-3 text-xs text-slate-200 outline-none focus:border-blue-400/40 focus:ring-2 focus:ring-blue-400/10" /></label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label><span className="mb-1.5 block text-[11px] font-medium text-slate-400">Data region</span><select value={region} onChange={(event) => setRegion(event.target.value)} className="h-10 w-full rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option>China East</option><option>Hong Kong</option><option>Singapore</option><option>US West</option><option>Europe West</option></select></label>
            <label><span className="mb-1.5 block text-[11px] font-medium text-slate-400">Plan</span><select value={plan} onChange={(event) => setPlan(event.target.value as Organization["plan"])} className="h-10 w-full rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option>Evaluation</option><option>Enterprise</option><option>Open Source</option></select></label>
          </div>
          <div className="rounded-xl border border-amber-400/12 bg-amber-400/[0.035] p-3 text-[11px] leading-5 text-amber-100/70">Persistence, domain claims, policy inheritance, and audit events will activate when the PostgreSQL organization API is connected.</div>
        </div>
        <div className="flex justify-end gap-2 border-t border-white/[0.06] p-4"><Button type="button" onPress={onClose} variant="tertiary" className="h-9 border border-white/8 bg-white/[0.025] text-xs text-slate-300">Cancel</Button><Button type="submit" isDisabled={!name.trim()} className="h-9 gap-2 bg-blue-500 text-xs text-white"><Plus size={13} /> Create</Button></div>
      </form>
    </div>
  );
}

export function OrganizationsPage() {
  const [organizations, setOrganizations] = useState(seedOrganizations);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<StatusFilter>("all");
  const [selectedId, setSelectedId] = useState(seedOrganizations[0]?.id ?? "");
  const [createOpen, setCreateOpen] = useState(false);
  const [apiState, setAPIState] = useState<"connecting" | "connected" | "fallback" | "saving">("connecting");
  void apiState;

  useEffect(() => {
    const controller = new AbortController();
    listOrganizations({}, controller.signal)
      .then(({ data }) => {
        setOrganizations(data);
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
    return organizations.filter((organization) => (status === "all" || organization.status === status) && (!normalized || [organization.name, organization.slug, organization.owner, organization.region].some((value) => value.toLowerCase().includes(normalized))));
  }, [organizations, query, status]);
  const selected = organizations.find((organization) => organization.id === selectedId) ?? null;
  const active = organizations.filter((organization) => organization.status === "active").length;
  const totalSpend = organizations.reduce((sum, organization) => sum + organization.monthlyCostUsd, 0);
  const totalMembers = organizations.reduce((sum, organization) => sum + organization.members, 0);

  function createOrganization(organization: Organization) {
    setOrganizations((current) => [organization, ...current]);
    setSelectedId(organization.id);
    setCreateOpen(false);
    setAPIState("saving");
    void createOrganizationRequest({
      name: organization.name,
      slug: organization.slug,
      plan: organization.plan,
      region: organization.region,
      owner: organization.owner,
    }).then(({ data }) => {
      setOrganizations((current) => current.map((item) => item.id === organization.id ? data : item));
      setSelectedId(data.id);
      setAPIState("connected");
    }).catch(() => setAPIState("fallback"));
  }

  return (
    <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div><div className="mb-2 flex items-center gap-2 text-[11px] font-medium tracking-[0.12em] text-blue-300/80 uppercase"><Building2 size={13} /> Enterprise control plane</div><h1 className="text-2xl font-semibold tracking-[-0.035em] text-white sm:text-3xl">Organizations</h1><p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">Manage tenant lifecycle, regional placement, ownership, plans, and inherited policy boundaries.</p></div>
        <Button onPress={() => setCreateOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={14} /> Create organization</Button>
      </section>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {[
          { label: "Organizations", value: organizations.length.toString(), hint: `${active} active tenants`, icon: Building2 },
          { label: "Members", value: totalMembers.toLocaleString(), hint: "Across all tenants", icon: Users },
          { label: "Monthly spend", value: `$${totalSpend.toLocaleString()}`, hint: "Development attribution", icon: CircleDollarSign },
          { label: "Data regions", value: new Set(organizations.map((item) => item.region)).size.toString(), hint: "Residency policy targets", icon: Globe2 },
        ].map((metric) => <Card key={metric.label} className="glass-panel px-0 py-0"><Card.Content className="flex items-start justify-between p-4"><div><p className="text-[10px] font-medium tracking-wider text-slate-600 uppercase">{metric.label}</p><p className="mt-2 text-xl font-semibold text-white">{metric.value}</p><p className="mt-1 text-[10px] text-slate-600">{metric.hint}</p></div><span className="rounded-lg bg-blue-400/8 p-2 text-blue-300 ring-1 ring-inset ring-blue-400/12"><metric.icon size={15} /></span></Card.Content></Card>)}
      </section>

      <Card className="glass-panel px-0 py-0"><Card.Content className="flex flex-col gap-2 p-3 lg:flex-row"><label className="relative flex-1"><Search className="absolute top-1/2 left-3 -translate-y-1/2 text-slate-600" size={14} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search name, slug, owner, or region..." className="h-10 w-full rounded-xl border border-white/8 bg-black/20 pr-3 pl-9 text-xs text-slate-200 outline-none focus:border-blue-400/40" /></label><select value={status} onChange={(event) => setStatus(event.target.value as StatusFilter)} className="h-10 rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none"><option value="all">All statuses</option><option value="active">Active</option><option value="provisioning">Provisioning</option><option value="suspended">Suspended</option></select></Card.Content></Card>

      <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_360px]" : "grid-cols-1"}`}>
        <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0">
          <Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm font-medium text-slate-100">Tenant registry</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">{filtered.length} visible organizations · local development state</Card.Description></Card.Header>
          <Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Organization registry" className="min-w-[960px]"><Table.Header>{[["organization", "Organization"], ["status", "Status"], ["region", "Region"], ["resources", "Resources"], ["spend", "Spend / Budget"], ["requests", "Requests"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} isRowHeader={id === "organization"} className={`bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase ${["spend", "requests"].includes(id) ? "text-right" : ""}`}>{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((organization) => <Table.Row key={organization.id} id={organization.id} onClick={() => setSelectedId(organization.id)} className={`cursor-pointer border-t border-white/[0.045] transition hover:bg-white/[0.03] ${selectedId === organization.id ? "bg-blue-400/[0.055]" : ""}`}><Table.Cell><p className="text-xs font-medium text-slate-200">{organization.name}</p><p className="mt-0.5 font-mono text-[10px] text-slate-600">{organization.slug}</p></Table.Cell><Table.Cell><OrganizationBadge status={organization.status} /></Table.Cell><Table.Cell><p className="text-xs text-slate-300">{organization.region}</p><p className="text-[10px] text-slate-600">{organization.plan}</p></Table.Cell><Table.Cell><p className="text-xs text-slate-300">{organization.workspaces} workspaces · {organization.projects} projects</p><p className="text-[10px] text-slate-600">{organization.members} members</p></Table.Cell><Table.Cell className="text-right"><p className="font-mono text-xs text-slate-300">${organization.monthlyCostUsd.toLocaleString()}</p><p className="text-[10px] text-slate-600">of ${organization.budgetUsd.toLocaleString()}</p></Table.Cell><Table.Cell className="text-right font-mono text-[11px] text-slate-400">{organization.requests.toLocaleString()}</Table.Cell><Table.Cell><ChevronRight className="ml-auto text-slate-700" size={14} /></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content>
        </Card>

        {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex items-center justify-between"><span className="rounded-xl bg-blue-400/10 p-2 text-blue-300"><Building2 size={17} /></span><OrganizationBadge status={selected.status} /></div><h2 className="mt-4 text-base font-semibold text-white">{selected.name}</h2><p className="mt-1 font-mono text-[10px] text-slate-600">{selected.id}</p></div><div className="space-y-5 p-5"><div className="grid grid-cols-2 gap-2">{[["Workspaces", selected.workspaces], ["Projects", selected.projects], ["Members", selected.members], ["Requests", selected.requests.toLocaleString()]].map(([label, value]) => <div key={label} className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><p className="text-[9px] tracking-wider text-slate-600 uppercase">{label}</p><p className="mt-1.5 text-sm font-medium text-slate-200">{value}</p></div>)}</div><dl className="space-y-3 text-[11px]">{[["Owner", selected.owner], ["Plan", selected.plan], ["Data region", selected.region], ["Created", selected.createdAt]].map(([label, value]) => <div key={label} className="flex justify-between gap-4"><dt className="text-slate-600">{label}</dt><dd className="truncate text-slate-300">{value}</dd></div>)}</dl><div className="rounded-xl border border-white/[0.055] bg-black/20 p-3"><div className="flex justify-between text-[10px]"><span className="text-slate-500">Monthly budget</span><span className="text-slate-300">{selected.budgetUsd ? `${Math.round(selected.monthlyCostUsd / selected.budgetUsd * 100)}%` : "Not set"}</span></div><div className="mt-2 h-1.5 overflow-hidden rounded-full bg-white/5"><span className="block h-full rounded-full bg-blue-400" style={{ width: `${selected.budgetUsd ? Math.min(100, selected.monthlyCostUsd / selected.budgetUsd * 100) : 0}%` }} /></div></div><Button className="h-9 w-full gap-2 bg-blue-500 text-xs text-white"><Check size={13} /> Manage organization</Button></div></aside>}
      </section>

      {createOpen && <NewOrganizationDialog onClose={() => setCreateOpen(false)} onCreate={createOrganization} />}
    </div>
  );
}
