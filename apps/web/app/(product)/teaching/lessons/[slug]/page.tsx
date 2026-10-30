import Link from "next/link";
import { notFound } from "next/navigation";
import { findLesson, lessonCatalog } from "../../../../../features/lesson-library/catalog";
import { Icon } from "../../../../../features/ui/Icon";

export default async function LessonPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const lesson = findLesson(slug);
  if (!lesson) notFound();
  const index = lessonCatalog.findIndex((item) => item.slug === slug);
  const next = lessonCatalog[index + 1];
  return (
    <section className="lesson-reader">
      <div className="lesson-reader-bar"><Link className="inline-link arrow-link" href="/teaching/lessons"><Icon name="arrow" className="back" size={17} />Все уроки</Link><span className="muted">Урок {index + 1} из {lessonCatalog.length}</span>{next ? <Link className="inline-link arrow-link" href={`/teaching/lessons/${next.slug}`}>Следующий<Icon name="arrow" size={17} /></Link> : <span />}</div>
      <header className="lesson-reader-heading"><p className="library-eyebrow">{lesson.topic}</p><h1>{lesson.title}</h1></header>
      <div className="actions workshop-actions"><Link className="button" href={`/tasks/library/html/import?workshop=${lesson.slug}`}><Icon name="plus" size={18}/>Добавить в мою библиотеку</Link><Link className="button secondary" href={`/tasks/library/html/import?workshop=${lesson.slug}&convert=1`}><Icon name="task" size={18}/>Создать проверяемые упражнения</Link></div>
      <iframe className="lesson-frame" title={lesson.title} src={`/lesson-content/${lesson.slug}${lesson.section ? `#${lesson.section}` : ""}`} sandbox="allow-scripts allow-same-origin allow-modals allow-downloads" referrerPolicy="no-referrer" />
    </section>
  );
}
