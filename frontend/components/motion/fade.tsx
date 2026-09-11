import type { ReactNode } from "react";

interface FadeInProps {
  children: ReactNode;
  delay?: number;
  /** Positive rises from below; negative drops from above. */
  y?: number;
  className?: string;
}

/**
 * Entrance animation: fade + rise. Implemented with pure CSS so content
 * is visible the moment styles load — even if JavaScript is slow or
 * fails to hydrate entirely. (Previously framer-motion held content at
 * opacity 0 until hydration, which on slow loads left pages blank.)
 */
export function FadeIn({
  children,
  delay = 0,
  y = 24,
  className,
}: FadeInProps) {
  const anim = y < 0 ? "fade-down" : "fade-up";
  return (
    <div
      className={`${anim}${className ? ` ${className}` : ""}`}
      style={{ animationDelay: `${delay}s` }}
    >
      {children}
    </div>
  );
}
