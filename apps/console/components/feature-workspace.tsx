import { Button, Card } from "@heroui/react";
import { ArrowRight, Check, Construction, FileText, GitBranch, Layers3 } from "lucide-react";
import Link from "next/link";

import type { NavigationItem } from "@/lib/navigation";

const sourceStyles = {
  Helicone: "bg-violet-400/10 text-violet-300 ring-violet-400/20",
  AetherGate: "bg-blue-400/10 text-blue-300 ring-blue-400/20",
  Shared: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20",
};

export function FeatureWorkspace({ feature }: { feature: NavigationItem }) {
  const Icon = feature.icon;
  return (
    <div className="mx-auto flex w-full max-w-[1480px] flex-col gap-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <div className="mb-3 flex items-center gap-2">
            <span className={`rounded-full px-2.5 py-1 text-[10px] font-medium ring-1 ring-inset ${sourceStyles[feature.source]}`}>{feature.source} scope</span>
            <span className="flex items-center gap-1.5 text-[10px] text-amber-300"><Construction size={11} /> Migration workspace</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="rounded-xl bg-blue-400/10 p-2.5 text-blue-300 ring-1 ring-inset ring-blue-400/15"><Icon size={20} /></span>
            <h1 className="text-2xl font-semibold tracking-[-0.035em] text-white sm:text-3xl">{feature.name}</h1>
          </div>
          <p className="mt-3 max-w-3xl text-sm leading-6 text-slate-400">{feature.description}</p>
        </div>
        <div className="flex gap-2">
          <Link href="/requests"><Button variant="tertiary" className="h-9 gap-2 border border-white/9 bg-white/[0.025] px-3 text-xs text-slate-300">View working module</Button></Link>
          <Button className="h-9 gap-2 bg-blue-500 px-3 text-xs text-white">Open specification <ArrowRight size={13} /></Button>
        </div>
      </section>

      <section className="grid gap-4 lg:grid-cols-[minmax(0,1.4fr)_minmax(300px,0.6fr)]">
        <Card className="glass-panel px-0 py-0">
          <Card.Header className="border-b border-white/[0.055] p-5">
            <Card.Title className="flex items-center gap-2 text-sm font-medium text-slate-100"><Layers3 size={15} className="text-blue-300" /> Migration scope</Card.Title>
            <Card.Description className="mt-1 text-xs text-slate-500">The route and product contract are registered; service behavior will be delivered against the parity matrix.</Card.Description>
          </Card.Header>
          <Card.Content className="grid gap-3 p-5 sm:grid-cols-2">
            {feature.capabilities.map((capability) => (
              <div key={capability} className="flex items-start gap-3 rounded-xl border border-white/[0.055] bg-white/[0.018] p-3.5">
                <span className="mt-0.5 rounded-full bg-blue-400/10 p-1 text-blue-300"><Check size={11} /></span>
                <div><p className="text-xs font-medium text-slate-200">{capability}</p><p className="mt-1 text-[10px] leading-4 text-slate-600">Tracked as an independently testable capability.</p></div>
              </div>
            ))}
          </Card.Content>
        </Card>

        <div className="flex flex-col gap-4">
          <Card className="glass-panel px-0 py-0">
            <Card.Content className="space-y-4 p-5">
              <div className="flex items-center gap-2 text-sm font-medium text-slate-100"><GitBranch size={15} className="text-violet-300" /> Delivery state</div>
              <div className="space-y-3 text-xs">
                <div className="flex items-center justify-between"><span className="text-slate-500">Feature inventory</span><span className="text-emerald-300">Mapped</span></div>
                <div className="flex items-center justify-between"><span className="text-slate-500">Console route</span><span className="text-emerald-300">Registered</span></div>
                <div className="flex items-center justify-between"><span className="text-slate-500">Domain API</span><span className="text-amber-300">Queued</span></div>
                <div className="flex items-center justify-between"><span className="text-slate-500">End-to-end proof</span><span className="text-slate-500">Pending</span></div>
              </div>
            </Card.Content>
          </Card>
          <Card className="border border-blue-400/12 bg-blue-400/[0.035] px-0 py-0">
            <Card.Content className="p-5">
              <FileText size={17} className="text-blue-300" />
              <p className="mt-3 text-xs font-medium text-slate-200">No false-complete states</p>
              <p className="mt-1.5 text-[11px] leading-5 text-slate-500">A module is only marked complete after its API, authorization, persistence, UI states, and automated verification all pass.</p>
            </Card.Content>
          </Card>
        </div>
      </section>
    </div>
  );
}
