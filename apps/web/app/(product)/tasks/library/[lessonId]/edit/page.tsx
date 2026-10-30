import { LessonEditorPage } from "../../../../../../features/material-library/LessonEditor";

export default async function Page({ params }: { params: Promise<{ lessonId: string }> }) {
  return <LessonEditorPage lessonID={(await params).lessonId} />;
}
