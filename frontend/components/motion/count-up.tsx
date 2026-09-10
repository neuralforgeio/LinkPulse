"use client";

import { useEffect, useState } from "react";
import { animate } from "framer-motion";

interface CountUpProps {
  to: number;
  duration?: number;
  className?: string;
}

/** Animates a number from 0 up to `to` on mount. */
export function CountUp({ to, duration = 1.8, className }: CountUpProps) {
  const [value, setValue] = useState(0);

  useEffect(() => {
    const controls = animate(0, to, {
      duration,
      ease: "easeOut",
      onUpdate: (latest) => setValue(Math.round(latest)),
    });
    return () => controls.stop();
  }, [to, duration]);

  return <span className={className}>{value.toLocaleString("en-US")}</span>;
}
