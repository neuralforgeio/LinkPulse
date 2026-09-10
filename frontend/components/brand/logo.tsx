export function Logo({ variant = "auto" }: { variant?: "auto" | "light" }) {
  // "auto" adapts to the active theme; "light" forces the dark-bg look
  // (used on the always-dark auth scene).
  const word =
    variant === "light" ? "text-white" : "text-slate-900 dark:text-white";
  const pulse =
    variant === "light"
      ? "from-emerald-300 to-teal-300"
      : "from-emerald-500 to-teal-500 dark:from-emerald-300 dark:to-teal-300";

  return (
    <span className="inline-flex items-center gap-2.5">
      <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-400 via-teal-400 to-cyan-400 shadow-lg shadow-emerald-500/25">
        <svg
          viewBox="0 0 24 24"
          className="h-5 w-5"
          fill="none"
          stroke="#022c22"
          strokeWidth={2.6}
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M3.5 13h3.5l2-6.5 4 13 2-6.5h5.5" />
        </svg>
      </span>
      <span className={`text-xl font-semibold tracking-tight ${word}`}>
        Link
        <span
          className={`bg-gradient-to-r ${pulse} bg-clip-text text-transparent`}
        >
          Pulse
        </span>
      </span>
    </span>
  );
}
