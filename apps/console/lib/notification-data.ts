import type { InboxNotification, NotificationDelivery, NotificationEscalationPolicy, NotificationPreference } from "@/types/notification";

export const notificationRecipientId = "holden@topoai.dev";

export const seedNotifications: InboxNotification[] = [
  { id: "note_provider_offline", organizationId: "org_topoai", recipientId: notificationRecipientId, category: "provider", severity: "critical", title: "Azure East provider is offline", body: "Three active probes failed and the provider was removed from eligible routing targets.", sourceType: "provider_health", sourceId: "provider_azure_east", actionUrl: "/providers", status: "unread", readAt: null, createdAt: "2026-07-15T02:30:00Z", updatedAt: "2026-07-15T02:30:00Z" },
  { id: "note_budget_warning", organizationId: "org_topoai", recipientId: notificationRecipientId, category: "budget", severity: "warning", title: "Engineering Copilot reached 80% budget", body: "Current spend is $7,984 of the $10,000 monthly project budget.", sourceType: "budget", sourceId: "budget_engineering", actionUrl: "/budgets", status: "unread", readAt: null, createdAt: "2026-07-15T01:45:00Z", updatedAt: "2026-07-15T01:45:00Z" },
  { id: "note_report_ready", organizationId: "org_topoai", recipientId: notificationRecipientId, category: "report", severity: "info", title: "Executive weekly summary is ready", body: "XLSX and PDF artifacts were generated and delivered to two approved recipients.", sourceType: "report_run", sourceId: "rrun_exec_success", actionUrl: "/reports", status: "read", readAt: "2026-07-14T08:10:00Z", createdAt: "2026-07-14T08:30:00Z", updatedAt: "2026-07-14T08:10:00Z" },
  { id: "note_access_change", organizationId: "org_topoai", recipientId: notificationRecipientId, category: "access", severity: "warning", title: "Administrator role granted", body: "li.ming@topoai.dev was granted Administrator access by the organization owner.", sourceType: "member_role", sourceId: "member_li_ming", actionUrl: "/members", status: "unread", readAt: null, createdAt: "2026-07-14T04:30:00Z", updatedAt: "2026-07-14T04:30:00Z" },
];

export const seedNotificationPreference: NotificationPreference = {
  organizationId: "org_topoai", recipientId: notificationRecipientId,
  destinations: [
    { channel: "in_app", target: notificationRecipientId, displayName: "AetherGate inbox" },
    { channel: "email", target: notificationRecipientId, displayName: "Work email" },
    { channel: "slack", target: "C_PLATFORM_OPS", displayName: "#platform-ops" },
  ],
  categoryChannels: {
    alert: ["in_app", "email", "slack"], budget: ["in_app", "email"], provider: ["in_app", "slack"],
    report: ["in_app", "email"], access: ["in_app", "email"], security: ["in_app", "email", "slack"], platform: ["in_app"],
  },
  digestFrequency: "realtime", minimumSeverity: "info", timezone: "Asia/Shanghai",
  quietHoursEnabled: true, quietStart: "22:00", quietEnd: "08:00", updatedAt: "2026-07-01T00:00:00Z",
};

export const seedNotificationPolicies: NotificationEscalationPolicy[] = [{
  id: "npol_critical_ops", organizationId: "org_topoai", name: "Critical platform escalation", status: "active",
  categories: ["alert", "budget", "provider", "security"], minimumSeverity: "critical", acknowledgeWithinMinutes: 15,
  repeatEveryMinutes: 15, maxEscalations: 3,
  routes: [
    { level: 1, delayMinutes: 0, channel: "slack", target: "C_PLATFORM_OPS", displayName: "#platform-ops" },
    { level: 2, delayMinutes: 15, channel: "email", target: "oncall@topoai.dev", displayName: "Platform on-call" },
    { level: 3, delayMinutes: 30, channel: "teams", target: "vault://notifications/teams/leadership", displayName: "AI leadership" },
  ],
  createdAt: "2026-06-01T00:00:00Z", updatedAt: "2026-06-01T00:00:00Z",
}];

export const seedNotificationDeliveries: NotificationDelivery[] = [
  { id: "ndel_provider_slack", organizationId: "org_topoai", notificationId: "note_provider_offline", notification: "Azure East provider is offline", recipientId: notificationRecipientId, channel: "slack", target: "C_PLATFORM_OPS", displayName: "#platform-ops", status: "delivered", attempt: 1, availableAt: "2026-07-15T02:30:00Z", deliveredAt: "2026-07-15T02:30:04Z", errorMessage: "", parentId: null, createdAt: "2026-07-15T02:30:00Z" },
  { id: "ndel_budget_email_failed", organizationId: "org_topoai", notificationId: "note_budget_warning", notification: "Engineering Copilot reached 80% budget", recipientId: notificationRecipientId, channel: "email", target: notificationRecipientId, displayName: "Work email", status: "failed", attempt: 1, availableAt: "2026-07-15T01:45:00Z", deliveredAt: null, errorMessage: "SMTP relay timed out before acknowledgement.", parentId: null, createdAt: "2026-07-15T01:45:00Z" },
];
