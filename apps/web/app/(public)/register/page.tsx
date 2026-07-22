import { Suspense } from "react";
import { RegisterPage } from "../../../features/auth/AuthPages";

export default function Page() {
  return <Suspense fallback={<main className="full-state">Загружаем регистрацию...</main>}><RegisterPage /></Suspense>;
}
