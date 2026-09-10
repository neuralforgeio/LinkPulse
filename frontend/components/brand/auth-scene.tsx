"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { TrendingUp } from "lucide-react";

interface FeedItem {
  id: number;
  code: string;
  referrer: string;
  browser: string;
  device: string;
}

const SAMPLES: Omit<FeedItem, "id">[] = [
  { code: "promo2026", referrer: "instagram.com", browser: "Chrome", device: "Android" },
  { code: "launch-day", referrer: "x.com", browser: "Safari", device: "iOS" },
  { code: "docs-guide", referrer: "google.com", browser: "Firefox", device: "Desktop" },
  { code: "careers", referrer: "linkedin.com", browser: "Edge", device: "Windows" },
  { code: "q1-webinar", referrer: "newsletter", browser: "Chrome", device: "macOS" },
];

const SPARK = [30, 45, 38, 60, 52, 70, 58, 76, 64, 82, 74, 90];

const REFERRERS = [
  { name: "instagram.com", pct: 62 },
  { name: "google.com", pct: 38 },
  { name: "x.com", pct: 22 },
];

const glass =
  "rounded-2xl bg-white/[0.06] ring-1 ring-white/10 backdrop-blur-xl shadow-2xl shadow-black/40";

/**
 * The right-hand side of the auth pages: a dark gradient scene with
 * floating glass analytics cards (simulated demo data until real click
 * tracking lands in Milestone 4).
 */
export function AuthScene() {
  const [clicks, setClicks] = useState(12418);
  const [feed, setFeed] = useState<FeedItem[]>([]);
  const nextId = useRef(1);

  useEffect(() => {
    const push = () => {
      const sample = SAMPLES[Math.floor(Math.random() * SAMPLES.length)];
      setFeed((prev) => [{ ...sample, id: nextId.current++ }, ...prev].slice(0, 3));
      setClicks((prev) => prev + 1 + Math.floor(Math.random() * 3));
    };
    for (let i = 0; i < 3; i++) push();
    const timer = setInterval(push, 3000);
    return () => clearInterval(timer);
  }, []);

  return (
    <div className="absolute inset-0">
      <div className="dot-grid absolute inset-0" aria-hidden="true" />
      <div
        className="absolute -right-24 -top-24 h-[26rem] w-[26rem] rounded-full bg-emerald-500/20 blur-[130px] animate-drift"
        aria-hidden="true"
      />
      <div
        className="absolute -bottom-32 -left-16 h-[22rem] w-[22rem] rounded-full bg-teal-600/20 blur-[110px] animate-drift-reverse"
        aria-hidden="true"
      />

      <div className="relative flex h-full items-center justify-center p-12">
        <div className="relative h-[520px] w-full max-w-xl">
          {/* Clicks counter card */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1, y: [0, -9, 0] }}
            transition={{
              opacity: { duration: 0.7, delay: 0.2 },
              scale: { duration: 0.7, delay: 0.2 },
              y: { duration: 5, repeat: Infinity, ease: "easeInOut" },
            }}
            className={`absolute left-0 top-6 w-64 -rotate-2 p-5 ${glass}`}
          >
            <p className="text-xs font-medium uppercase tracking-wider text-slate-400">
              Clicks today
            </p>
            <p className="mt-1 bg-gradient-to-r from-emerald-300 to-teal-300 bg-clip-text text-3xl font-bold tracking-tight text-transparent">
              {clicks.toLocaleString("en-US")}
            </p>
            <div className="mt-4 flex h-10 items-end gap-1" aria-hidden="true">
              {SPARK.map((h, i) => (
                <div
                  key={i}
                  style={{ height: `${h}%` }}
                  className={`w-full rounded-sm ${
                    i === SPARK.length - 1 ? "bg-emerald-400" : "bg-white/15"
                  }`}
                />
              ))}
            </div>
          </motion.div>

          {/* Live feed card */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1, y: [0, 10, 0] }}
            transition={{
              opacity: { duration: 0.7, delay: 0.35 },
              scale: { duration: 0.7, delay: 0.35 },
              y: { duration: 6, repeat: Infinity, ease: "easeInOut" },
            }}
            className={`absolute right-0 top-24 w-72 rotate-1 p-4 ${glass}`}
          >
            <div className="flex items-center justify-between px-1 pb-2.5">
              <div className="flex items-center gap-2 text-xs font-medium text-slate-400">
                <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse-ring" />
                LIVE — click stream
              </div>
              <span className="rounded-full bg-white/5 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider text-slate-500">
                demo
              </span>
            </div>
            <ul className="space-y-2">
              <AnimatePresence initial={false}>
                {feed.map((item) => (
                  <motion.li
                    key={item.id}
                    layout
                    initial={{ opacity: 0, y: -12 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, scale: 0.94 }}
                    transition={{ duration: 0.3 }}
                    className="flex items-center justify-between rounded-xl bg-white/[0.04] px-3.5 py-2.5 ring-1 ring-white/5"
                  >
                    <div className="min-w-0">
                      <p className="truncate font-mono text-sm text-emerald-300">
                        /{item.code}
                      </p>
                      <p className="mt-0.5 truncate text-xs text-slate-500">
                        from {item.referrer}
                      </p>
                    </div>
                    <div className="ml-3 shrink-0 text-right text-[11px] leading-snug text-slate-500">
                      <p>{item.browser}</p>
                      <p>{item.device}</p>
                    </div>
                  </motion.li>
                ))}
              </AnimatePresence>
            </ul>
          </motion.div>

          {/* Referrers card */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1, y: [0, -7, 0] }}
            transition={{
              opacity: { duration: 0.7, delay: 0.5 },
              scale: { duration: 0.7, delay: 0.5 },
              y: { duration: 5.5, repeat: Infinity, ease: "easeInOut" },
            }}
            className={`absolute bottom-8 left-14 w-64 rotate-2 p-5 ${glass}`}
          >
            <p className="text-xs font-medium uppercase tracking-wider text-slate-400">
              Top referrers
            </p>
            <div className="mt-3 space-y-3">
              {REFERRERS.map((r, i) => (
                <div key={r.name}>
                  <div className="flex items-center justify-between text-xs text-slate-400">
                    <span>{r.name}</span>
                    <span>{r.pct}%</span>
                  </div>
                  <div className="mt-1.5 h-1.5 rounded-full bg-white/10">
                    <motion.div
                      initial={{ width: 0 }}
                      animate={{ width: `${r.pct}%` }}
                      transition={{ duration: 0.9, delay: 0.7 + i * 0.15, ease: "easeOut" }}
                      className="h-full rounded-full bg-gradient-to-r from-emerald-400 to-teal-400"
                    />
                  </div>
                </div>
              ))}
            </div>
          </motion.div>

          {/* Floating stat chip */}
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1, y: [0, 8, 0] }}
            transition={{
              opacity: { duration: 0.6, delay: 0.65 },
              scale: { duration: 0.6, delay: 0.65 },
              y: { duration: 4.5, repeat: Infinity, ease: "easeInOut" },
            }}
            className="absolute -right-2 bottom-32 flex items-center gap-2 rounded-full bg-white/[0.06] px-3.5 py-2 text-xs font-medium text-emerald-300 ring-1 ring-white/10 backdrop-blur-xl"
          >
            <TrendingUp className="h-3.5 w-3.5" />
            +312 clicks this week
          </motion.div>
        </div>
      </div>

      <p className="absolute inset-x-0 bottom-5 text-center text-[11px] text-slate-600">
        Live preview — demo data. Real analytics arrive in Milestone 4.
      </p>
    </div>
  );
}
