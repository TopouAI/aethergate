"use client";

import { Button } from "@heroui/react";
import {
  Bell,
  ChevronsUpDown,
  CircleHelp,
  Menu,
  PanelLeftClose,
  Search,
  Sparkles,
  X,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";

import { getNavigationItem, navigationGroups } from "@/lib/navigation";

type AppShellProps = {
  children: ReactNode;
};

export function AppShell({ children }: AppShellProps) {
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const current = useMemo(() => getNavigationItem(pathname), [pathname]);

  const sidebar = (
    <div className="flex h-full flex-col">
      <div className="flex h-16 items-center gap-3 border-b border-white/8 px-4">
        <div className="relative flex size-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-aether-400 via-aether-500 to-indigo-600 shadow-[0_0_28px_rgba(76,132,255,0.28)]">
          <Sparkles size={18} className="text-white" />
          <span className="absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full border-2 border-[#0b0f17] bg-emerald-400" />
        </div>
        {!collapsed && (
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold tracking-[0.01em] text-white">AetherGate</p>
            <p className="truncate text-[11px] text-slate-500">Enterprise Console</p>
          </div>
        )}
        <button
          type="button"
          aria-label="关闭移动导航"
          className="ml-auto rounded-lg p-2 text-slate-400 transition hover:bg-white/5 hover:text-white lg:hidden"
          onClick={() => setMobileOpen(false)}
        >
          <X size={18} />
        </button>
      </div>

      <div className="px-3 pt-3">
        <button
          type="button"
          className="flex w-full items-center gap-3 rounded-xl border border-white/8 bg-white/[0.025] px-3 py-2.5 text-left transition hover:border-white/14 hover:bg-white/[0.045]"
        >
          <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-blue-400/12 text-xs font-semibold text-blue-300">T</span>
          {!collapsed && (
            <>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-xs font-medium text-slate-200">TopoAI</span>
                <span className="block truncate text-[10px] text-slate-500">Production workspace</span>
              </span>
              <ChevronsUpDown size={14} className="text-slate-500" />
            </>
          )}
        </button>
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4" aria-label="主导航">
        <div className="flex flex-col gap-5">
          {navigationGroups.map((group) => (
            <div key={group.name} className="flex flex-col gap-1">
              {!collapsed && <p className="px-2 pb-1 text-[10px] font-semibold tracking-[0.16em] text-slate-600 uppercase">{group.name}</p>}
              {group.items.map((item) => {
                const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    title={collapsed ? item.name : undefined}
                    onClick={() => setMobileOpen(false)}
                    className={`group relative flex items-center gap-3 rounded-lg px-2.5 py-2 text-xs transition ${
                      active
                        ? "bg-blue-400/10 text-blue-200"
                        : "text-slate-400 hover:bg-white/[0.035] hover:text-slate-100"
                    }`}
                  >
                    {active && <span className="absolute inset-y-2 left-0 w-0.5 rounded-full bg-blue-400 shadow-[0_0_10px_rgba(96,165,250,0.7)]" />}
                    <Icon size={16} strokeWidth={active ? 2.1 : 1.7} className={active ? "text-blue-300" : "text-slate-500 group-hover:text-slate-300"} />
                    {!collapsed && <span className="truncate">{item.name}</span>}
                  </Link>
                );
              })}
            </div>
          ))}
        </div>
      </nav>

      <div className="border-t border-white/8 p-3">
        <div className={`flex items-center ${collapsed ? "justify-center" : "gap-3"}`}>
          <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-slate-700 to-slate-900 text-xs font-semibold text-slate-100 ring-1 ring-white/10">HS</div>
          {!collapsed && (
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-medium text-slate-200">Holden Sun</p>
              <p className="truncate text-[10px] text-slate-500">Platform administrator</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-transparent">
      {mobileOpen && (
        <button
          type="button"
          aria-label="关闭导航遮罩"
          className="fixed inset-0 z-40 bg-black/70 backdrop-blur-sm lg:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      <aside
        className={`fixed inset-y-0 left-0 z-50 border-r border-white/8 bg-[#080c13]/96 shadow-2xl shadow-black/40 backdrop-blur-2xl transition-[width,transform] duration-300 ${
          collapsed ? "w-[76px]" : "w-[248px]"
        } ${mobileOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"}`}
      >
        {sidebar}
      </aside>

      <div className={`min-h-screen transition-[padding] duration-300 ${collapsed ? "lg:pl-[76px]" : "lg:pl-[248px]"}`}>
        <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b border-white/8 bg-[#05070b]/78 px-4 backdrop-blur-2xl sm:px-6">
          <button
            type="button"
            aria-label="打开导航"
            className="rounded-lg p-2 text-slate-400 transition hover:bg-white/5 hover:text-white lg:hidden"
            onClick={() => setMobileOpen(true)}
          >
            <Menu size={19} />
          </button>
          <button
            type="button"
            aria-label={collapsed ? "展开导航" : "收起导航"}
            className="hidden rounded-lg p-2 text-slate-500 transition hover:bg-white/5 hover:text-white lg:block"
            onClick={() => setCollapsed((value) => !value)}
          >
            <PanelLeftClose size={18} className={collapsed ? "rotate-180 transition" : "transition"} />
          </button>

          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-slate-100">{current?.name ?? "AetherGate"}</p>
            <p className="hidden truncate text-[11px] text-slate-500 md:block">{current?.description ?? "Enterprise AI control plane"}</p>
          </div>

          <div className="ml-auto flex items-center gap-1.5">
            <Button variant="tertiary" className="hidden h-9 min-w-52 justify-start gap-2 border border-white/8 bg-white/[0.025] px-3 text-xs text-slate-500 xl:flex">
              <Search size={14} />
              Search anything
              <span className="ml-auto rounded border border-white/10 px-1.5 py-0.5 font-mono text-[9px] text-slate-600">⌘K</span>
            </Button>
            <button type="button" aria-label="帮助" className="rounded-lg p-2 text-slate-500 transition hover:bg-white/5 hover:text-slate-200">
              <CircleHelp size={18} />
            </button>
            <button type="button" aria-label="通知" className="relative rounded-lg p-2 text-slate-500 transition hover:bg-white/5 hover:text-slate-200">
              <Bell size={18} />
              <span className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-blue-400 ring-2 ring-[#080b11]" />
            </button>
          </div>
        </header>

        <main className="surface-grid min-h-[calc(100vh-4rem)] px-4 py-5 sm:px-6 lg:px-8 lg:py-7">{children}</main>
      </div>
    </div>
  );
}

