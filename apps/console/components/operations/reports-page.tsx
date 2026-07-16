"use client";

import { Button, Card, Table } from "@heroui/react";
import { CalendarClock, Check, ChevronRight, Clock3, FileSpreadsheet, History, Mail, MessageSquare, Pause, Play, Plus, RefreshCcw, Send, TimerReset } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Dialog, Field, inputClass, MetricGrid, PageHeader, SearchBar, selectClass, StateBadge, type APIState } from "@/components/foundation/resource-ui";
import {
  activateReport, createReport, listReportRuns, listReports, pauseReport, queueReportRun, retryReportRun,
} from "@/lib/control-plane";
import { foundationOrganizationId } from "@/lib/foundation-data";
import { reportTemplateOptions, seedReportRuns, seedReports } from "@/lib/report-data";
import type {
  ReportFormat, ReportFrequency, ReportRecipient, ReportRun, ReportSchedule, ReportStatus, ReportTemplate,
} from "@/types/report";

const weekdayOptions = ["monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"];
const formatOptions: ReportFormat[] = ["csv", "xlsx", "pdf"];

export function ReportsPage() {
  const [reports, setReports] = useState<ReportSchedule[]>(seedReports);
  const [runs, setRuns] = useState<ReportRun[]>(seedReportRuns);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"all" | ReportStatus>("all");
  const [selectedId, setSelectedId] = useState(seedReports[0]?.id ?? "");
  const [apiState, setAPIState] = useState<APIState>("connecting");
  const [notice, setNotice] = useState("");
  const [open, setOpen] = useState(false);

  const [name, setName] = useState("");
  const [template, setTemplate] = useState<ReportTemplate>("executive_summary");
  const [initialStatus, setInitialStatus] = useState<ReportStatus>("active");
  const [frequency, setFrequency] = useState<ReportFrequency>("weekly");
  const [dayOfWeek, setDayOfWeek] = useState("monday");
  const [dayOfMonth, setDayOfMonth] = useState("1");
  const [localTime, setLocalTime] = useState("10:00");
  const [timezone, setTimezone] = useState("Asia/Shanghai");
  const [formats, setFormats] = useState<ReportFormat[]>(["xlsx", "pdf"]);
  const [email, setEmail] = useState("holden@topoai.dev");
  const [slackChannel, setSlackChannel] = useState("");
  const [filterKey, setFilterKey] = useState("environment");
  const [filterValue, setFilterValue] = useState("production");
  const [includeRawData, setIncludeRawData] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      listReports({ organizationId: foundationOrganizationId }, controller.signal),
      listReportRuns({ organizationId: foundationOrganizationId }, controller.signal),
    ]).then(([reportResponse, runResponse]) => {
      setReports(reportResponse.data);
      setRuns(runResponse.data);
      setSelectedId((current) => reportResponse.data.some((item) => item.id === current) ? current : reportResponse.data[0]?.id ?? "");
      setAPIState("connected");
    }).catch((error: unknown) => {
      if (!(error instanceof DOMException && error.name === "AbortError")) setAPIState("fallback");
    });
    return () => controller.abort();
  }, []);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return reports.filter((report) => (status === "all" || report.status === status)
      && (!needle || [report.name, report.template, report.frequency, report.timezone, ...report.formats].some((value) => value.toLowerCase().includes(needle))));
  }, [query, reports, status]);
  const selected = reports.find((report) => report.id === selectedId) ?? null;
  const selectedRuns = runs.filter((run) => !selected || run.reportId === selected.id);

  function toggleFormat(format: ReportFormat) {
    setFormats((current) => current.includes(format) ? current.filter((item) => item !== format) : [...current, format]);
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const recipients: ReportRecipient[] = [];
    if (email.trim()) recipients.push({ channel: "email", target: email.trim(), displayName: email.trim() });
    if (slackChannel.trim()) recipients.push({ channel: "slack", target: slackChannel.trim(), displayName: slackChannel.trim() });
    setAPIState("saving");
    setNotice("");
    try {
      const response = await createReport({
        organizationId: foundationOrganizationId, name, template, status: initialStatus, frequency,
        dayOfWeek: frequency === "weekly" ? dayOfWeek : "", dayOfMonth: frequency === "monthly" ? Number(dayOfMonth) : 0,
        localTime, timezone, formats, recipients,
        filters: filterKey.trim() && filterValue.trim() ? { [filterKey.trim()]: filterValue.trim() } : {},
        includeRawData,
      });
      setReports((current) => [response.data, ...current]);
      setSelectedId(response.data.id);
      setOpen(false);
      setName("");
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Report could not be created.");
      setAPIState("fallback");
    }
  }

  async function changeStatus(enabled: boolean) {
    if (!selected) return;
    setAPIState("saving");
    try {
      const response = enabled ? await activateReport(selected.id, foundationOrganizationId) : await pauseReport(selected.id, foundationOrganizationId);
      setReports((current) => current.map((item) => item.id === response.data.id ? response.data : item));
      setAPIState("connected");
      setNotice(enabled ? "Schedule activated with a timezone-safe next run." : "Schedule paused; no new scheduled run will be claimed.");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Report status could not be changed.");
      setAPIState("fallback");
    }
  }

  async function runNow() {
    if (!selected) return;
    try {
      const response = await queueReportRun(selected.id, { organizationId: foundationOrganizationId, requestedBy: "holden@topoai.dev", periodStart: null, periodEnd: null });
      setRuns((current) => [response.data, ...current]);
      setNotice("Manual report run accepted by the Reports Worker queue.");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Report run could not be queued.");
    }
  }

  async function retry(run: ReportRun) {
    try {
      const response = await retryReportRun(run.id, foundationOrganizationId, "holden@topoai.dev");
      setRuns((current) => [response.data, ...current]);
      setNotice("Failed report run accepted for retry with preserved reporting period.");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Report retry could not be queued.");
    }
  }

  return <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
    <PageHeader eyebrow="Scheduled intelligence" title="Reports" description="Schedule timezone-aware enterprise summaries and exports, then inspect generation, artifact, and delivery evidence." icon={CalendarClock} apiState={apiState} action={<Button onPress={() => setOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={14} />Create report</Button>} />
    {notice && <div className="rounded-xl border border-amber-400/15 bg-amber-400/[0.04] px-4 py-3 text-xs text-amber-200">{notice}</div>}

    <MetricGrid items={[
      { label: "Schedules", value: reports.length.toString(), hint: `${reports.filter((item) => item.status === "active").length} active`, icon: CalendarClock },
      { label: "Queued/running", value: runs.filter((item) => item.status === "queued" || item.status === "running").length.toString(), hint: "Reports Worker backlog", icon: Clock3 },
      { label: "Succeeded", value: runs.filter((item) => item.status === "succeeded").length.toString(), hint: "Artifact and delivery evidence", icon: Check },
      { label: "Failed", value: runs.filter((item) => item.status === "failed").length.toString(), hint: "Retry preserves the period", icon: TimerReset },
    ]} />

    <SearchBar value={query} onChange={setQuery} placeholder="Search report, template, frequency, timezone, or format..." trailing={<select value={status} onChange={(event) => setStatus(event.target.value as "all" | ReportStatus)} className={selectClass}><option value="all">All states</option><option value="active">Active</option><option value="paused">Paused</option></select>} />

    <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_390px]" : "grid-cols-1"}`}>
      <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">Report schedules</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Timezone-aware recurrence, formats, recipients, and next-run visibility.</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Report schedules" className="min-w-[1120px]"><Table.Header>{[["name", "Report"], ["status", "Status"], ["schedule", "Schedule"], ["template", "Template"], ["format", "Formats"], ["recipient", "Recipients"], ["last", "Last run"], ["next", "Next run"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} isRowHeader={id === "name"} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((report) => <Table.Row key={report.id} id={report.id} onClick={() => setSelectedId(report.id)} className={`cursor-pointer border-t border-white/[0.045] hover:bg-white/[0.03] ${selectedId === report.id ? "bg-blue-400/[0.055]" : ""}`}><Table.Cell><p className="text-xs font-medium text-slate-200">{report.name}</p><p className="font-mono text-[9px] text-slate-700">{report.id}</p></Table.Cell><Table.Cell><StateBadge value={report.status} /></Table.Cell><Table.Cell><p className="text-[11px] capitalize text-slate-300">{scheduleText(report)}</p><p className="text-[9px] text-slate-700">{report.timezone}</p></Table.Cell><Table.Cell className="text-[11px] capitalize text-slate-400">{report.template.replaceAll("_", " ")}</Table.Cell><Table.Cell><div className="flex gap-1">{report.formats.map((format) => <span key={format} className="rounded-md bg-blue-400/[0.06] px-1.5 py-1 font-mono text-[9px] uppercase text-blue-200">{format}</span>)}</div></Table.Cell><Table.Cell><p className="text-xs text-slate-400">{report.recipients.length}</p><p className="text-[9px] text-slate-700">{report.recipients.map((item) => item.channel).join(", ")}</p></Table.Cell><Table.Cell className="text-[11px] text-slate-500">{report.lastRunAt ? new Date(report.lastRunAt).toLocaleString("zh-CN") : "Never"}</Table.Cell><Table.Cell className="text-[11px] text-slate-500">{report.nextRunAt ? new Date(report.nextRunAt).toLocaleString("zh-CN") : "Paused"}</Table.Cell><Table.Cell><ChevronRight size={14} className="ml-auto text-slate-700" /></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>

      {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex justify-between"><span className="rounded-xl bg-blue-400/10 p-2 text-blue-300"><FileSpreadsheet size={17} /></span><StateBadge value={selected.status} /></div><h2 className="mt-4 text-base font-semibold text-white">{selected.name}</h2><p className="mt-1 text-[10px] capitalize text-slate-600">{selected.template.replaceAll("_", " ")}</p></div><div className="space-y-5 p-5"><div className="rounded-xl border border-blue-400/12 bg-blue-400/[0.035] p-4"><p className="text-xs font-medium text-slate-200">{scheduleText(selected)}</p><p className="mt-1 text-[10px] text-slate-500">{selected.timezone} · next {selected.nextRunAt ? new Date(selected.nextRunAt).toLocaleString("zh-CN") : "run paused"}</p></div><section><p className="mb-2 text-[10px] uppercase text-slate-600">Recipients</p><div className="space-y-2">{selected.recipients.map((recipient) => <div key={`${recipient.channel}:${recipient.target}`} className="flex items-center gap-2 rounded-xl border border-white/[0.055] bg-white/[0.018] p-3">{recipient.channel === "email" ? <Mail size={13} className="text-blue-300" /> : <MessageSquare size={13} className="text-violet-300" />}<div className="min-w-0"><p className="truncate text-[11px] text-slate-300">{recipient.displayName || recipient.target}</p><p className="truncate font-mono text-[9px] text-slate-700">{recipient.target}</p></div></div>)}</div></section><dl className="space-y-3 text-[11px]">{[["Formats", selected.formats.join(", ").toUpperCase()], ["Raw data", selected.includeRawData ? "Included" : "Excluded"], ["Filters", Object.entries(selected.filters).map(([key, value]) => `${key}=${value}`).join(", ") || "None"], ["Last run", selected.lastRunAt ? new Date(selected.lastRunAt).toLocaleString("zh-CN") : "Never"]].map(([label, value]) => <div key={label} className="flex justify-between gap-4"><dt className="text-slate-600">{label}</dt><dd className="max-w-56 truncate text-right text-slate-300">{value}</dd></div>)}</dl><div className="grid grid-cols-2 gap-2"><Button onPress={runNow} variant="tertiary" className="h-9 gap-2 border border-white/8 text-xs text-slate-300"><Send size={13} />Run now</Button>{selected.status === "active" ? <Button onPress={() => changeStatus(false)} variant="tertiary" className="h-9 gap-2 border border-amber-400/15 text-xs text-amber-200"><Pause size={13} />Pause</Button> : <Button onPress={() => changeStatus(true)} className="h-9 gap-2 bg-blue-500 text-xs text-white"><Play size={13} />Activate</Button>}</div></div></aside>}
    </section>

    <Card className="glass-panel px-0 py-0"><Card.Header className="p-4"><Card.Title className="text-sm text-slate-100">Run history</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Generation queue, reporting period, artifacts, delivery state, and retry lineage.</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Report run history" className="min-w-[1180px]"><Table.Header>{[["run", "Run"], ["status", "Status"], ["trigger", "Trigger"], ["period", "Period"], ["artifact", "Artifacts"], ["rows", "Rows"], ["delivery", "Delivery"], ["created", "Created"], ["action", "Action"]].map(([id, label]) => <Table.Column key={id} id={id} isRowHeader={id === "run"} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{selectedRuns.map((run) => <Table.Row key={run.id} id={run.id} className="border-t border-white/[0.045]"><Table.Cell><p className="text-xs text-slate-300">{run.reportName}</p><p className="font-mono text-[9px] text-slate-700">{run.id}</p></Table.Cell><Table.Cell><RunBadge value={run.status} /></Table.Cell><Table.Cell><p className="text-[11px] capitalize text-slate-400">{run.trigger}</p><p className="text-[9px] text-slate-700">attempt {run.attempt}</p></Table.Cell><Table.Cell><p className="font-mono text-[10px] text-slate-400">{new Date(run.periodStart).toLocaleDateString("zh-CN")} → {new Date(run.periodEnd).toLocaleDateString("zh-CN")}</p></Table.Cell><Table.Cell className="font-mono text-[11px] text-slate-400">{run.artifactCount || "—"}</Table.Cell><Table.Cell className="font-mono text-[11px] text-slate-400">{run.rowCount ? run.rowCount.toLocaleString() : "—"}</Table.Cell><Table.Cell><RunBadge value={run.deliveryStatus} /><p className="mt-1 max-w-64 truncate text-[9px] text-rose-300">{run.errorMessage}</p></Table.Cell><Table.Cell className="text-[11px] text-slate-500">{new Date(run.createdAt).toLocaleString("zh-CN")}</Table.Cell><Table.Cell>{run.status === "failed" ? <Button onPress={() => retry(run)} variant="tertiary" className="h-8 gap-1.5 border border-white/8 text-[10px] text-slate-300"><RefreshCcw size={11} />Retry</Button> : run.status === "queued" || run.status === "running" ? <span className="flex items-center gap-1 text-[10px] text-slate-600"><Clock3 size={11} />Worker queue</span> : <span className="flex items-center gap-1 text-[10px] text-slate-700"><History size={11} />Evidence</span>}</Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>

    {open && <Dialog title="Create scheduled report" description="Configure a timezone-aware schedule; generation and delivery are isolated in the Reports Worker." submitLabel="Create report" canSubmit={Boolean(name.trim() && formats.length && (email.trim() || slackChannel.trim()) && localTime && timezone)} onClose={() => setOpen(false)} onSubmit={submit}>
      <Field label="Report name"><input autoFocus value={name} onChange={(event) => setName(event.target.value)} className={inputClass} placeholder="Executive weekly summary" /></Field>
      <div className="grid gap-4 sm:grid-cols-2"><Field label="Template"><select value={template} onChange={(event) => setTemplate(event.target.value as ReportTemplate)} className={selectClass}>{reportTemplateOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></Field><Field label="Initial state"><select value={initialStatus} onChange={(event) => setInitialStatus(event.target.value as ReportStatus)} className={selectClass}><option value="active">Active</option><option value="paused">Paused</option></select></Field></div>
      <div className="rounded-xl border border-white/8 bg-black/15 p-3 text-[10px] leading-5 text-slate-500">{reportTemplateOptions.find((option) => option.value === template)?.description}</div>
      <div className="grid gap-4 sm:grid-cols-2"><Field label="Frequency"><select value={frequency} onChange={(event) => setFrequency(event.target.value as ReportFrequency)} className={selectClass}><option value="daily">Daily</option><option value="weekly">Weekly</option><option value="monthly">Monthly</option></select></Field>{frequency === "weekly" ? <Field label="Day of week"><select value={dayOfWeek} onChange={(event) => setDayOfWeek(event.target.value)} className={selectClass}>{weekdayOptions.map((day) => <option key={day} value={day} className="capitalize">{day}</option>)}</select></Field> : frequency === "monthly" ? <Field label="Day of month (1–28)"><input type="number" min="1" max="28" value={dayOfMonth} onChange={(event) => setDayOfMonth(event.target.value)} className={inputClass} /></Field> : <div />}</div>
      <div className="grid gap-4 sm:grid-cols-2"><Field label="Local run time"><input type="time" value={localTime} onChange={(event) => setLocalTime(event.target.value)} className={inputClass} /></Field><Field label="IANA timezone"><select value={timezone} onChange={(event) => setTimezone(event.target.value)} className={selectClass}>{["Asia/Shanghai", "Asia/Singapore", "UTC", "Europe/London", "America/Los_Angeles"].map((zone) => <option key={zone} value={zone}>{zone}</option>)}</select></Field></div>
      <Field label="Output formats"><div className="grid grid-cols-3 gap-2">{formatOptions.map((format) => <button key={format} type="button" onClick={() => toggleFormat(format)} className={`flex items-center justify-center gap-2 rounded-xl border px-3 py-2 text-[11px] uppercase ${formats.includes(format) ? "border-blue-400/25 bg-blue-400/[0.07] text-blue-200" : "border-white/8 bg-black/15 text-slate-500"}`}><span className={`grid h-4 w-4 place-items-center rounded border ${formats.includes(format) ? "border-blue-400/40 bg-blue-500/20" : "border-white/10"}`}>{formats.includes(format) && <Check size={10} />}</span>{format}</button>)}</div></Field>
      <div className="grid gap-4 sm:grid-cols-2"><Field label="Email recipient"><input type="email" value={email} onChange={(event) => setEmail(event.target.value)} className={inputClass} placeholder="finance@example.com" /></Field><Field label="Slack channel ID (optional)"><input value={slackChannel} onChange={(event) => setSlackChannel(event.target.value)} className={inputClass} placeholder="C_FINOPS" /></Field></div>
      <div className="grid gap-4 sm:grid-cols-2"><Field label="Filter key (optional)"><input value={filterKey} onChange={(event) => setFilterKey(event.target.value)} className={inputClass} /></Field><Field label="Filter value"><input value={filterValue} onChange={(event) => setFilterValue(event.target.value)} className={inputClass} /></Field></div>
      <label className="flex items-center justify-between rounded-xl border border-white/8 bg-black/15 p-3"><span><span className="block text-[11px] font-medium text-slate-300">Include request-level raw data</span><span className="mt-0.5 block text-[10px] text-slate-600">Use only when recipient authorization and retention policy allow it.</span></span><input type="checkbox" checked={includeRawData} onChange={(event) => setIncludeRawData(event.target.checked)} className="h-4 w-4 accent-blue-500" /></label>
    </Dialog>}
  </div>;
}

function scheduleText(report: ReportSchedule) {
  if (report.frequency === "weekly") return `Every ${report.dayOfWeek} at ${report.localTime}`;
  if (report.frequency === "monthly") return `Day ${report.dayOfMonth} monthly at ${report.localTime}`;
  return `Daily at ${report.localTime}`;
}

function RunBadge({ value }: { value: string }) {
  const style = value === "succeeded" || value === "delivered"
    ? "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20"
    : value === "failed" || value === "cancelled"
      ? "bg-rose-400/10 text-rose-300 ring-rose-400/20"
      : value === "running" || value === "partial"
        ? "bg-amber-400/10 text-amber-300 ring-amber-400/20"
        : "bg-blue-400/10 text-blue-300 ring-blue-400/20";
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] capitalize ring-1 ring-inset ${style}`}>{value}</span>;
}
