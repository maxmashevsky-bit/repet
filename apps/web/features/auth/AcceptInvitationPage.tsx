"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { productApi } from "../../shared/api/client";

export function AcceptInvitationPage({ token }: { token: string }) {
  const [state, setState] = useState<"loading" | "guest" | "student" | "tutor">("loading");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    productApi.me().then((user) => setState(user.role === "student" ? "student" : "tutor"))
      .catch(() => setState("guest"));
  }, []);

  async function accept() {
    setBusy(true);
    setError("");
    try {
      await productApi.acceptInvitation(token);
      window.location.href = "/";
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось принять приглашение");
      setBusy(false);
    }
  }

  const returnTo = encodeURIComponent(`/invite/${token}`);
  return (
    <main className="legal-page">
      <Link href="/">Репет</Link>
      <h1>Приглашение на занятия</h1>
      {state === "loading" ? <p>Проверяем аккаунт...</p> : null}
      {state === "guest" ? <p>Войдите или создайте аккаунт ученика с тем email, на который отправлено приглашение.</p> : null}
      {state === "guest" ? <div className="actions"><Link className="button" href={`/login?returnTo=${returnTo}`}>Войти</Link><Link className="button secondary" href={`/register?returnTo=${returnTo}`}>Регистрация</Link></div> : null}
      {state === "tutor" ? <p>Приглашение может принять только ученик.</p> : null}
      {state === "student" ? <button type="button" disabled={busy} onClick={accept}>{busy ? "Подключаем..." : "Принять приглашение"}</button> : null}
      {error ? <p className="field-error">{error}</p> : null}
    </main>
  );
}
