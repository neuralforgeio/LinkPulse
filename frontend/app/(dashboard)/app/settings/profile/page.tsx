"use client";

import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { Check, Eye, EyeOff } from "lucide-react";
import { api, ApiError } from "@/lib/api/client";
import { useAuth } from "@/components/auth/auth-provider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FadeIn } from "@/components/motion/fade";

interface UserOut {
  id: string;
  name: string;
  email: string;
}

export default function ProfilePage() {
  const { user, reload } = useAuth();

  // --- Name form ---
  const [name, setName] = useState(user?.name ?? "");
  const [nameSaved, setNameSaved] = useState(false);
  const [nameError, setNameError] = useState<string | null>(null);

  const nameMutation = useMutation({
    mutationFn: () =>
      api<UserOut>("/api/v1/auth/me", {
        method: "PATCH",
        body: JSON.stringify({ name: name.trim() }),
      }),
    onSuccess: () => {
      setNameSaved(true);
      setNameError(null);
      reload(); // sidebar user name updates immediately
      setTimeout(() => setNameSaved(false), 2500);
    },
    onError: (err) =>
      setNameError(
        err instanceof ApiError ? err.message : "Something went wrong.",
      ),
  });

  // --- Password form ---
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPasswords, setShowPasswords] = useState(false);
  const [pwSaved, setPwSaved] = useState(false);
  const [pwError, setPwError] = useState<string | null>(null);

  const passwordMutation = useMutation({
    mutationFn: (body: string) =>
      api<UserOut>("/api/v1/auth/me", { method: "PATCH", body }),
    onSuccess: () => {
      setPwSaved(true);
      setPwError(null);
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setTimeout(() => setPwSaved(false), 5000);
    },
    onError: (err) =>
      setPwError(
        err instanceof ApiError ? err.message : "Something went wrong.",
      ),
  });

  function onNameSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (name.trim().length < 2) {
      setNameError("Name must be at least 2 characters.");
      return;
    }
    setNameError(null);
    nameMutation.mutate();
  }

  function onPasswordSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (
      newPassword.length < 8 ||
      !/[A-Za-z]/.test(newPassword) ||
      !/[0-9]/.test(newPassword)
    ) {
      setPwError(
        "New password must be at least 8 characters with letters and numbers.",
      );
      return;
    }
    if (newPassword !== confirmPassword) {
      setPwError("New passwords do not match.");
      return;
    }
    setPwError(null);
    // The API requires the name field; send the current one unchanged.
    passwordMutation.mutate(
      JSON.stringify({
        name: (user?.name ?? name).trim(),
        old_password: oldPassword,
        new_password: newPassword,
      }),
    );
  }

  return (
    <div className="max-w-xl">
      <FadeIn>
        <h2 className="text-lg font-semibold tracking-tight text-zinc-900 dark:text-white">
          Profile
        </h2>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Your account details.
        </p>
      </FadeIn>

      {/* Name + email */}
      <FadeIn delay={0.08}>
        <form
          onSubmit={onNameSubmit}
          noValidate
          className="mt-6 space-y-5 rounded-xl border border-zinc-200 bg-white p-6 sm:p-8 dark:border-zinc-800 dark:bg-zinc-900"
        >
          <Input
            label="Name"
            placeholder="Dearly"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <Input
            label="Email (read-only)"
            value={user?.email ?? ""}
            disabled
            hint="The email cannot be changed in this version."
          />

          {nameError && (
            <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
              {nameError}
            </p>
          )}
          {nameSaved && (
            <p className="flex items-center gap-2 rounded-lg bg-emerald-500/10 px-4 py-3 text-sm text-emerald-600 dark:text-emerald-400">
              <Check className="h-4 w-4" />
              Name saved.
            </p>
          )}

          <Button type="submit" loading={nameMutation.isPending}>
            Save name
          </Button>
        </form>
      </FadeIn>

      {/* Password change */}
      <FadeIn delay={0.16}>
        <form
          onSubmit={onPasswordSubmit}
          noValidate
          className="mt-6 space-y-5 rounded-xl border border-zinc-200 bg-white p-6 sm:p-8 dark:border-zinc-800 dark:bg-zinc-900"
        >
          <div>
            <h3 className="font-semibold text-zinc-900 dark:text-white">
              Change password
            </h3>
            <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
              Other signed-in devices will be signed out. This device stays.
            </p>
          </div>

          <Input
            label="Current password"
            type={showPasswords ? "text" : "password"}
            autoComplete="current-password"
            placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            trailing={
              <button
                type="button"
                onClick={() => setShowPasswords((v) => !v)}
                aria-label={showPasswords ? "Hide passwords" : "Show passwords"}
                className="text-zinc-400 transition hover:text-zinc-600 dark:hover:text-zinc-300"
              >
                {showPasswords ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            }
          />
          <Input
            label="New password"
            type={showPasswords ? "text" : "password"}
            autoComplete="new-password"
            placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
            hint="At least 8 characters, with letters and numbers."
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
          />
          <Input
            label="Confirm new password"
            type={showPasswords ? "text" : "password"}
            autoComplete="new-password"
            placeholder="&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
          />

          {pwError && (
            <p className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:bg-rose-500/10 dark:text-rose-400">
              {pwError}
            </p>
          )}
          {pwSaved && (
            <p className="flex items-center gap-2 rounded-lg bg-emerald-500/10 px-4 py-3 text-sm text-emerald-600 dark:text-emerald-400">
              <Check className="h-4 w-4" />
              Password updated. Other devices have been signed out.
            </p>
          )}

          <Button type="submit" loading={passwordMutation.isPending}>
            Update password
          </Button>
        </form>
      </FadeIn>
    </div>
  );
}
