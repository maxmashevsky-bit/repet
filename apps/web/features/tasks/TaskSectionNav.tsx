"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

export function TaskSectionNav() {
  const path = usePathname(), library = path.startsWith("/tasks/library"), study = path.startsWith("/tasks/study"), progress = path.startsWith("/tasks/progress"), review=path.startsWith("/tasks/review");
  return <nav className="task-section-nav" aria-label="Разделы заданий">
    <Link href="/tasks" className={!library && !study && !progress && !review ? "active" : ""} aria-current={!library && !study && !progress && !review ? "page" : undefined}>Назначенные задания</Link>
    <Link href="/tasks/library" className={library ? "active" : ""} aria-current={library ? "page" : undefined}>Библиотека</Link>
    <Link href="/tasks/study" className={study ? "active" : ""} aria-current={study ? "page" : undefined}>Тренировки и пробники</Link>
    <Link href="/tasks/progress" className={progress ? "active" : ""} aria-current={progress ? "page" : undefined}>Успеваемость</Link>
    <Link href="/tasks/review" className={review?"active":""} aria-current={review?"page":undefined}>Повторение</Link>
  </nav>;
}
