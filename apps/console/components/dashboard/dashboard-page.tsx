"use client";

import { Button, Card, Table } from "@heroui/react";
import { ArrowDownRight, ArrowUpRight, Download, ExternalLink, Plus, Sparkles } from "lucide-react";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

import { getOverview, listRequests } from "@/lib/control-plane";
import { modelShare, overviewMetrics as seedOverviewMetrics, requests as seedRequests, trafficSeries } from "@/lib/mock-data";
import type { LlmRequest, OverviewMetric } from "@/types/observability";

import { TrafficChart } from "./traffic-chart";

type RangeKey = keyof typeof trafficSeries;

const toneClasses: Record<OverviewMetric["tone"], string> = {
  accent: "bg-blue-400/10 text-blue-300 ring-blue-400/15",
  success: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/15",
  warning: "bg-amber-400/10 text-amber-300 ring-amber-400/15",
  danger: "bg-rose-400/10 text-rose-300 ring-rose-400/15",
};

function formatTime(timestamp: string) {
  return new Intl.DateTimeFormat("zh-CN", { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false }).format(new Date(timestamp));
}

function statusBadge(status: LlmRequest["status"]) {
  const styles = {
    success: "bg-emerald-400/8 text-emerald-300 ring-emerald-400/15",
    error: "bg-rose-400/8 text-rose-300 ring-rose-400/15",
    rate_limited: "bg-amber-400/8 text-amber-300 ring-amber-400/15",
  };
  const labels = { success: "Success", error: "Error", rate_limited: "Rate limited" };
  return <span className={`inline-flex rounded-full px-2 py-1 text-[10px] font-medium ring-1 ring-inset ${styles[status]}`}>{labels[status]}</span>;
}

export function DashboardPage() {
  const [range, setRange] = useState<RangeKey>("7d");
  const [overviewMetrics, setOverviewMetrics] = useState(seedOverviewMetrics);
  const [requests, setRequests] = useState(seedRequests);

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      getOverview(controller.signal),
      listRequests({}, controller.signal),
    ]).then(([overview, requestList]) => {
      setOverviewMetrics(overview.data.metrics);
      setRequests(requestList.data);
    }).catch((error: unknown) => {
      if (error instanceof DOMException && error.name === "AbortError") return;
    });
    return () => controller.abort();
  }, []);

  const rangeLabel = useMemo(() => ({ "24h": "Last 24 hours", "7d": "Last 7 days", "30d": "Last 30 days" })[range], [range]);

  return (
    <div className="mx-auto flex w-full max-w-[1680px] flex-col gap-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2 text-[11px] font-medium tracking-[0.12em] text-blue-300/80 uppercase">
            <Sparkles size={13} />
            Usage intelligence
          </div>
          <h1 className="text-2xl font-semibold tracking-[-0.035em] text-white sm:text-3xl">Good afternoon, Holden.</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">Your AI estate is healthy. Request volume is growing while cost efficiency and P95 latency remain within policy.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="tertiary" className="h-9 gap-2 border border-white/9 bg-white/[0.025] px-3 text-xs text-slate-300">
            <Download size={14} /> Export
          </Button>
          <Button className="h-9 gap-2 bg-blue-500 px-3 text-xs font-medium text-white shadow-[0_8px_26px_rgba(59,130,246,0.24)]">
            <Plus size={14} /> Create API key
          </Button>
        </div>
      </section>

      <section className="grid gap-3 sm:grid-cols-2 2xl:grid-cols-4">
        {overviewMetrics.map((metric) => {
          const ImprovingIcon = metric.change >= 0 ? ArrowUpRight : ArrowDownRight;
          const positive = metric.label === "Error rate" || metric.label === "P95 latency" ? metric.change < 0 : metric.change >= 0;
          return (
            <Card key={metric.label} className="metric-glow border border-white/8 bg-[#0b1019]/88 px-0 py-0">
              <Card.Content className="flex flex-col gap-4 p-4 sm:p-5">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-[11px] font-medium tracking-wide text-slate-500 uppercase">{metric.label}</p>
                    <p className="mt-2 text-2xl font-semibold tracking-[-0.04em] text-slate-50">{metric.value}</p>
                  </div>
                  <span className={`rounded-lg p-2 ring-1 ring-inset ${toneClasses[metric.tone]}`}>
                    <ImprovingIcon size={16} />
                  </span>
                </div>
                <div className="flex items-center gap-2 text-[11px]">
                  <span className={positive ? "text-emerald-300" : "text-rose-300"}>{metric.change > 0 ? "+" : ""}{metric.change}%</span>
                  <span className="truncate text-slate-600">{metric.hint}</span>
                </div>
              </Card.Content>
            </Card>
          );
        })}
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.75fr)_minmax(320px,0.75fr)]">
        <Card className="glass-panel px-0 py-0">
          <Card.Header className="flex-row items-start justify-between gap-4 border-b border-white/[0.055] p-5">
            <div>
              <Card.Title className="text-sm font-medium text-slate-100">Request volume</Card.Title>
              <Card.Description className="mt-1 text-xs text-slate-500">Successful and failed gateway requests · {rangeLabel}</Card.Description>
            </div>
            <div className="flex rounded-lg border border-white/8 bg-black/20 p-0.5">
              {(["24h", "7d", "30d"] as RangeKey[]).map((item) => (
                <button
                  key={item}
                  type="button"
                  onClick={() => setRange(item)}
                  className={`rounded-md px-2.5 py-1.5 text-[10px] font-medium transition ${range === item ? "bg-white/9 text-white shadow-sm" : "text-slate-500 hover:text-slate-300"}`}
                >
                  {item.toUpperCase()}
                </button>
              ))}
            </div>
          </Card.Header>
          <Card.Content className="p-5 pt-2">
            <TrafficChart values={trafficSeries[range]} />
            <div className="flex items-center justify-between border-t border-white/[0.055] pt-4 text-[10px] text-slate-600">
              <span>{range === "24h" ? "00:00" : range === "7d" ? "Mon" : "Jun 15"}</span>
              <span>{range === "24h" ? "12:00" : range === "7d" ? "Thu" : "Jun 30"}</span>
              <span>{range === "24h" ? "Now" : range === "7d" ? "Today" : "Today"}</span>
            </div>
          </Card.Content>
        </Card>

        <Card className="glass-panel px-0 py-0">
          <Card.Header className="border-b border-white/[0.055] p-5">
            <Card.Title className="text-sm font-medium text-slate-100">Model mix</Card.Title>
            <Card.Description className="mt-1 text-xs text-slate-500">Share of spend for the selected period</Card.Description>
          </Card.Header>
          <Card.Content className="flex flex-col gap-4 p-5">
            <div className="flex h-2 overflow-hidden rounded-full bg-white/5">
              {modelShare.map((item) => <span key={item.model} style={{ width: `${item.share}%`, backgroundColor: item.color }} />)}
            </div>
            <div className="flex flex-col gap-3.5">
              {modelShare.map((item) => (
                <div key={item.model} className="flex items-center gap-3">
                  <span className="size-2 rounded-full" style={{ backgroundColor: item.color, boxShadow: `0 0 10px ${item.color}66` }} />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-medium text-slate-200">{item.model}</p>
                    <p className="truncate text-[10px] text-slate-600">{item.provider}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-xs font-medium text-slate-200">{item.share}%</p>
                    <p className="text-[10px] text-slate-600">{item.cost}</p>
                  </div>
                </div>
              ))}
            </div>
          </Card.Content>
        </Card>
      </section>

      <Card className="glass-panel overflow-hidden px-0 py-0">
        <Card.Header className="flex-row items-center justify-between gap-4 border-b border-white/[0.055] p-5">
          <div>
            <Card.Title className="text-sm font-medium text-slate-100">Live requests</Card.Title>
            <Card.Description className="mt-1 text-xs text-slate-500">Most recent traffic across all projects and providers</Card.Description>
          </div>
          <Link href="/requests" className="flex items-center gap-1.5 text-[11px] font-medium text-blue-300 transition hover:text-blue-200">
            Open explorer <ExternalLink size={12} />
          </Link>
        </Card.Header>
        <Card.Content className="p-0">
          <Table className="rounded-none border-0 bg-transparent">
            <Table.ScrollContainer>
              <Table.Content aria-label="Live LLM requests" className="min-w-[900px]">
                <Table.Header>
                  <Table.Column id="time" isRowHeader className="bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase">Time</Table.Column>
                  <Table.Column id="model" className="bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase">Model</Table.Column>
                  <Table.Column id="project" className="bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase">Project</Table.Column>
                  <Table.Column id="status" className="bg-white/[0.018] text-[10px] font-medium tracking-wide text-slate-600 uppercase">Status</Table.Column>
                  <Table.Column id="tokens" className="bg-white/[0.018] text-right text-[10px] font-medium tracking-wide text-slate-600 uppercase">Tokens</Table.Column>
                  <Table.Column id="latency" className="bg-white/[0.018] text-right text-[10px] font-medium tracking-wide text-slate-600 uppercase">Latency</Table.Column>
                  <Table.Column id="cost" className="bg-white/[0.018] text-right text-[10px] font-medium tracking-wide text-slate-600 uppercase">Cost</Table.Column>
                </Table.Header>
                <Table.Body>
                  {requests.slice(0, 5).map((request) => (
                    <Table.Row key={request.id} id={request.id} className="border-t border-white/[0.045] transition hover:bg-white/[0.025]">
                      <Table.Cell className="font-mono text-[10px] text-slate-500">{formatTime(request.timestamp)}</Table.Cell>
                      <Table.Cell>
                        <p className="text-xs font-medium text-slate-200">{request.model}</p>
                        <p className="text-[10px] text-slate-600">{request.provider}</p>
                      </Table.Cell>
                      <Table.Cell>
                        <p className="max-w-44 truncate text-xs text-slate-300">{request.project}</p>
                        <p className="max-w-44 truncate text-[10px] text-slate-600">{request.user}</p>
                      </Table.Cell>
                      <Table.Cell>{statusBadge(request.status)}</Table.Cell>
                      <Table.Cell className="text-right font-mono text-[11px] text-slate-400">{(request.inputTokens + request.outputTokens).toLocaleString()}</Table.Cell>
                      <Table.Cell className="text-right font-mono text-[11px] text-slate-400">{request.latencyMs.toLocaleString()}ms</Table.Cell>
                      <Table.Cell className="text-right font-mono text-[11px] text-slate-300">${request.costUsd.toFixed(4)}</Table.Cell>
                    </Table.Row>
                  ))}
                </Table.Body>
              </Table.Content>
            </Table.ScrollContainer>
          </Table>
        </Card.Content>
      </Card>
    </div>
  );
}

