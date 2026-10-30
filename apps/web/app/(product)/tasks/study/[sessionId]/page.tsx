import { StudySessionPage } from "../../../../../features/study/StudySessionPage";
export default async function Page({ params }: { params: Promise<{ sessionId: string }> }) { const { sessionId } = await params; return <StudySessionPage sessionID={sessionId} />; }
