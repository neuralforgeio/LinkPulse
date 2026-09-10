import { Logo } from "@/components/brand/logo";
import { AuthScene } from "@/components/brand/auth-scene";
import { ThemeToggle } from "@/components/theme-toggle";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-dvh bg-white dark:bg-zinc-950">
      {/* Form side — theme-aware, no scrolling on desktop */}
      <main className="relative flex w-full flex-col lg:w-[46%]">
        <div className="flex items-center justify-between p-6 sm:p-8">
          <Logo />
          <ThemeToggle />
        </div>

        <div className="flex flex-1 items-center justify-center px-6 pb-8">
          <div className="w-full max-w-sm">{children}</div>
        </div>

        <p className="px-6 pb-6 text-center text-xs text-zinc-400 dark:text-zinc-600">
          &copy; 2026 LinkPulse &middot; MIT licensed &middot; Self-hosted
        </p>
      </main>

      {/* Scene side — always dark, like the reference photo */}
      <aside className="relative hidden flex-1 overflow-hidden bg-zinc-950 lg:block">
        <AuthScene />
      </aside>
    </div>
  );
}
