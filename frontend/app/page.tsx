import Link from "next/link";

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-slate-950 p-6 text-white">
      <p className="text-sm font-semibold uppercase tracking-widest text-indigo-400">
        LinkPulse
      </p>
      <h1 className="mt-3 max-w-2xl text-center text-4xl font-bold sm:text-5xl">
        Short links. Real analytics. Workspace kamu sendiri.
      </h1>
      <p className="mt-4 max-w-xl text-center text-slate-400">
        Self-hosted URL shortener dengan click tracking, workspace multi-tenant,
        dan API publik untuk developer.
      </p>
      <div className="mt-8 flex gap-4">
        <Link
          href="/login"
          className="rounded-lg bg-indigo-600 px-6 py-3 font-semibold transition hover:bg-indigo-500"
        >
          Login
        </Link>
        <Link
          href="/login"
          className="rounded-lg border border-slate-700 px-6 py-3 font-semibold transition hover:bg-slate-800"
        >
          Register — segera
        </Link>
      </div>
    </main>
  );
}
