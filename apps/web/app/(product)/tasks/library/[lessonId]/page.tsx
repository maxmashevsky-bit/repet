import { MaterialLessonPage } from "../../../../../features/material-library/LessonPage";

export default async function Page({ params }: { params: Promise<{ lessonId: string }> }) {
  return <MaterialLessonPage lessonID={(await params).lessonId} />;
}
