import Link from "next/link";
import { Brand } from "../../../features/ui/Brand";

export default function Page() {
  return (
    <main className="legal-page">
      <Brand href="/register" />
      <section className="panel legal-card">
        <p className="eyebrow">Репет</p>
        <h1>Условия использования</h1>
        <p>
          Это рабочая версия условий для локального MVP. Перед production-запуском текст должен пройти юридическую проверку,
          особенно для сценариев с несовершеннолетними учениками.
        </p>
        <p>
          Пользователь отвечает за корректность учебных материалов, а платформа хранит только минимальные данные,
          необходимые для занятий, заданий и сообщений.
        </p>
        <Link className="button" href="/register">Вернуться к регистрации</Link>
      </section>
    </main>
  );
}

