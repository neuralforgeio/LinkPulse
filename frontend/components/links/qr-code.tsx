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
      width: size * 2,
      margin: 2,
      color: { dark: "#000000", light: "#ffffff" },
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
        className="animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800"
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
