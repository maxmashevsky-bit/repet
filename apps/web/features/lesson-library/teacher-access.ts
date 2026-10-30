import "server-only";
import { cookies } from "next/headers";

export async function teacherAccess(): Promise<"allowed" | "unauthorized" | "forbidden"> {
  const session = (await cookies()).get("repet_session");
  if (!session) return "unauthorized";
  const response = await fetch(`${process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080"}/api/v1/me`, {
    headers: { Cookie: `repet_session=${session.value}` },
    cache: "no-store"
  });
  if (response.status === 401) return "unauthorized";
  if (!response.ok) throw new Error("Не удалось проверить доступ к урокам");
  const user = await response.json();
  return user.role === "tutor" ? "allowed" : "forbidden";
}
