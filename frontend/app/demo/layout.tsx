import type { Metadata } from "next";

// The demo shows sample data — keep search engines away from indexing it
// as the real product.
export const metadata: Metadata = {
  title: "Demo — LinkPulse",
  description:
    "Interactive demo of the LinkPulse dashboard with sample data. No account needed.",
  robots: { index: false, follow: false },
};

export default function DemoLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
