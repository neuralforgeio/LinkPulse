"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { motion, useMotionValue, useSpring, useTransform } from "framer-motion";
import {
  Activity,
  ArrowRight,
  Code2,
  Copy,
  Database,
  Link2,
  ShieldCheck,
  Users,
} from "lucide-react";
import { Logo } from "@/components/brand/logo";
import { ThemeToggle } from "@/components/theme-toggle";
import { FadeIn } from "@/components/motion/fade";
import { CountUp } from "@/components/motion/count-up";

const bars = [
  26, 40, 33, 52, 46, 60, 38, 68, 55, 64, 72, 50, 78, 66, 58, 86, 74, 62, 90,
  82, 70, 94, 76, 88,
];

const heroStats = [
  { label: "Total clicks", value: 12438 },
  { label: "Unique visitors", value: 8102 },
];

const showcaseLinks = [
  "promo2026",
  "launch-day",
  "docs-guide",
  "careers",
  "q1-webinar",
  "early-access",
  "app-v2",
  "black-friday",
];

function BentoCell({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 28 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-60px" }}
      transition={{ duration: 0.6, ease: "easeOut" }}
      whileHover={{ y: -4 }}
      className={`rounded-3xl border border-slate-200 bg-white p-6 transition-colors hover:border-emerald-500/40 dark:border-slate-800 dark:bg-slate-900/40 ${className}`}
    >
      {children}
    </motion.div>
  );
}

export default function Home() {
  const mouseX = useMotionValue(0);
  const mouseY = useMotionValue(0);
  const glowX = useSpring(mouseX, { stiffness: 55, damping: 18 });
  const glowY = useSpring(mouseY, { stiffness: 55, damping: 18 });

  const tiltX = useMotionValue(0);
  const tiltY = useMotionValue(0);
  const rotateX = useSpring(useTransform(tiltY, [-0.5, 0.5], [6, -6]), {
    stiffness: 160,
    damping: 18,
  });
  const rotateY = useSpring(useTransform(tiltX, [-0.5, 0.5], [-6, 6]), {
    stiffness: 160,
    damping: 18,
  });

  return (
    <div
      className="relative min-h-screen overflow-hidden bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-white"
      onMouseMove={(e) => {
        const rect = e.currentTarget.getBoundingClientRect();
        mouseX.set(e.clientX - rect.left);
        mouseY.set(e.clientY - rect.top);
      }}
    >
      <div
        className="dot-grid-light absolute inset-0 dark:hidden"
        aria-hidden="true"
      />
      <div
        className="dot-grid absolute inset-0 hidden dark:block"
        aria-hidden="true"
      />
      <div
        className="absolute -top-40 left-[calc(50%-26rem)] h-[32rem] w-[52rem] rounded-full bg-emerald-300/40 blur-[140px] animate-drift dark:bg-emerald-500/15"
        aria-hidden="true"
      />
      <motion.div
        aria-hidden="true"
        className="pointer-events-none absolute left-0 top-0 -ml-[18rem] -mt-[18rem] h-[36rem] w-[36rem] rounded-full bg-emerald-300/25 blur-[90px] dark:bg-emerald-400/10"
        style={{ x: glowX, y: glowY }}
      />

      {/* Header */}
      <header className="relative mx-auto flex max-w-6xl items-center justify-between px-6 py-6">
        <FadeIn y={-12}>
          <Logo />
        </FadeIn>
        <FadeIn y={-12} delay={0.1}>
          <nav className="flex items-center gap-4 text-sm">
            <ThemeToggle />
            <Link
              href="/login"
              className="text-slate-500 transition hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Sign in
            </Link>
            <Link
              href="/register"
              className="rounded-full bg-slate-900 px-4 py-2 font-semibold text-white transition hover:bg-slate-700 dark:bg-white dark:text-slate-950 dark:hover:bg-slate-200"
            >
              Get started
            </Link>
          </nav>
        </FadeIn>
      </header>

      {/* Hero */}
      <main className="relative mx-auto max-w-6xl px-6 pb-20">
        <section className="grid items-center gap-14 pt-14 lg:grid-cols-[1.05fr_1fr]">
          <div>
            <FadeIn delay={0.1}>
              <p className="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3.5 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                Open source &middot; Self-hosted &middot; MIT licensed
              </p>
            </FadeIn>

            <FadeIn delay={0.2}>
              <h1 className="mt-6 text-balance text-5xl font-bold leading-[1.05] tracking-tight sm:text-6xl">
                The shortener you&apos;ll actually{" "}
                <span className="bg-gradient-to-r from-emerald-600 to-teal-500 bg-clip-text text-transparent dark:from-emerald-300 dark:to-teal-300">
                  own
                </span>
                .
              </h1>
            </FadeIn>

            <FadeIn delay={0.3}>
              <p className="mt-5 max-w-lg text-lg leading-relaxed text-slate-600 dark:text-slate-400">
                Click analytics, team workspaces, and a public API —
                self-hosted, MIT licensed, zero vendor lock-in. Your data stays
                in your database.
              </p>
            </FadeIn>

            <FadeIn delay={0.4}>
              <div className="mt-8 flex flex-wrap gap-4">
                <Link
                  href="/register"
                  className="group inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-emerald-500 to-teal-500 px-6 py-3 font-semibold text-slate-950 shadow-lg shadow-emerald-500/25 transition hover:from-emerald-400 hover:to-teal-400"
                >
                  Create your first link
                  <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
                </Link>
                <Link
                  href="/login"
                  className="rounded-full border border-slate-300 px-6 py-3 font-semibold text-slate-700 transition hover:border-slate-400 hover:text-slate-900 dark:border-slate-700 dark:text-slate-300 dark:hover:border-slate-500 dark:hover:text-white"
                >
                  Sign in
                </Link>
              </div>
            </FadeIn>

            <FadeIn delay={0.5}>
              <div className="mt-9 flex flex-wrap gap-x-6 gap-y-2 text-xs text-slate-500">
                <span className="inline-flex items-center gap-1.5">
                  <ShieldCheck className="h-3.5 w-3.5 text-emerald-600/70 dark:text-emerald-500/70" />
                  No raw IPs stored
                </span>
                <span className="inline-flex items-center gap-1.5">
                  <Database className="h-3.5 w-3.5 text-emerald-600/70 dark:text-emerald-500/70" />
                  PostgreSQL inside
                </span>
                <span className="inline-flex items-center gap-1.5">
                  <Link2 className="h-3.5 w-3.5 text-emerald-600/70 dark:text-emerald-500/70" />
                  MIT licensed
                </span>
              </div>
            </FadeIn>
          </div>

          {/* Preview card with 3D tilt */}
          <FadeIn delay={0.45}>
            <div
              className="[perspective:1400px]"
              onMouseMove={(e) => {
                const rect = e.currentTarget.getBoundingClientRect();
                tiltX.set((e.clientX - rect.left) / rect.width - 0.5);
                tiltY.set((e.clientY - rect.top) / rect.height - 0.5);
              }}
              onMouseLeave={() => {
                tiltX.set(0);
                tiltY.set(0);
              }}
            >
              <motion.div
                style={{ rotateX, rotateY }}
                className="relative rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl dark:border-slate-800 dark:bg-slate-900/80"
              >
                <motion.div
                  animate={{ y: [0, -8, 0] }}
                  transition={{
                    duration: 4,
                    repeat: Infinity,
                    ease: "easeInOut",
                  }}
                  className="absolute -right-4 -top-5 flex items-center gap-2 rounded-full border border-emerald-500/40 bg-white px-3.5 py-2 text-xs font-medium text-emerald-700 shadow-xl dark:bg-slate-950/90 dark:text-emerald-300"
                >
                  <Activity className="h-3.5 w-3.5" />
                  Click spike +18%
                </motion.div>

                <div className="flex items-center justify-between border-b border-slate-200 pb-4 dark:border-slate-800">
                  <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                    <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse-ring" />
                    linkpulse / overview
                  </div>
                  <span className="rounded-md bg-slate-100 px-2.5 py-1 text-xs text-slate-500 dark:bg-slate-800 dark:text-slate-400">
                    Last 30 days
                  </span>
                </div>

                <div className="mt-5 grid grid-cols-2 gap-4">
                  {heroStats.map((stat) => (
                    <div
                      key={stat.label}
                      className="rounded-2xl bg-slate-100 p-4 dark:bg-slate-950/60"
                    >
                      <p className="text-xs font-medium uppercase tracking-wider text-slate-500">
                        {stat.label}
                      </p>
                      <p className="mt-1 text-2xl font-bold tracking-tight">
                        <CountUp to={stat.value} />
                      </p>
                    </div>
                  ))}
                </div>

                <div
                  className="mt-5 flex h-32 items-end gap-1.5 rounded-2xl bg-slate-100 p-4 dark:bg-slate-950/60"
                  aria-hidden="true"
                >
                  {bars.map((height, i) => (
                    <motion.div
                      key={i}
                      initial={{ height: "6%" }}
                      animate={{ height: `${height}%` }}
                      transition={{
                        duration: 0.9,
                        delay: 0.75 + i * 0.045,
                        ease: "easeOut",
                      }}
                      className="w-full rounded-sm bg-emerald-500/40 transition-colors hover:bg-emerald-500 dark:bg-emerald-500/30 dark:hover:bg-emerald-400"
                    />
                  ))}
                </div>
              </motion.div>
            </div>
          </FadeIn>
        </section>

        {/* Link ticker */}
        <FadeIn delay={0.6} className="relative mt-16">
          <div className="overflow-hidden [mask-image:linear-gradient(to_right,transparent,black_12%,black_88%,transparent)]">
            <div className="animate-marquee flex w-max gap-3">
              {[...showcaseLinks, ...showcaseLinks].map((code, i) => (
                <span
                  key={i}
                  className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white/70 px-4 py-2 font-mono text-sm dark:border-slate-800 dark:bg-slate-900/60"
                >
                  <Link2 className="h-3.5 w-3.5 text-emerald-500" />
                  <span className="text-slate-400 dark:text-slate-500">
                    linkpulse.app/
                  </span>
                  <span className="text-slate-600 dark:text-slate-300">
                    {code}
                  </span>
                </span>
              ))}
            </div>
          </div>
        </FadeIn>

        {/* Bento features */}
        <section className="mt-28">
          <FadeIn>
            <h2 className="text-balance text-3xl font-bold tracking-tight sm:text-4xl">
              Everything a shortener should be.
            </h2>
            <p className="mt-3 max-w-xl text-slate-600 dark:text-slate-400">
              Not just redirects — the whole lifecycle of a link, measured.
            </p>
          </FadeIn>

          <div className="mt-10 grid gap-5 md:grid-cols-3">
            <BentoCell className="md:col-span-2 md:row-span-2">
              <div className="flex h-full flex-col">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                  <Activity className="h-5 w-5" />
                </div>
                <h3 className="mt-4 text-lg font-semibold tracking-tight">
                  Analytics you trust
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-400">
                  Referrers, devices, browsers, campaigns, and time series —
                  from async, privacy-safe click tracking.
                </p>
                <div className="mt-auto pt-6">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="rounded-2xl bg-slate-100 p-4 dark:bg-slate-950/60">
                      <p className="text-xs uppercase tracking-wider text-slate-500">
                        Total clicks
                      </p>
                      <p className="mt-1 text-2xl font-bold">
                        <CountUp to={12438} />
                      </p>
                    </div>
                    <div className="rounded-2xl bg-slate-100 p-4 dark:bg-slate-950/60">
                      <p className="text-xs uppercase tracking-wider text-slate-500">
                        Unique visitors
                      </p>
                      <p className="mt-1 text-2xl font-bold">
                        <CountUp to={8102} />
                      </p>
                    </div>
                  </div>
                  <div
                    className="mt-4 flex h-24 items-end gap-1.5 rounded-2xl bg-slate-100 p-4 dark:bg-slate-950/60"
                    aria-hidden="true"
                  >
                    {bars.slice(0, 16).map((height, i) => (
                      <motion.div
                        key={i}
                        initial={{ height: "6%" }}
                        whileInView={{ height: `${height}%` }}
                        viewport={{ once: true }}
                        transition={{
                          duration: 0.7,
                          delay: i * 0.05,
                          ease: "easeOut",
                        }}
                        className="w-full rounded-sm bg-emerald-500/40 dark:bg-emerald-500/30"
                      />
                    ))}
                  </div>
                </div>
              </div>
            </BentoCell>

            <BentoCell>
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <Link2 className="h-5 w-5" />
              </div>
              <h3 className="mt-4 text-lg font-semibold tracking-tight">
                Links that work hard
              </h3>
              <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                Aliases, expiry, click limits, passwords, UTM.
              </p>
              <div className="mt-5 space-y-2">
                {["promo2026", "docs-guide"].map((code) => (
                  <div
                    key={code}
                    className="flex items-center justify-between rounded-xl bg-slate-100 px-3.5 py-2.5 font-mono text-xs dark:bg-slate-950/60"
                  >
                    <span className="text-emerald-600 dark:text-emerald-300">
                      /{code}
                    </span>
                    <Copy className="h-3.5 w-3.5 text-slate-400" />
                  </div>
                ))}
              </div>
            </BentoCell>

            <BentoCell>
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <Users className="h-5 w-5" />
              </div>
              <h3 className="mt-4 text-lg font-semibold tracking-tight">
                Built for teams
              </h3>
              <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                Workspaces with owner, admin, member, and viewer roles.
              </p>
              <div className="mt-5 flex -space-x-2.5">
                {["D", "B", "A", "+5"].map((initial) => (
                  <span
                    key={initial}
                    className="flex h-9 w-9 items-center justify-center rounded-full border-2 border-white bg-gradient-to-br from-emerald-400 to-teal-500 text-xs font-bold text-slate-950 dark:border-slate-900"
                  >
                    {initial}
                  </span>
                ))}
              </div>
            </BentoCell>

            <BentoCell className="md:col-span-2">
              <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
                <div className="sm:w-2/5">
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                    <Code2 className="h-5 w-5" />
                  </div>
                  <h3 className="mt-4 text-lg font-semibold tracking-tight">
                    API-first
                  </h3>
                  <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                    Scoped API keys and a clean public REST API, so your tools
                    create links on their own.
                  </p>
                </div>
                <div className="flex-1 rounded-2xl bg-slate-100 p-4 font-mono text-xs leading-relaxed dark:bg-slate-950/80">
                  <p>
                    <span className="font-semibold text-emerald-600 dark:text-emerald-400">
                      POST
                    </span>{" "}
                    <span className="text-slate-700 dark:text-slate-300">
                      /api/v1/public/links
                    </span>
                  </p>
                  <p className="text-slate-500">
                    Authorization: Bearer{" "}
                    <span className="text-teal-600 dark:text-teal-300">
                      lp_live_&bull;&bull;&bull;
                    </span>
                  </p>
                  <p className="text-slate-500">
                    {"{ "}
                    <span className="text-emerald-600 dark:text-emerald-300">
                      &quot;destination_url&quot;
                    </span>
                    {" : "}
                    <span className="text-slate-700 dark:text-slate-300">
                      &quot;https://&hellip;&quot;
                    </span>
                    {" }"}
                  </p>
                </div>
              </div>
            </BentoCell>

            <BentoCell>
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <ShieldCheck className="h-5 w-5" />
              </div>
              <h3 className="mt-4 text-lg font-semibold tracking-tight">
                Privacy by design
              </h3>
              <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                Clicks are tracked without storing raw IP addresses — hashed and
                salted only.
              </p>
            </BentoCell>
          </div>
        </section>

        {/* Final CTA */}
        <section className="relative mx-auto mt-28 max-w-3xl">
          <FadeIn>
            <div className="rounded-3xl bg-gradient-to-r from-emerald-500/60 via-teal-500/40 to-emerald-500/60 p-px">
              <div className="rounded-[calc(1.5rem-1px)] bg-white px-8 py-12 text-center dark:bg-slate-950">
                <h2 className="text-3xl font-bold tracking-tight">
                  Ready to own your links?
                </h2>
                <p className="mx-auto mt-3 max-w-md text-slate-500 dark:text-slate-400">
                  Spin up LinkPulse locally in minutes — Go backend, Next.js
                  frontend, your PostgreSQL.
                </p>
                <div className="mt-7 flex justify-center gap-4">
                  <Link
                    href="/register"
                    className="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-emerald-500 to-teal-500 px-6 py-3 font-semibold text-slate-950 shadow-lg shadow-emerald-500/25 transition hover:from-emerald-400 hover:to-teal-400"
                  >
                    Get started
                    <ArrowRight className="h-4 w-4" />
                  </Link>
                  <Link
                    href="/login"
                    className="rounded-full border border-slate-300 px-6 py-3 font-semibold text-slate-700 transition hover:border-slate-400 dark:border-slate-700 dark:text-slate-300 dark:hover:border-slate-500"
                  >
                    Sign in
                  </Link>
                </div>
              </div>
            </div>
          </FadeIn>
        </section>
      </main>

      {/* Footer */}
      <footer className="relative border-t border-slate-200 dark:border-slate-800/60">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 px-6 py-8 text-sm text-slate-500 sm:flex-row">
          <p>&copy; 2026 LinkPulse &middot; Built with Go &amp; Next.js</p>
          <p>MIT License</p>
        </div>
      </footer>
    </div>
  );
}
