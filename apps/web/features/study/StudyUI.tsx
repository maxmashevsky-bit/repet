import { ApiError } from "../../shared/api/client";
import { LibraryError } from "../material-library/LibraryUI";

export function StudyError({ error, retry }: { error: unknown; retry?: () => void }) {
  const message = error instanceof ApiError && error.status === 409 ? "Операция конфликтует с сохранённой работой. Проверьте историю прохождений и повторите запрос."
    : error instanceof ApiError && error.status === 422 ? "Действие недоступно. Проверьте состав варианта, количество заданий и срок выполнения." : null;
  return <LibraryError error={message ? new Error(message) : error} retry={retry} />;
}
