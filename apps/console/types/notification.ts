export type NotificationCategory = "alert" | "budget" | "provider" | "report" | "access" | "security" | "platform";
export type NotificationSeverity = "info" | "warning" | "critical";
export type NotificationStatus = "unread" | "read" | "archived";
export type NotificationChannel = "in_app" | "email" | "slack" | "teams" | "webhook";
export type NotificationDigest = "realtime" | "hourly" | "daily" | "weekly";

export type InboxNotification = {
  id: string;
  organizationId: string;
  recipientId: string;
  category: NotificationCategory;
  severity: NotificationSeverity;
  title: string;
  body: string;
  sourceType: string;
  sourceId: string;
  actionUrl: string;
  status: NotificationStatus;
  readAt: string | null;
  createdAt: string;
  updatedAt: string;
};

export type NotificationDestination = {
  channel: NotificationChannel;
  target: string;
  displayName: string;
};

export type NotificationPreference = {
  organizationId: string;
  recipientId: string;
  destinations: NotificationDestination[];
  categoryChannels: Partial<Record<NotificationCategory, NotificationChannel[]>>;
  digestFrequency: NotificationDigest;
  minimumSeverity: NotificationSeverity;
  timezone: string;
  quietHoursEnabled: boolean;
  quietStart: string;
  quietEnd: string;
  updatedAt: string;
};

export type NotificationEscalationRoute = {
  level: number;
  delayMinutes: number;
  channel: NotificationChannel;
  target: string;
  displayName: string;
};

export type NotificationEscalationPolicy = {
  id: string;
  organizationId: string;
  name: string;
  status: "active" | "paused";
  categories: NotificationCategory[];
  minimumSeverity: NotificationSeverity;
  acknowledgeWithinMinutes: number;
  repeatEveryMinutes: number;
  maxEscalations: number;
  routes: NotificationEscalationRoute[];
  createdAt: string;
  updatedAt: string;
};

export type NotificationDelivery = {
  id: string;
  organizationId: string;
  notificationId: string;
  notification: string;
  recipientId: string;
  channel: Exclude<NotificationChannel, "in_app">;
  target: string;
  displayName: string;
  status: "queued" | "deferred" | "sending" | "delivered" | "failed" | "suppressed";
  attempt: number;
  availableAt: string;
  deliveredAt: string | null;
  errorMessage: string;
  parentId: string | null;
  createdAt: string;
};

export type NotificationEscalationEvaluation = {
  matched: boolean;
  matches: Array<{ policyId: string; policyName: string; routes: NotificationEscalationRoute[] }>;
};
