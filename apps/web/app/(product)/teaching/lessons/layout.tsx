import { notFound, redirect } from "next/navigation";
import { teacherAccess } from "../../../../features/lesson-library/teacher-access";

export default async function LessonsLayout({ children }: { children: React.ReactNode }) {
  const access = await teacherAccess();
  if (access === "unauthorized") redirect("/login?returnTo=/teaching/lessons");
  if (access !== "allowed") notFound();
  return children;
}
