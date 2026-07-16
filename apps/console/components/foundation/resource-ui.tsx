"use client";

import { Button, Card } from "@heroui/react";
import type { LucideIcon } from "lucide-react";
import { Search, X } from "lucide-react";
import type { FormEvent, ReactNode } from "react";

export type APIState = "connecting" | "connected" | "fallback" | "saving";

const apiStateStyle: Record<APIState, string> = {
  connecting: "bg-slate-400/10 text-slate-400 ring-slate-400/15",
  connected: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/15",
  fallback: "bg-amber-400/10 text-amber-300 ring-amber-400/15",
  saving: "bg-blue-400/10 text-blue-300 ring-blue-400/15",
};

const apiStateLabel: Record<APIState, string> = {
  connecting: "Connecting",
  connected: "API connected",
  fallback: "Local preview",
  saving: "Saving",
};

export function PageHeader({ eyebrow, title, description, icon: Icon, apiState, action }: { eyebrow: string; title: string; description: string; icon: LucideIcon; apiState: APIState; action: ReactNode }) {
  return (
    <section className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <div className="mb-2 flex items-center gap-2 text-[11px] font-medium tracking-[0.12em] text-blue-300/80 uppercase"><Icon size={13} /> {eyebrow}</div>
        <div className="flex flex-wrap items-center gap-3"><h1 className="text-2xl font-semibold tracking-[-0.035em] text-white sm:text-3xl">{title}</h1><span className={`rounded-full px-2 py-1 text-[9px] font-medium tracking-wide uppercase ring-1 ring-inset ${apiStateStyle[apiState]}`}>{apiStateLabel[apiState]}</span></div>
        <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">{description}</p>
      </div>
      {action}
    </section>
  );
}

export function MetricGrid({ items }: { items: Array<{ label: string; value: string; hint: string; icon: LucideIcon }> }) {
  return <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">{items.map((metric) => <Card key={metric.label} className="glass-panel px-0 py-0"><Card.Content className="flex items-start justify-between p-4"><div><p className="text-[10px] font-medium tracking-wider text-slate-600 uppercase">{metric.label}</p><p className="mt-2 text-xl font-semibold text-white">{metric.value}</p><p className="mt-1 text-[10px] text-slate-600">{metric.hint}</p></div><span className="rounded-lg bg-blue-400/8 p-2 text-blue-300 ring-1 ring-inset ring-blue-400/12"><metric.icon size={15} /></span></Card.Content></Card>)}</section>;
}

export function SearchBar({ value, onChange, placeholder, trailing }: { value: string; onChange: (value: string) => void; placeholder: string; trailing?: ReactNode }) {
  return <Card className="glass-panel px-0 py-0"><Card.Content className="flex flex-col gap-2 p-3 lg:flex-row"><label className="relative flex-1"><Search className="absolute top-1/2 left-3 -translate-y-1/2 text-slate-600" size={14} /><input value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} className="h-10 w-full rounded-xl border border-white/8 bg-black/20 pr-3 pl-9 text-xs text-slate-200 outline-none focus:border-blue-400/40" /></label>{trailing}</Card.Content></Card>;
}

export function Dialog({ title, description, submitLabel, canSubmit, onClose, onSubmit, children }: { title: string; description: string; submitLabel: string; canSubmit: boolean; onClose: () => void; onSubmit: (event: FormEvent<HTMLFormElement>) => void; children: ReactNode }) {
  return <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#02040a]/80 p-4 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label={title}><form onSubmit={onSubmit} className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-white/10 bg-[#0a0f18] shadow-2xl shadow-black/60"><div className="flex items-start justify-between border-b border-white/[0.06] p-5"><div><h2 className="text-base font-semibold text-white">{title}</h2><p className="mt-1 text-xs text-slate-500">{description}</p></div><button type="button" onClick={onClose} className="rounded-lg p-2 text-slate-500 hover:bg-white/5 hover:text-white"><X size={15} /></button></div><div className="space-y-4 p-5">{children}</div><div className="flex justify-end gap-2 border-t border-white/[0.06] p-4"><Button type="button" onPress={onClose} variant="tertiary" className="h-9 border border-white/8 bg-white/[0.025] text-xs text-slate-300">Cancel</Button><Button type="submit" isDisabled={!canSubmit} className="h-9 bg-blue-500 px-4 text-xs text-white">{submitLabel}</Button></div></form></div>;
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return <label className="block"><span className="mb-1.5 block text-[11px] font-medium text-slate-400">{label}</span>{children}</label>;
}

export const inputClass = "h-10 w-full rounded-xl border border-white/8 bg-black/20 px-3 text-xs text-slate-200 outline-none focus:border-blue-400/40 focus:ring-2 focus:ring-blue-400/10";
export const selectClass = "h-10 w-full rounded-xl border border-white/8 bg-[#080d15] px-3 text-xs text-slate-300 outline-none focus:border-blue-400/40";

export function StateBadge({ value }: { value: string }) {
  const style = value === "active" || value === "production" ? "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20" : value === "invited" || value === "preview" || value === "staging" ? "bg-blue-400/10 text-blue-300 ring-blue-400/20" : value === "suspended" || value === "disabled" ? "bg-rose-400/10 text-rose-300 ring-rose-400/20" : "bg-violet-400/10 text-violet-300 ring-violet-400/20";
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] font-medium capitalize ring-1 ring-inset ${style}`}>{value}</span>;
}

export function EmptyState({ message }: { message: string }) {
  return <div className="p-10 text-center text-xs text-slate-600">{message}</div>;
}
