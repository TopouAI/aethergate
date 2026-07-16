"use client";

import { Button, Card, Table } from "@heroui/react";
import {
  Activity,
  CheckCircle2,
  ChevronRight,
  Download,
  FileClock,
  Fingerprint,
  LockKeyhole,
  RefreshCcw,
  SearchCheck,
  ShieldAlert,
  ShieldCheck,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  Dialog,
  Field,
  inputClass,
  MetricGrid,
  PageHeader,
  SearchBar,
  selectClass,
  StateBadge,
  type APIState,
} from "@/components/foundation/resource-ui";
import {
  getAuditRetention,
  listAuditEvents,
  listAuditExports,
  queueAuditExport,
  retryAuditExport,
  updateAuditRetention,
  verifyAuditIntegrity,
} from "@/lib/control-plane";
import { seedAuditEvents, seedAuditExports, seedAuditRetention } from "@/lib/audit-data";
import { foundationOrganizationId } from "@/lib/foundation-data";
import type {
  AuditEvent,
  AuditExport,
  AuditIntegrityResult,
  AuditOutcome,
  AuditRetentionPolicy,
  AuditRisk,
} from "@/types/audit";

type Workspace = "events" | "retention" | "exports";
type OutcomeFilter = "all" | AuditOutcome;
type RiskFilter = "all" | AuditRisk;

const operatorEmail = "holden@topoai.dev";

export function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>(seedAuditEvents);
  const [retention, setRetention] = useState<AuditRetentionPolicy>(seedAuditRetention);
  const [exports, setExports] = useState<AuditExport[]>(seedAuditExports);
  const [integrity, setIntegrity] = useState<AuditIntegrityResult>({
    valid: true,
    eventCount: seedAuditEvents.length,
    headHash: seedAuditEvents[0]?.integrityHash ?? "",
    firstInvalidId: "",
  });
  const [workspace, setWorkspace] = useState<Workspace>("events");
  const [selectedId, setSelectedId] = useState(seedAuditEvents[0]?.id ?? "");
  const [query, setQuery] = useState("");
  const [outcome, setOutcome] = useState<OutcomeFilter>("all");
  const [risk, setRisk] = useState<RiskFilter>("all");
  const [apiState, setAPIState] = useState<APIState>("connecting");
  const [notice, setNotice] = useState("");
  const [exportOpen, setExportOpen] = useState(false);
  const [retentionDays, setRetentionDays] = useState(String(seedAuditRetention.retentionDays));
  const [legalHold, setLegalHold] = useState(seedAuditRetention.legalHold);
  const [retentionFormat, setRetentionFormat] = useState<"csv" | "jsonl">(seedAuditRetention.exportFormat);
  const [exportFormat, setExportFormat] = useState<"csv" | "jsonl">("csv");
  const [exportStart, setExportStart] = useState("2026-06-15T00:00");
  const [exportEnd, setExportEnd] = useState("2026-07-15T23:59");
  const [exportRisk, setExportRisk] = useState<RiskFilter>("all");
  const [exportOutcome, setExportOutcome] = useState<OutcomeFilter>("all");

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      listAuditEvents({ organizationId: foundationOrganizationId }, controller.signal),
      getAuditRetention(foundationOrganizationId, controller.signal),
      listAuditExports({ organizationId: foundationOrganizationId }, controller.signal),
      verifyAuditIntegrity(foundationOrganizationId, controller.signal),
    ])
      .then(([eventResponse, retentionResponse, exportResponse, integrityResponse]) => {
        setEvents(eventResponse.data);
        setRetention(retentionResponse.data);
        setExports(exportResponse.data);
        setIntegrity(integrityResponse.data);
        setRetentionDays(String(retentionResponse.data.retentionDays));
        setLegalHold(retentionResponse.data.legalHold);
        setRetentionFormat(retentionResponse.data.exportFormat);
        setSelectedId((current) =>
          eventResponse.data.some((item) => item.id === current) ? current : eventResponse.data[0]?.id ?? "",
        );
        setAPIState("connected");
      })
      .catch((error: unknown) => {
        if (!(error instanceof DOMException && error.name === "AbortError")) setAPIState("fallback");
      });
    return () => controller.abort();
  }, []);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return events.filter(
      (event) =>
        (outcome === "all" || event.outcome === outcome) &&
        (risk === "all" || event.riskLevel === risk) &&
        (!needle ||
          [
            event.actorEmail,
            event.actorId,
            event.action,
            event.resourceType,
            event.resourceId,
            event.reason,
            event.requestId,
            event.ipAddress,
          ].some((value) => value.toLowerCase().includes(needle))),
    );
  }, [events, outcome, query, risk]);

  const selected = events.find((event) => event.id === selectedId) ?? null;
  const elevatedCount = events.filter((event) => event.riskLevel === "high" || event.riskLevel === "critical").length;
  const deniedCount = events.filter((event) => event.outcome === "denied" || event.outcome === "failure").length;

  async function verifyChain() {
    setAPIState("saving");
    setNotice("");
    try {
      const response = await verifyAuditIntegrity(foundationOrganizationId);
      setIntegrity(response.data);
      setNotice(
        response.data.valid
          ? `Verified ${response.data.eventCount} immutable events against the SHA-256 chain.`
          : `Integrity verification failed at ${response.data.firstInvalidId}.`,
      );
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Integrity verification could not be completed.");
      setAPIState("fallback");
    }
  }

  async function saveRetention() {
    setAPIState("saving");
    setNotice("");
    try {
      const response = await updateAuditRetention({
        organizationId: foundationOrganizationId,
        retentionDays: Number(retentionDays),
        legalHold,
        exportFormat: retentionFormat,
        updatedBy: operatorEmail,
      });
      setRetention(response.data);
      setRetentionDays(String(response.data.retentionDays));
      setNotice(
        response.data.legalHold
          ? "Legal hold enabled. Retention expiry is suspended for this tenant."
          : `Retention policy saved at ${response.data.retentionDays} days.`,
      );
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Retention policy could not be saved.");
      setAPIState("fallback");
    }
  }

  async function createExport(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAPIState("saving");
    setNotice("");
    const filters: Record<string, string> = {};
    if (exportRisk !== "all") filters.riskLevel = exportRisk;
    if (exportOutcome !== "all") filters.outcome = exportOutcome;
    try {
      const response = await queueAuditExport({
        organizationId: foundationOrganizationId,
        requestedBy: operatorEmail,
        format: exportFormat,
        filters,
        periodStart: new Date(exportStart).toISOString(),
        periodEnd: new Date(exportEnd).toISOString(),
      });
      setExports((current) => [response.data, ...current]);
      setExportOpen(false);
      setWorkspace("exports");
      setNotice("Export accepted by the isolated Audit Export Worker queue.");
      setAPIState("connected");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Audit export could not be queued.");
      setAPIState("fallback");
    }
  }

  async function retryExport(item: AuditExport) {
    setNotice("");
    try {
      const response = await retryAuditExport(item.id, foundationOrganizationId, operatorEmail);
      setExports((current) => [response.data, ...current]);
      setNotice(`Retry ${response.data.id} queued with parent evidence ${item.id}.`);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Audit export could not be retried.");
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
      <PageHeader
        eyebrow="Security evidence"
        title="Audit Trail"
        description="Search tenant-scoped administrative events, verify the append-only hash chain, and operate retention and export evidence without mutating history."
        icon={ShieldCheck}
        apiState={apiState}
        action={
          <Button onPress={verifyChain} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white">
            <SearchCheck size={14} /> Verify integrity
          </Button>
        }
      />

      {notice && (
        <div className="rounded-xl border border-amber-400/15 bg-amber-400/[0.04] px-4 py-3 text-xs text-amber-200">
          {notice}
        </div>
      )}

      <MetricGrid
        items={[
          { label: "Retained events", value: events.length.toString(), hint: "Current tenant evidence", icon: Activity },
          { label: "Elevated risk", value: elevatedCount.toString(), hint: "High or critical events", icon: ShieldAlert },
          { label: "Denied / failed", value: deniedCount.toString(), hint: "Authorization and operation failures", icon: LockKeyhole },
          {
            label: "Integrity",
            value: integrity.valid ? "Verified" : "Failed",
            hint: `${integrity.eventCount} events in hash chain`,
            icon: integrity.valid ? CheckCircle2 : ShieldAlert,
          },
        ]}
      />

      <nav className="glass-panel flex flex-wrap items-center justify-between gap-3 p-2" aria-label="Audit workspaces">
        <div className="flex flex-wrap gap-2">
          {(
            [
              ["events", "Events", Fingerprint],
              ["retention", "Retention", FileClock],
              ["exports", "Exports", Download],
            ] as const
          ).map(([id, label, Icon]) => (
            <button
              key={id}
              type="button"
              onClick={() => setWorkspace(id)}
              className={`flex items-center gap-2 rounded-xl px-3 py-2 text-xs transition ${
                workspace === id
                  ? "bg-blue-500/15 text-blue-200 ring-1 ring-inset ring-blue-400/20"
                  : "text-slate-500 hover:bg-white/[0.035] hover:text-slate-200"
              }`}
            >
              <Icon size={13} /> {label}
            </button>
          ))}
        </div>
        <p className="px-2 font-mono text-[9px] text-slate-700">
          head {integrity.headHash ? `${integrity.headHash.slice(0, 12)}…` : "not verified"}
        </p>
      </nav>

      {workspace === "events" && (
        <>
          <SearchBar
            value={query}
            onChange={setQuery}
            placeholder="Search actor, action, resource, request, IP, or reason..."
            trailing={
              <div className="grid gap-2 sm:grid-cols-2">
                <select value={risk} onChange={(event) => setRisk(event.target.value as RiskFilter)} className={selectClass}>
                  <option value="all">All risk levels</option>
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
                <select value={outcome} onChange={(event) => setOutcome(event.target.value as OutcomeFilter)} className={selectClass}>
                  <option value="all">All outcomes</option>
                  <option value="success">Success</option>
                  <option value="failure">Failure</option>
                  <option value="denied">Denied</option>
                </select>
              </div>
            }
          />

          <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_420px]" : "grid-cols-1"}`}>
            <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0">
              <Card.Header className="border-b border-white/[0.055] p-4">
                <Card.Title className="text-sm text-slate-100">Immutable event registry</Card.Title>
                <Card.Description className="mt-1 text-xs text-slate-600">
                  {filtered.length} matching events · newest first · no update or delete operation is exposed
                </Card.Description>
              </Card.Header>
              <Card.Content className="p-0">
                <Table className="rounded-none border-0 bg-transparent">
                  <Table.ScrollContainer>
                    <Table.Content aria-label="Audit event registry" className="min-w-[1080px]">
                      <Table.Header>
                        {[
                          ["event", "Event"],
                          ["actor", "Actor"],
                          ["resource", "Resource"],
                          ["risk", "Risk"],
                          ["outcome", "Outcome"],
                          ["source", "Source"],
                          ["time", "Created"],
                          ["open", ""],
                        ].map(([id, label]) => (
                          <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">
                            {label}
                          </Table.Column>
                        ))}
                      </Table.Header>
                      <Table.Body>
                        {filtered.map((item) => (
                          <Table.Row
                            key={item.id}
                            id={item.id}
                            onClick={() => setSelectedId(item.id)}
                            className={`cursor-pointer border-t border-white/[0.045] hover:bg-white/[0.03] ${
                              selectedId === item.id ? "bg-blue-400/[0.055]" : ""
                            }`}
                          >
                            <Table.Cell>
                              <p className="text-xs font-medium text-slate-200">{item.action}</p>
                              <p className="mt-1 max-w-72 truncate text-[10px] text-slate-600">{item.reason}</p>
                            </Table.Cell>
                            <Table.Cell>
                              <p className="text-[11px] text-slate-300">{item.actorEmail}</p>
                              <p className="font-mono text-[9px] text-slate-700">{item.ipAddress}</p>
                            </Table.Cell>
                            <Table.Cell>
                              <p className="text-[11px] capitalize text-slate-400">{item.resourceType}</p>
                              <p className="max-w-44 truncate font-mono text-[9px] text-slate-700">{item.resourceId}</p>
                            </Table.Cell>
                            <Table.Cell><RiskBadge value={item.riskLevel} /></Table.Cell>
                            <Table.Cell><OutcomeBadge value={item.outcome} /></Table.Cell>
                            <Table.Cell className="font-mono text-[10px] text-slate-500">{item.source}</Table.Cell>
                            <Table.Cell className="text-[10px] text-slate-500">
                              {new Date(item.createdAt).toLocaleString("zh-CN")}
                            </Table.Cell>
                            <Table.Cell><ChevronRight size={14} className="ml-auto text-slate-700" /></Table.Cell>
                          </Table.Row>
                        ))}
                      </Table.Body>
                    </Table.Content>
                  </Table.ScrollContainer>
                </Table>
              </Card.Content>
            </Card>

            {selected && <EventDetail event={selected} />}
          </section>
        </>
      )}

      {workspace === "retention" && (
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
          <Card className="glass-panel px-0 py-0">
            <Card.Header className="border-b border-white/[0.055] p-5">
              <Card.Title className="text-sm text-slate-100">Tenant retention policy</Card.Title>
              <Card.Description className="mt-1 text-xs text-slate-600">
                Configure evidence lifetime and legal hold. The immutable event table remains protected from application-level mutation.
              </Card.Description>
            </Card.Header>
            <Card.Content className="grid gap-5 p-5 lg:grid-cols-2">
              <Field label="Retention days (30–2555)">
                <input
                  type="number"
                  min="30"
                  max="2555"
                  value={retentionDays}
                  onChange={(event) => setRetentionDays(event.target.value)}
                  className={inputClass}
                />
              </Field>
              <Field label="Default export format">
                <select value={retentionFormat} onChange={(event) => setRetentionFormat(event.target.value as "csv" | "jsonl")} className={selectClass}>
                  <option value="csv">CSV</option>
                  <option value="jsonl">JSON Lines</option>
                </select>
              </Field>
              <label className="flex items-center justify-between rounded-xl border border-white/[0.055] bg-white/[0.018] p-4 lg:col-span-2">
                <span>
                  <span className="block text-xs text-slate-300">Legal hold</span>
                  <span className="mt-1 block text-[10px] leading-5 text-slate-600">
                    Suspend retention expiry for investigations, litigation, or regulatory evidence preservation.
                  </span>
                </span>
                <input type="checkbox" checked={legalHold} onChange={(event) => setLegalHold(event.target.checked)} className="h-4 w-4 accent-blue-500" />
              </label>
            </Card.Content>
            <Card.Footer className="flex justify-end border-t border-white/[0.055] p-4">
              <Button
                onPress={saveRetention}
                isDisabled={Number(retentionDays) < 30 || Number(retentionDays) > 2555}
                className="h-9 bg-blue-500 px-4 text-xs text-white"
              >
                Save retention policy
              </Button>
            </Card.Footer>
          </Card>

          <Card className="glass-panel px-0 py-0">
            <Card.Header className="border-b border-white/[0.055] p-5">
              <Card.Title className="text-sm text-slate-100">Policy evidence</Card.Title>
            </Card.Header>
            <Card.Content className="space-y-4 p-5">
              <div className={`rounded-xl border p-4 ${retention.legalHold ? "border-amber-400/15 bg-amber-400/[0.035]" : "border-emerald-400/15 bg-emerald-400/[0.035]"}`}>
                <p className="text-xs text-slate-200">{retention.legalHold ? "Legal hold active" : "Standard retention active"}</p>
                <p className="mt-1 text-[10px] leading-5 text-slate-500">
                  {retention.legalHold ? "No event is eligible for expiry while the hold remains enabled." : `Evidence is retained for ${retention.retentionDays} days.`}
                </p>
              </div>
              <dl className="space-y-3 text-[11px]">
                {[
                  ["Organization", retention.organizationId],
                  ["Updated by", retention.updatedBy],
                  ["Updated", new Date(retention.updatedAt).toLocaleString("zh-CN")],
                  ["Export format", retention.exportFormat.toUpperCase()],
                ].map(([label, value]) => (
                  <div key={label} className="flex justify-between gap-4">
                    <dt className="text-slate-600">{label}</dt>
                    <dd className="max-w-64 truncate text-right text-slate-300">{value}</dd>
                  </div>
                ))}
              </dl>
              <p className="rounded-xl border border-blue-400/10 bg-blue-400/[0.025] p-3 text-[10px] leading-5 text-blue-100/60">
                Physical expiry is intentionally delegated to a privileged partition-retention worker; the application role cannot update or delete audit rows.
              </p>
            </Card.Content>
          </Card>
        </div>
      )}

      {workspace === "exports" && (
        <Card className="glass-panel overflow-hidden px-0 py-0">
          <Card.Header className="flex-row items-start justify-between gap-4 border-b border-white/[0.055] p-4">
            <div>
              <Card.Title className="text-sm text-slate-100">Export evidence</Card.Title>
              <Card.Description className="mt-1 text-xs text-slate-600">
                Worker-owned object generation with filters, checksums, result size, and parent-linked retries.
              </Card.Description>
            </div>
            <Button onPress={() => setExportOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white">
              <Download size={13} /> Queue export
            </Button>
          </Card.Header>
          <Card.Content className="p-0">
            <Table className="rounded-none border-0 bg-transparent">
              <Table.ScrollContainer>
                <Table.Content aria-label="Audit export history" className="min-w-[1080px]">
                  <Table.Header>
                    {[
                      ["export", "Export"],
                      ["period", "Period"],
                      ["filters", "Filters"],
                      ["status", "Status"],
                      ["evidence", "Evidence"],
                      ["requested", "Requested"],
                      ["retry", ""],
                    ].map(([id, label]) => (
                      <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>
                    ))}
                  </Table.Header>
                  <Table.Body>
                    {exports.map((item) => (
                      <Table.Row key={item.id} id={item.id} className="border-t border-white/[0.045]">
                        <Table.Cell>
                          <p className="text-xs text-slate-200">{item.format.toUpperCase()} export</p>
                          <p className="font-mono text-[9px] text-slate-700">{item.id}</p>
                          {item.parentId && <p className="font-mono text-[9px] text-blue-300/50">retry of {item.parentId}</p>}
                        </Table.Cell>
                        <Table.Cell className="text-[10px] text-slate-500">
                          <p>{new Date(item.periodStart).toLocaleDateString("zh-CN")}</p>
                          <p>{new Date(item.periodEnd).toLocaleDateString("zh-CN")}</p>
                        </Table.Cell>
                        <Table.Cell className="max-w-48 text-[10px] text-slate-500">
                          {Object.entries(item.filters).map(([key, value]) => `${key}=${value}`).join(", ") || "All events"}
                        </Table.Cell>
                        <Table.Cell><StateBadge value={item.status} /></Table.Cell>
                        <Table.Cell>
                          {item.status === "succeeded" ? (
                            <>
                              <p className="text-[10px] text-slate-400">{item.rowCount.toLocaleString()} rows · {formatBytes(item.sizeBytes)}</p>
                              <p className="max-w-56 truncate font-mono text-[9px] text-slate-700">sha256 {item.checksum}</p>
                            </>
                          ) : (
                            <p className="max-w-64 text-[10px] text-rose-300/60">{item.errorMessage || "Awaiting worker"}</p>
                          )}
                        </Table.Cell>
                        <Table.Cell>
                          <p className="text-[10px] text-slate-500">{item.requestedBy}</p>
                          <p className="text-[9px] text-slate-700">{new Date(item.createdAt).toLocaleString("zh-CN")}</p>
                        </Table.Cell>
                        <Table.Cell>
                          {item.status === "failed" && (
                            <Button onPress={() => retryExport(item)} variant="tertiary" className="h-8 gap-2 border border-white/8 px-3 text-[10px] text-slate-300">
                              <RefreshCcw size={11} /> Retry
                            </Button>
                          )}
                        </Table.Cell>
                      </Table.Row>
                    ))}
                  </Table.Body>
                </Table.Content>
              </Table.ScrollContainer>
            </Table>
          </Card.Content>
        </Card>
      )}

      {exportOpen && (
        <Dialog
          title="Queue audit export"
          description="The control plane records the request; a dedicated worker produces and checksums the object."
          submitLabel="Queue export"
          canSubmit={Boolean(exportStart && exportEnd && new Date(exportStart) < new Date(exportEnd))}
          onClose={() => setExportOpen(false)}
          onSubmit={createExport}
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Period start">
              <input type="datetime-local" value={exportStart} onChange={(event) => setExportStart(event.target.value)} className={inputClass} />
            </Field>
            <Field label="Period end">
              <input type="datetime-local" value={exportEnd} onChange={(event) => setExportEnd(event.target.value)} className={inputClass} />
            </Field>
            <Field label="Format">
              <select value={exportFormat} onChange={(event) => setExportFormat(event.target.value as "csv" | "jsonl")} className={selectClass}>
                <option value="csv">CSV</option>
                <option value="jsonl">JSON Lines</option>
              </select>
            </Field>
            <Field label="Risk filter">
              <select value={exportRisk} onChange={(event) => setExportRisk(event.target.value as RiskFilter)} className={selectClass}>
                <option value="all">All risk levels</option>
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="critical">Critical</option>
              </select>
            </Field>
          </div>
          <Field label="Outcome filter">
            <select value={exportOutcome} onChange={(event) => setExportOutcome(event.target.value as OutcomeFilter)} className={selectClass}>
              <option value="all">All outcomes</option>
              <option value="success">Success</option>
              <option value="failure">Failure</option>
              <option value="denied">Denied</option>
            </select>
          </Field>
        </Dialog>
      )}
    </div>
  );
}

function EventDetail({ event }: { event: AuditEvent }) {
  return (
    <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto">
      <div className="border-b border-white/[0.06] p-5">
        <div className="flex items-center justify-between">
          <RiskBadge value={event.riskLevel} />
          <OutcomeBadge value={event.outcome} />
        </div>
        <h2 className="mt-4 text-base font-semibold text-white">{event.action}</h2>
        <p className="mt-2 text-xs leading-5 text-slate-500">{event.reason}</p>
      </div>
      <div className="space-y-5 p-5">
        <dl className="space-y-3 text-[11px]">
          {[
            ["Actor", event.actorEmail],
            ["Resource", `${event.resourceType}/${event.resourceId}`],
            ["Source", event.source],
            ["Request", event.requestId],
            ["IP address", event.ipAddress],
            ["Created", new Date(event.createdAt).toLocaleString("zh-CN")],
          ].map(([label, value]) => (
            <div key={label} className="flex justify-between gap-4">
              <dt className="text-slate-600">{label}</dt>
              <dd className="max-w-64 truncate text-right text-slate-300">{value}</dd>
            </div>
          ))}
        </dl>

        <JsonEvidence label="Before state" value={event.beforeState} />
        <JsonEvidence label="After state" value={event.afterState} />

        <div className="rounded-xl border border-white/[0.055] bg-black/20 p-4">
          <p className="text-[9px] font-medium tracking-wide text-slate-600 uppercase">Integrity evidence</p>
          <div className="mt-3 space-y-3">
            <HashValue label="Previous" value={event.previousHash || "GENESIS"} />
            <HashValue label="Event SHA-256" value={event.integrityHash} />
          </div>
        </div>

        <p className="text-[9px] leading-4 text-slate-700">User agent: {event.userAgent}</p>
      </div>
    </aside>
  );
}

function JsonEvidence({ label, value }: { label: string; value: Record<string, unknown> }) {
  return (
    <div>
      <p className="mb-2 text-[9px] font-medium tracking-wide text-slate-600 uppercase">{label}</p>
      <pre className="max-h-44 overflow-auto rounded-xl border border-white/[0.055] bg-black/25 p-3 font-mono text-[9px] leading-5 text-slate-500">
        {JSON.stringify(value, null, 2)}
      </pre>
    </div>
  );
}

function HashValue({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[9px] text-slate-700">{label}</p>
      <p className="mt-1 break-all font-mono text-[9px] leading-4 text-blue-200/55">{value}</p>
    </div>
  );
}

function RiskBadge({ value }: { value: AuditRisk }) {
  const style =
    value === "critical"
      ? "bg-rose-400/10 text-rose-300 ring-rose-400/20"
      : value === "high"
        ? "bg-amber-400/10 text-amber-300 ring-amber-400/20"
        : value === "medium"
          ? "bg-blue-400/10 text-blue-300 ring-blue-400/20"
          : "bg-slate-400/10 text-slate-400 ring-slate-400/20";
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] capitalize ring-1 ring-inset ${style}`}>{value}</span>;
}

function OutcomeBadge({ value }: { value: AuditOutcome }) {
  const style =
    value === "success"
      ? "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20"
      : "bg-rose-400/10 text-rose-300 ring-rose-400/20";
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] capitalize ring-1 ring-inset ${style}`}>{value}</span>;
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`;
  return `${(value / 1024 / 1024).toFixed(1)} MiB`;
}
