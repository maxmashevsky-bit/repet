import { readFile } from "node:fs/promises";
import path from "node:path";
import { findLesson } from "../../../features/lesson-library/catalog";
import { teacherAccess } from "../../../features/lesson-library/teacher-access";
import { lessonTheme } from "../../../features/lesson-library/theme";

export const runtime = "nodejs";

export async function GET(_request: Request, { params }: { params: Promise<{ slug: string }> }) {
  const access = await teacherAccess();
  if (access !== "allowed") return new Response("Нет доступа", { status: access === "unauthorized" ? 401 : 403 });
  const lesson = findLesson((await params).slug);
  if (!lesson) return new Response("Урок не найден", { status: 404 });
  const html = await readFile(path.join(process.cwd(), "lesson-content", lesson.file), "utf8");
  return new Response(html.replace(/<\/head>/i, `${lessonTheme}</head>`), { headers: {
    "Content-Type": "text/html; charset=utf-8",
    "Cache-Control": "private, no-store",
    "X-Content-Type-Options": "nosniff",
    "Content-Security-Policy": "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'"
  } });
}
