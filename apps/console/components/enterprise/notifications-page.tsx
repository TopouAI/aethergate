"use client";
import { Button, Card, Table } from "@heroui/react";
import { Archive, BellRing, CheckCheck, ChevronRight, CircleAlert, Eye, EyeOff, Mail, Pause, Play, Plus, RefreshCcw, Route, Settings2, ShieldAlert } from "lucide-react";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { Dialog, Field, inputClass, MetricGrid, PageHeader, SearchBar, selectClass, StateBadge, type APIState } from "@/components/foundation/resource-ui";
import { activateNotificationPolicy, archiveNotification, createNotificationPolicy, evaluateNotificationEscalation, getNotificationPreference, listNotificationDeliveries, listNotificationPolicies, listNotifications, markAllNotificationsRead, markNotificationRead, markNotificationUnread, pauseNotificationPolicy, retryNotificationDelivery, updateNotificationPreference, } from "@/lib/control-plane";
import { foundationOrganizationId } from "@/lib/foundation-data";
import { notificationRecipientId, seedNotificationDeliveries, seedNotificationPolicies, seedNotificationPreference, seedNotifications, } from "@/lib/notification-data";
import type { InboxNotification, NotificationCategory, NotificationChannel, NotificationDelivery, NotificationEscalationEvaluation, NotificationEscalationPolicy, NotificationPreference, NotificationSeverity, NotificationStatus, } from "@/types/notification";
const categories: NotificationCategory[] = ["alert", "budget", "provider", "report", "access", "security", "platform"];
const preferenceChannels: NotificationChannel[] = ["in_app", "email", "slack"];
type Workspace = "inbox" | "preferences" | "escalations" | "deliveries";
export function NotificationsPage() {
    const [notifications, setNotifications] = useState<InboxNotification[]>(seedNotifications);
    const [preference, setPreference] = useState<NotificationPreference>(seedNotificationPreference);
    const [policies, setPolicies] = useState<NotificationEscalationPolicy[]>(seedNotificationPolicies);
    const [deliveries, setDeliveries] = useState<NotificationDelivery[]>(seedNotificationDeliveries);
    const [workspace, setWorkspace] = useState<Workspace>("inbox");
    const [selectedId, setSelectedId] = useState(seedNotifications[0]?.id ?? "");
    const [query, setQuery] = useState("");
    const [status, setStatus] = useState<"all" | NotificationStatus>("all");
    const [category, setCategory] = useState<"all" | NotificationCategory>("all");
    const [apiState, setAPIState] = useState<APIState>("connecting");
    const [notice, setNotice] = useState("");
    const [policyOpen, setPolicyOpen] = useState(false);
    const [emailTarget, setEmailTarget] = useState(seedNotificationPreference.destinations.find((item) => item.channel === "email")?.target ?? "");
    const [slackTarget, setSlackTarget] = useState(seedNotificationPreference.destinations.find((item) => item.channel === "slack")?.target ?? "");
    const [policyName, setPolicyName] = useState("");
    const [policyCategory, setPolicyCategory] = useState<NotificationCategory>("alert");
    const [policySeverity, setPolicySeverity] = useState<NotificationSeverity>("critical");
    const [acknowledgeMinutes, setAcknowledgeMinutes] = useState("15");
    const [repeatMinutes, setRepeatMinutes] = useState("15");
    const [routeChannel, setRouteChannel] = useState<Exclude<NotificationChannel, "in_app">>("slack");
    const [routeTarget, setRouteTarget] = useState("C_PLATFORM_OPS");
    const [routeName, setRouteName] = useState("#platform-ops");
    const [evaluationMinutes, setEvaluationMinutes] = useState("30");
    const [evaluation, setEvaluation] = useState<NotificationEscalationEvaluation | null>(null);
    useEffect(() => {
        const controller = new AbortController();
        Promise.all([
            listNotifications({ organizationId: foundationOrganizationId, recipientId: notificationRecipientId }, controller.signal),
            getNotificationPreference(foundationOrganizationId, notificationRecipientId, controller.signal),
            listNotificationPolicies({ organizationId: foundationOrganizationId }, controller.signal),
            listNotificationDeliveries({ organizationId: foundationOrganizationId, recipientId: notificationRecipientId }, controller.signal),
        ]).then(([notificationResponse, preferenceResponse, policyResponse, deliveryResponse]) => {
            setNotifications(notificationResponse.data);
            setPreference(preferenceResponse.data);
            setPolicies(policyResponse.data);
            setDeliveries(deliveryResponse.data);
            setEmailTarget(preferenceResponse.data.destinations.find((item) => item.channel === "email")?.target ?? "");
            setSlackTarget(preferenceResponse.data.destinations.find((item) => item.channel === "slack")?.target ?? "");
            setSelectedId((current) => notificationResponse.data.some((item) => item.id === current) ? current : notificationResponse.data[0]?.id ?? "");
            setAPIState("connected");
        }).catch((error: unknown) => {
            if (!(error instanceof DOMException && error.name === "AbortError"))
                setAPIState("fallback");
        });
        return () => controller.abort();
    }, []);
    const filtered = useMemo(() => {
        const needle = query.trim().toLowerCase();
        return notifications.filter((notification) => (status === "all" || notification.status === status)
            && (category === "all" || notification.category === category)
            && (!needle || [notification.title, notification.body, notification.sourceType, notification.sourceId].some((value) => value.toLowerCase().includes(needle))));
    }, [category, notifications, query, status]);
    const selected = notifications.find((notification) => notification.id === selectedId) ?? null;
    const unread = notifications.filter((notification) => notification.status === "unread").length;
    async function changeNotification(action: "read" | "unread" | "archive") {
        if (!selected)
            return;
        setAPIState("saving");
        try {
            const response = action === "read"
                ? await markNotificationRead(selected.id, foundationOrganizationId, notificationRecipientId)
                : action === "unread"
                    ? await markNotificationUnread(selected.id, foundationOrganizationId, notificationRecipientId)
                    : await archiveNotification(selected.id, foundationOrganizationId, notificationRecipientId);
            setNotifications((current) => current.map((item) => item.id === response.data.id ? response.data : item));
            setAPIState("connected");
        }
        catch (error) {
            setNotice(error instanceof Error ? error.message : "Notification state could not be changed.");
            setAPIState("fallback");
        }
    }
    async function readAll() {
        if (!unread)
            return;
        setAPIState("saving");
        try {
            const response = await markAllNotificationsRead(foundationOrganizationId, notificationRecipientId);
            const at = new Date().toISOString();
            setNotifications((current) => current.map((item) => item.status === "unread" ? { ...item, status: "read", readAt: at, updatedAt: at } : item));
            setNotice(`${response.data.updated} inbox item${response.data.updated === 1 ? "" : "s"} marked as read.`);
            setAPIState("connected");
        }
        catch (error) {
            setNotice(error instanceof Error ? error.message : "Inbox could not be updated.");
            setAPIState("fallback");
        }
    }
    function toggleCategoryChannel(routeCategory: NotificationCategory, channel: NotificationChannel) {
        if (channel === "in_app")
            return;
        setPreference((current) => {
            const existing = current.categoryChannels[routeCategory] ?? ["in_app"];
            const next = existing.includes(channel) ? existing.filter((item) => item !== channel) : [...existing, channel];
            return { ...current, categoryChannels: { ...current.categoryChannels, [routeCategory]: next } };
        });
    }
    async function savePreference() {
        setAPIState("saving");
        setNotice("");
        const availableChannels: NotificationChannel[] = ["in_app"];
        if (emailTarget.trim())
            availableChannels.push("email");
        if (slackTarget.trim())
            availableChannels.push("slack");
        const categoryChannels = Object.fromEntries(categories.map((item) => [item, (preference.categoryChannels[item] ?? ["in_app"]).filter((channel) => availableChannels.includes(channel))])) as Partial<Record<NotificationCategory, NotificationChannel[]>>;
        try {
            const response = await updateNotificationPreference({
                organizationId: foundationOrganizationId, recipientId: notificationRecipientId,
                destinations: [
                    { channel: "in_app", target: notificationRecipientId, displayName: "AetherGate inbox" },
                    ...(emailTarget.trim() ? [{ channel: "email" as const, target: emailTarget.trim(), displayName: "Work email" }] : []),
                    ...(slackTarget.trim() ? [{ channel: "slack" as const, target: slackTarget.trim(), displayName: "Operations channel" }] : []),
                ],
                categoryChannels, digestFrequency: preference.digestFrequency, minimumSeverity: preference.minimumSeverity,
                timezone: preference.timezone, quietHoursEnabled: preference.quietHoursEnabled,
                quietStart: preference.quietStart, quietEnd: preference.quietEnd,
            });
            setPreference(response.data);
            setNotice("Notification preferences saved with server-side channel routing.");
            setAPIState("connected");
        }
        catch (error) {
            setNotice(error instanceof Error ? error.message : "Notification preferences could not be saved.");
            setAPIState("fallback");
        }
    }
    async function submitPolicy(event: React.FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setAPIState("saving");
        try {
            const response = await createNotificationPolicy({
                organizationId: foundationOrganizationId, name: policyName, status: "active", categories: [policyCategory],
                minimumSeverity: policySeverity, acknowledgeWithinMinutes: Number(acknowledgeMinutes),
                repeatEveryMinutes: Number(repeatMinutes), maxEscalations: 1,
                routes: [{ level: 1, delayMinutes: 0, channel: routeChannel, target: routeTarget, displayName: routeName }],
            });
            setPolicies((current) => [response.data, ...current]);
            setPolicyOpen(false);
            setPolicyName("");
            setAPIState("connected");
        }
        catch (error) {
            setNotice(error instanceof Error ? error.message : "Escalation policy could not be created.");
            setAPIState("fallback");
        }
    }
    async function changePolicy(policy: NotificationEscalationPolicy) {
        try {
            const response = policy.status === "active"
                ? await pauseNotificationPolicy(policy.id, foundationOrganizationId)
                : await activateNotificationPolicy(policy.id, foundationOrganizationId);
            setPolicies((current) => current.map((item) => item.id === response.data.id ? response.data : item));
        }
        catch (error) {
            setNotice(error instanceof Error ? error.message : "Escalation policy state could not be changed.");
        }
    }
    async function evaluate() {
        try {
            const response = await evaluateNotificationEscalation({ organizationId: foundationOrganizationId, category: policyCategory, severity: policySeverity, unacknowledgedMinutes: Number(evaluationMinutes) });
            setEvaluation(response.data);
        }
        catch (error) {
            setNotice(error instanceof Error ? error.message : "Escalation evaluation failed.");
        }
    }
    async function retry(delivery: NotificationDelivery) {
        try {
            const response = await retryNotificationDelivery(delivery.id, foundationOrganizationId);
            setDeliveries((current) => [response.data, ...current]);
            setNotice("Failed delivery accepted by the isolated Notifications Worker queue.");
        }
        catch (error) {
            setNotice(error instanceof Error ? error.message : "Delivery could not be retried.");
        }
    }
    return <div className="mx-auto flex w-full max-w-[1760px] flex-col gap-5">
    <PageHeader eyebrow="Enterprise communication" title="Notifications" description="Operate a tenant-scoped inbox, personal delivery preferences, escalation policy, and external channel evidence without exposing integration secrets." icon={BellRing} apiState={apiState} action={<Button onPress={readAll} isDisabled={!unread} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><CheckCheck size={14}/>Mark all read</Button>}/>
    {notice && <div className="rounded-xl border border-amber-400/15 bg-amber-400/[0.04] px-4 py-3 text-xs text-amber-200">{notice}</div>}

    <MetricGrid items={[
            { label: "Unread", value: unread.toString(), hint: `${notifications.length} retained inbox items`, icon: BellRing },
            { label: "Critical", value: notifications.filter((item) => item.severity === "critical" && item.status !== "archived").length.toString(), hint: "Active high-severity notices", icon: CircleAlert },
            { label: "Escalations", value: policies.filter((item) => item.status === "active").length.toString(), hint: "Acknowledgement-aware policies", icon: ShieldAlert },
            { label: "Delivery failures", value: deliveries.filter((item) => item.status === "failed").length.toString(), hint: "Retryable worker evidence", icon: RefreshCcw },
        ]}/>

    <nav className="glass-panel flex flex-wrap gap-2 p-2" aria-label="Notification workspaces">
      {([[
                "inbox", "Inbox", BellRing,
            ], ["preferences", "Preferences", Settings2], ["escalations", "Escalations", Route], ["deliveries", "Delivery evidence", Mail]] as const).map(([id, label, Icon]) => <button key={id} type="button" onClick={() => setWorkspace(id)} className={`flex items-center gap-2 rounded-xl px-3 py-2 text-xs transition ${workspace === id ? "bg-blue-500/15 text-blue-200 ring-1 ring-inset ring-blue-400/20" : "text-slate-500 hover:bg-white/[0.035] hover:text-slate-200"}`}><Icon size={13}/>{label}</button>)}
    </nav>

    {workspace === "inbox" && <>
      <SearchBar value={query} onChange={setQuery} placeholder="Search title, body, source, or resource..." trailing={<div className="grid gap-2 sm:grid-cols-2"><select value={status} onChange={(event) => setStatus(event.target.value as "all" | NotificationStatus)} className={selectClass}><option value="all">All states</option><option value="unread">Unread</option><option value="read">Read</option><option value="archived">Archived</option></select><select value={category} onChange={(event) => setCategory(event.target.value as "all" | NotificationCategory)} className={selectClass}><option value="all">All categories</option>{categories.map((item) => <option key={item}>{item}</option>)}</select></div>}/>
      <section className={`grid min-w-0 gap-4 ${selected ? "2xl:grid-cols-[minmax(0,1fr)_390px]" : "grid-cols-1"}`}>
        <Card className="glass-panel min-w-0 overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">Recipient inbox</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Read state is isolated to {notificationRecipientId}.</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Notification inbox" className="min-w-[980px]"><Table.Header>{[["event", "Notification"], ["severity", "Severity"], ["category", "Category"], ["state", "State"], ["source", "Source"], ["time", "Created"], ["open", ""]].map(([id, label]) => <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{filtered.map((notification) => <Table.Row key={notification.id} id={notification.id} onClick={() => setSelectedId(notification.id)} className={`cursor-pointer border-t border-white/[0.045] hover:bg-white/[0.03] ${notification.status === "unread" ? "bg-blue-400/[0.025]" : ""} ${selectedId === notification.id ? "bg-blue-400/[0.06]" : ""}`}><Table.Cell><div className="flex items-start gap-2"><span className={`mt-1 h-1.5 w-1.5 rounded-full ${notification.status === "unread" ? "bg-blue-300" : "bg-slate-800"}`}/><div><p className="text-xs font-medium text-slate-200">{notification.title}</p><p className="mt-1 max-w-[420px] truncate text-[10px] text-slate-600">{notification.body}</p></div></div></Table.Cell><Table.Cell><ToneBadge value={notification.severity}/></Table.Cell><Table.Cell className="text-[11px] capitalize text-slate-400">{notification.category}</Table.Cell><Table.Cell><StateBadge value={notification.status}/></Table.Cell><Table.Cell><p className="font-mono text-[10px] text-slate-500">{notification.sourceType || "platform"}</p><p className="max-w-36 truncate font-mono text-[9px] text-slate-700">{notification.sourceId || "—"}</p></Table.Cell><Table.Cell className="text-[10px] text-slate-500">{new Date(notification.createdAt).toLocaleString("zh-CN")}</Table.Cell><Table.Cell><ChevronRight size={14} className="ml-auto text-slate-700"/></Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>

        {selected && <aside className="glass-panel sticky top-[76px] max-h-[calc(100vh-100px)] overflow-y-auto"><div className="border-b border-white/[0.06] p-5"><div className="flex items-center justify-between"><ToneBadge value={selected.severity}/><StateBadge value={selected.status}/></div><h2 className="mt-4 text-base font-semibold text-white">{selected.title}</h2><p className="mt-3 text-xs leading-6 text-slate-400">{selected.body}</p></div><div className="space-y-5 p-5"><dl className="space-y-3 text-[11px]">{[["Category", selected.category], ["Source", selected.sourceType || "platform"], ["Resource", selected.sourceId || "—"], ["Created", new Date(selected.createdAt).toLocaleString("zh-CN")], ["Read", selected.readAt ? new Date(selected.readAt).toLocaleString("zh-CN") : "Not yet"]].map(([label, value]) => <div key={label} className="flex justify-between gap-3"><dt className="text-slate-600">{label}</dt><dd className="max-w-56 truncate text-right text-slate-300">{value}</dd></div>)}</dl>{selected.actionUrl && <Link href={selected.actionUrl} className="flex h-9 w-full items-center justify-center rounded-xl border border-white/8 text-xs text-slate-200 hover:bg-white/[0.035]">Open related resource</Link>}<div className="grid grid-cols-2 gap-2">{selected.status === "unread" ? <Button onPress={() => changeNotification("read")} className="h-9 gap-2 bg-blue-500 text-xs text-white"><Eye size={13}/>Mark read</Button> : <Button onPress={() => changeNotification("unread")} variant="tertiary" className="h-9 gap-2 border border-white/8 text-xs text-slate-300"><EyeOff size={13}/>Mark unread</Button>}<Button onPress={() => changeNotification("archive")} variant="tertiary" className="h-9 gap-2 border border-white/8 text-xs text-slate-300"><Archive size={13}/>Archive</Button></div></div></aside>}
      </section>
    </>}

    {workspace === "preferences" && <Card className="glass-panel px-0 py-0"><Card.Header className="flex-row items-start justify-between gap-4 border-b border-white/[0.055] p-5"><div><Card.Title className="text-sm text-slate-100">Personal routing preference</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Channel destinations remain server-side; category routing never exposes connector credentials.</Card.Description></div><Button onPress={savePreference} className="h-9 bg-blue-500 px-4 text-xs text-white">Save preferences</Button></Card.Header><Card.Content className="grid gap-6 p-5 xl:grid-cols-[380px_minmax(0,1fr)]"><div className="space-y-4"><Field label="Digest frequency"><select value={preference.digestFrequency} onChange={(event) => setPreference((current) => ({ ...current, digestFrequency: event.target.value as NotificationPreference["digestFrequency"] }))} className={selectClass}><option value="realtime">Realtime</option><option value="hourly">Hourly digest</option><option value="daily">Daily digest</option><option value="weekly">Weekly digest</option></select></Field><Field label="Minimum external severity"><select value={preference.minimumSeverity} onChange={(event) => setPreference((current) => ({ ...current, minimumSeverity: event.target.value as NotificationSeverity }))} className={selectClass}><option value="info">Info</option><option value="warning">Warning</option><option value="critical">Critical</option></select></Field><Field label="IANA timezone"><input value={preference.timezone} onChange={(event) => setPreference((current) => ({ ...current, timezone: event.target.value }))} className={inputClass}/></Field><label className="flex items-center justify-between rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><span><span className="block text-xs text-slate-300">Quiet hours</span><span className="mt-1 block text-[10px] text-slate-600">External deliveries are deferred, not dropped.</span></span><input type="checkbox" checked={preference.quietHoursEnabled} onChange={(event) => setPreference((current) => ({ ...current, quietHoursEnabled: event.target.checked }))} className="h-4 w-4 accent-blue-500"/></label><div className="grid grid-cols-2 gap-3"><Field label="Quiet start"><input type="time" value={preference.quietStart} onChange={(event) => setPreference((current) => ({ ...current, quietStart: event.target.value }))} className={inputClass}/></Field><Field label="Quiet end"><input type="time" value={preference.quietEnd} onChange={(event) => setPreference((current) => ({ ...current, quietEnd: event.target.value }))} className={inputClass}/></Field></div><Field label="Email destination"><input type="email" value={emailTarget} onChange={(event) => setEmailTarget(event.target.value)} className={inputClass} placeholder="name@company.com"/></Field><Field label="Slack channel ID"><input value={slackTarget} onChange={(event) => setSlackTarget(event.target.value)} className={inputClass} placeholder="C_PLATFORM_OPS"/></Field></div><div><p className="text-[10px] font-medium tracking-wide text-slate-500 uppercase">Category channel routing</p><div className="mt-3 overflow-hidden rounded-xl border border-white/[0.055]">{categories.map((item) => <div key={item} className="grid grid-cols-[130px_1fr] items-center gap-3 border-t border-white/[0.045] px-4 py-3 first:border-t-0"><span className="text-xs capitalize text-slate-300">{item}</span><div className="flex flex-wrap gap-2">{preferenceChannels.map((channel) => { const enabled = (preference.categoryChannels[item] ?? ["in_app"]).includes(channel); const unavailable = channel === "email" && !emailTarget.trim() || channel === "slack" && !slackTarget.trim(); return <button key={channel} type="button" disabled={channel === "in_app" || unavailable} onClick={() => toggleCategoryChannel(item, channel)} className={`rounded-lg px-2.5 py-1.5 text-[10px] ring-1 ring-inset ${enabled ? "bg-blue-400/10 text-blue-200 ring-blue-400/20" : "bg-white/[0.02] text-slate-600 ring-white/[0.06]"} disabled:cursor-not-allowed disabled:opacity-50`}>{channel.replace("_", "-")}</button>; })}</div></div>)}</div></div></Card.Content></Card>}

    {workspace === "escalations" && <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_400px]"><Card className="glass-panel px-0 py-0"><Card.Header className="flex-row items-start justify-between border-b border-white/[0.055] p-4"><div><Card.Title className="text-sm text-slate-100">Escalation policies</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Route unacknowledged high-severity events through explicit levels.</Card.Description></div><Button onPress={() => setPolicyOpen(true)} className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white"><Plus size={13}/>Create policy</Button></Card.Header><Card.Content className="divide-y divide-white/[0.05] p-0">{policies.map((policy) => <div key={policy.id} className="p-5"><div className="flex flex-wrap items-start justify-between gap-3"><div><div className="flex items-center gap-2"><p className="text-sm font-medium text-slate-200">{policy.name}</p><StateBadge value={policy.status}/></div><p className="mt-1 text-[10px] capitalize text-slate-600">{policy.categories.join(", ")} · {policy.minimumSeverity}+ · acknowledge within {policy.acknowledgeWithinMinutes}m</p></div><Button onPress={() => changePolicy(policy)} variant="tertiary" className="h-8 gap-2 border border-white/8 px-3 text-[10px] text-slate-300">{policy.status === "active" ? <><Pause size={11}/>Pause</> : <><Play size={11}/>Activate</>}</Button></div><div className="mt-4 grid gap-2 md:grid-cols-3">{policy.routes.map((route) => <div key={`${route.level}-${route.channel}-${route.target}`} className="rounded-xl border border-white/[0.055] bg-white/[0.018] p-3"><div className="flex justify-between"><span className="text-[9px] uppercase text-slate-600">Level {route.level}</span><span className="text-[9px] text-slate-600">+{route.delayMinutes}m</span></div><p className="mt-2 text-xs capitalize text-slate-300">{route.channel}</p><p className="mt-1 truncate font-mono text-[9px] text-slate-600">{route.displayName || route.target}</p></div>)}</div></div>)}</Card.Content></Card><Card className="glass-panel px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">Escalation dry-run</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">Evaluate active policies without sending a notification.</Card.Description></Card.Header><Card.Content className="space-y-4 p-4"><Field label="Category"><select value={policyCategory} onChange={(event) => setPolicyCategory(event.target.value as NotificationCategory)} className={selectClass}>{categories.map((item) => <option key={item}>{item}</option>)}</select></Field><Field label="Severity"><select value={policySeverity} onChange={(event) => setPolicySeverity(event.target.value as NotificationSeverity)} className={selectClass}><option>info</option><option>warning</option><option>critical</option></select></Field><Field label="Unacknowledged minutes"><input type="number" min="0" value={evaluationMinutes} onChange={(event) => setEvaluationMinutes(event.target.value)} className={inputClass}/></Field><Button onPress={evaluate} className="h-9 w-full bg-blue-500 text-xs text-white">Evaluate policies</Button>{evaluation && <div className={`rounded-xl border p-4 ${evaluation.matched ? "border-amber-400/15 bg-amber-400/[0.035]" : "border-emerald-400/15 bg-emerald-400/[0.035]"}`}><p className="text-xs text-slate-200">{evaluation.matched ? `${evaluation.matches.length} policy match${evaluation.matches.length === 1 ? "" : "es"}` : "No escalation would fire"}</p>{evaluation.matches.map((match) => <p key={match.policyId} className="mt-2 text-[10px] text-slate-500">{match.policyName} · {match.routes.length} eligible route{match.routes.length === 1 ? "" : "s"}</p>)}</div>}</Card.Content></Card></div>}

    {workspace === "deliveries" && <Card className="glass-panel overflow-hidden px-0 py-0"><Card.Header className="border-b border-white/[0.055] p-4"><Card.Title className="text-sm text-slate-100">External delivery evidence</Card.Title><Card.Description className="mt-1 text-xs text-slate-600">The control plane records queued/deferred/delivered/failed state; an isolated worker owns outbound requests.</Card.Description></Card.Header><Card.Content className="p-0"><Table className="rounded-none border-0 bg-transparent"><Table.ScrollContainer><Table.Content aria-label="Notification deliveries" className="min-w-[980px]"><Table.Header>{[["notification", "Notification"], ["channel", "Channel"], ["target", "Destination"], ["status", "Status"], ["attempt", "Attempt"], ["available", "Available"], ["error", "Evidence"], ["retry", ""]].map(([id, label]) => <Table.Column key={id} id={id} className="bg-white/[0.018] text-[10px] uppercase text-slate-600">{label}</Table.Column>)}</Table.Header><Table.Body>{deliveries.map((delivery) => <Table.Row key={delivery.id} id={delivery.id} className="border-t border-white/[0.045]"><Table.Cell><p className="text-xs text-slate-200">{delivery.notification}</p><p className="font-mono text-[9px] text-slate-700">{delivery.id}</p></Table.Cell><Table.Cell className="text-[11px] capitalize text-slate-400">{delivery.channel}</Table.Cell><Table.Cell><p className="text-[11px] text-slate-400">{delivery.displayName || delivery.target}</p><p className="max-w-48 truncate font-mono text-[9px] text-slate-700">{delivery.target}</p></Table.Cell><Table.Cell><DeliveryBadge value={delivery.status}/></Table.Cell><Table.Cell className="font-mono text-xs text-slate-400">{delivery.attempt}</Table.Cell><Table.Cell className="text-[10px] text-slate-500">{new Date(delivery.availableAt).toLocaleString("zh-CN")}</Table.Cell><Table.Cell className="max-w-64 text-[10px] text-slate-600">{delivery.errorMessage || (delivery.deliveredAt ? `Delivered ${new Date(delivery.deliveredAt).toLocaleString("zh-CN")}` : "Awaiting worker")}</Table.Cell><Table.Cell>{delivery.status === "failed" && <Button onPress={() => retry(delivery)} variant="tertiary" className="h-8 gap-2 border border-white/8 px-3 text-[10px] text-slate-300"><RefreshCcw size={11}/>Retry</Button>}</Table.Cell></Table.Row>)}</Table.Body></Table.Content></Table.ScrollContainer></Table></Card.Content></Card>}

    {policyOpen && <Dialog title="Create escalation policy" description="Define the first explicit route. Additional levels stay visible in the policy registry." submitLabel="Create policy" canSubmit={Boolean(policyName.trim() && routeTarget.trim() && Number(acknowledgeMinutes) > 0)} onClose={() => setPolicyOpen(false)} onSubmit={submitPolicy}><Field label="Policy name"><input autoFocus value={policyName} onChange={(event) => setPolicyName(event.target.value)} className={inputClass} placeholder="Critical security escalation"/></Field><div className="grid gap-4 sm:grid-cols-2"><Field label="Category"><select value={policyCategory} onChange={(event) => setPolicyCategory(event.target.value as NotificationCategory)} className={selectClass}>{categories.map((item) => <option key={item}>{item}</option>)}</select></Field><Field label="Minimum severity"><select value={policySeverity} onChange={(event) => setPolicySeverity(event.target.value as NotificationSeverity)} className={selectClass}><option>info</option><option>warning</option><option>critical</option></select></Field><Field label="Acknowledge within"><input type="number" min="1" max="1440" value={acknowledgeMinutes} onChange={(event) => setAcknowledgeMinutes(event.target.value)} className={inputClass}/></Field><Field label="Repeat minutes"><input type="number" min="0" max="1440" value={repeatMinutes} onChange={(event) => setRepeatMinutes(event.target.value)} className={inputClass}/></Field><Field label="Level 1 channel"><select value={routeChannel} onChange={(event) => setRouteChannel(event.target.value as Exclude<NotificationChannel, "in_app">)} className={selectClass}><option>email</option><option>slack</option><option>teams</option><option>webhook</option></select></Field><Field label="Destination"><input value={routeTarget} onChange={(event) => setRouteTarget(event.target.value)} className={inputClass}/></Field></div><Field label="Destination label"><input value={routeName} onChange={(event) => setRouteName(event.target.value)} className={inputClass}/></Field></Dialog>}
  </div>;
}
function ToneBadge({ value }: {
    value: NotificationSeverity;
}) {
    const style = value === "critical" ? "bg-rose-400/10 text-rose-300 ring-rose-400/20" : value === "warning" ? "bg-amber-400/10 text-amber-300 ring-amber-400/20" : "bg-blue-400/10 text-blue-300 ring-blue-400/20";
    return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] capitalize ring-1 ring-inset ${style}`}>{value}</span>;
}
function DeliveryBadge({ value }: {
    value: NotificationDelivery["status"];
}) {
    const style = value === "delivered" ? "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20" : value === "failed" ? "bg-rose-400/10 text-rose-300 ring-rose-400/20" : value === "deferred" || value === "suppressed" ? "bg-amber-400/10 text-amber-300 ring-amber-400/20" : "bg-blue-400/10 text-blue-300 ring-blue-400/20";
    return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] capitalize ring-1 ring-inset ${style}`}>{value}</span>;
}
