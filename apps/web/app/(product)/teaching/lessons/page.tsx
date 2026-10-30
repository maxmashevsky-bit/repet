import Link from "next/link";
import { lessonCatalog } from "../../../../features/lesson-library/catalog";
import { Icon } from "../../../../features/ui/Icon";

export default function LessonsPage() {
  return (
    <section className="lesson-library">
      <Link className="inline-link arrow-link" href="/"><Icon name="arrow" className="back" size={16} />К кабинету</Link>
      <header className="library-heading">
        <div><p className="library-eyebrow">Кабинет преподавателя</p><h1>Мастерская уроков</h1><p className="muted">Математика: от первого наблюдения до самостоятельного решения.</p></div>
        <span className="library-count"><Icon name="book" size={20} />Тем: {lessonCatalog.length}</span>
      </header>
      <div className="lesson-catalog">
        {lessonCatalog.map((lesson, index) => (
          <Link href={`/teaching/lessons/${lesson.slug}`} className="lesson-tile" key={lesson.slug}>
            <div className={`lesson-art lesson-art-${index % 4}`} aria-hidden="true"><span className="lesson-art-number">{String(index + 1).padStart(2, "0")}</span><strong>{lesson.mark}</strong><span className="lesson-art-rule" /></div>
            <div className="lesson-tile-body"><span className="lesson-topic">{lesson.topic}</span><h2>{lesson.title}</h2><p>{lesson.description}</p><span className="lesson-open">Открыть урок<Icon name="arrow" size={18} /></span></div>
          </Link>
        ))}
      </div>
    </section>
  );
}
