export function Logo({ variant = "auto" }: { variant?: "auto" | "light" }) {
  // "auto" adapts to the active theme; "light" forces the dark-bg look.
  const word =
    variant === "light" ? "text-white" : "text-zinc-900 dark:text-white";
  const pulse =
    variant === "light" ? "text-blue-400" : "text-blue-600 dark:text-blue-400";

  return (
    <span className="inline-flex items-center gap-2.5">
      <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-blue-600 to-indigo-600 shadow-lg shadow-blue-600/20">
        <svg
          viewBox="0 0 24 24"
          className="h-5 w-5"
          fill="none"
          stroke="#ffffff"
          strokeWidth={2.6}
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M3.5 13h3.5l2-6.5 4 13 2-6.5h5.5" />
        </svg>
      </span>
      <span className={`text-xl font-semibold tracking-tight ${word}`}>
        Link<span className={pulse}>Pulse</span>
      </span>
    </span>
  );
}
