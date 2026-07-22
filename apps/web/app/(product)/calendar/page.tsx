import { Suspense } from "react";
import { CalendarPage } from "../../../features/calendar/CalendarPage";

export default function Page() {
  return <Suspense fallback={<div className="full-state">Загружаем календарь...</div>}><CalendarPage /></Suspense>;
}
