"use client";

import { useId, type InputHTMLAttributes, type ReactNode } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  hint?: string;
  /** Optional element rendered inside the field on the right. */
  trailing?: ReactNode;
}

export function Input({ label, hint, trailing, ...props }: InputProps) {
  const id = useId();

  return (
    <div>
      <label
        htmlFor={id}
        className="block text-sm font-medium text-slate-700 dark:text-slate-300"
      >
        {label}
      </label>
      <div className="relative mt-1.5">
        <input
          id={id}
          {...props}
          className={`w-full rounded-xl border border-transparent bg-slate-100 px-4 py-3 text-sm text-slate-900 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-500 focus:bg-white focus:outline-none focus:ring-4 focus:ring-emerald-500/10 dark:bg-slate-900 dark:text-white dark:placeholder:text-slate-500 dark:focus:border-emerald-500 dark:focus:bg-slate-950 ${
            trailing ? "pr-12" : ""
          }`}
        />
        {trailing && (
          <div className="absolute inset-y-0 right-0 flex items-center pr-4">
            {trailing}
          </div>
        )}
      </div>
      {hint && (
        <p className="mt-1.5 text-xs text-slate-500 dark:text-slate-500">
          {hint}
        </p>
      )}
    </div>
  );
}
