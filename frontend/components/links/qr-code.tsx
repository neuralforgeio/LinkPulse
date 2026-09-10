"use client";

import { useEffect, useState } from "react";
import QRCode from "qrcode";

interface QrCodeProps {
  /** The value to encode — usually a short URL. */
  value: string;
  size?: number;
}

/** Renders a QR code for the given value, with a skeleton while it renders. */
export function QrCode({ value, size = 180 }: QrCodeProps) {
  const [src, setSrc] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    QRCode.toDataURL(value, {
      width: size * 2, // rendered at 2x so it stays crisp on any screen
      margin: 2,
      color: { dark: "#022c22", light: "#ffffff" },
    })
      .then((url) => {
        if (!cancelled) setSrc(url);
      })
      .catch(() => {
        // Rendering failure is non-fatal: the URL text is always shown too.
      });
    return () => {
      cancelled = true;
    };
  }, [value, size]);

  if (!src) {
    return (
      <div
        style={{ width: size, height: size }}
        className="animate-pulse rounded-xl bg-slate-100 dark:bg-slate-800"
        aria-hidden="true"
      />
    );
  }

  return (
    // eslint-disable-next-line @next/next/no-img-element -- a data URL, not a remote image
    <img
      src={src}
      alt="QR code"
      width={size}
      height={size}
      className="rounded-xl"
    />
  );
}
