import Link from "next/link";
import { Brand } from "../../../features/ui/Brand";

export default function Page() {
  return (
    <main className="legal-page">
      <Brand href="/register" />
      <section className="panel legal-card">
        <p className="eyebrow">Репет</p>
        <h1>Политика конфиденциальности</h1>
        <p>
          «Репет» проектируется по принципу минимизации данных: ученик не имеет публичного профиля,
          а общение доступно только после подтверждённой связи с преподавателем.
        </p>
        <p>
          Эта страница является MVP-заготовкой. Юридическая проверка обязательна до работы с реальными пользователями.
        </p>
        <Link className="button" href="/register">Вернуться к регистрации</Link>
      </section>
    </main>
  );
}

