"use client";
import { Select } from "../ui/Select";


import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useEffect, useOptimistic, useState, useTransition } from "react";
import { allCatalogPages, catalogApi, loadLibraryStructure, type CatalogPage, type LearningLesson, type Course, type Topic, type CatalogQuery } from "../../shared/api/catalog";
import { useWorkspace } from "../app-shell/AppShell";
import { TaskSectionNav } from "../tasks/TaskSectionNav";
import { Icon } from "../ui/Icon";
import { classLabel, countLabel, difficultyLabels, LessonCover, LibraryError, LibraryLoading, statusLabels, topicLabel, trackLabels } from "./LibraryUI";

const pageSize = 12;
const normalized = (value: string) => value.toLocaleLowerCase("ru").replaceAll("ё", "е").replace(/\s+/g, " ").trim();

export function LibraryPage() {
  const { dashboard } = useWorkspace();
  const teacher = dashboard.user.role === "tutor";
  const routeSearch = useSearchParams();
  const [query, setQuery] = useOptimistic(routeSearch.toString());
  const search = new URLSearchParams(query);
  const [, startTransition] = useTransition();
  const router = useRouter();
  const [retry, setRetry] = useState(0);
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [structure, setStructure] = useState<{ courses: Course[]; topics: Topic[]; error?: unknown } | null>(null);
  const [result, setResult] = useState<{ key: string; page?: CatalogPage<LearningLesson>; error?: unknown }>({ key: "" });
  const page = Math.max(1, Math.min(8334, Math.trunc(Number(search.get("page"))) || 1));
  const q = search.get("q") ?? "";
  const filter: CatalogQuery = {
    q, grade: search.get("grade") ?? "", difficulty: search.get("difficulty") ?? "", track: search.get("track") ?? "",
    course_id: search.get("course_id") ?? "", topic_id: search.get("topic_id") ?? "",
    status: teacher ? search.get("status") ?? "" : "", author_id: teacher && search.get("mine") === "1" ? dashboard.user.id : ""
  };
  const filterKey = JSON.stringify(filter);
  const requestKey = `${filterKey}:${page}:${retry}`;
  const loading = result.key !== requestKey;

  useEffect(() => {
    const controller = new AbortController();
    loadLibraryStructure(controller.signal).then((data) => { if (!controller.signal.aborted) setStructure(data); }).catch((error: unknown) => { if (!controller.signal.aborted) setStructure({ courses: [], topics: [], error }); });
    return () => controller.abort();
  }, [retry]);

  useEffect(() => {
    const controller = new AbortController();
    const values: CatalogQuery = JSON.parse(filterKey);
    async function load() {
      if (values.q?.trim()) {
        // API q searches body as well. Filter every matching page for an exact title search and correct totals.
        const candidates = await allCatalogPages((offset) => catalogApi.lessons({ ...values, limit: 100, offset }, controller.signal));
        const items = candidates.filter((lesson) => normalized(lesson.version.title).includes(normalized(values.q ?? "")));
        return { items: items.slice((page - 1) * pageSize, page * pageSize), total: items.length, limit: pageSize, offset: (page - 1) * pageSize };
      }
      return catalogApi.lessons({ ...values, limit: pageSize, offset: (page - 1) * pageSize }, controller.signal);
    }
    load().then((data) => { if (!controller.signal.aborted) setResult({ key: requestKey, page: data }); }).catch((error: unknown) => { if (!controller.signal.aborted) setResult({ key: requestKey, error }); });
    return () => controller.abort();
  }, [filterKey, page, requestKey]);

  function change(values: Record<string, string>, paginate = false) {
    const params = new URLSearchParams(search.toString());
    if (!paginate) params.delete("page");
    Object.entries(values).forEach(([key, value]) => { if (value) params.set(key, value); else params.delete(key); });
    startTransition(() => {
      setQuery(params.toString());
      router.replace(`/tasks/library${params.size ? `?${params}` : ""}`, { scroll: false });
    });
  }
  function find(event: FormEvent<HTMLFormElement>) { event.preventDefault(); change({ q: String(new FormData(event.currentTarget).get("q") ?? "").trim() }); }
  const topics = structure?.topics.filter((topic) => !filter.course_id || topic.course_id === filter.course_id) ?? [];
  const total = result.page?.total ?? 0;
  return <section className="material-library">
    <TaskSectionNav />
    <header className="material-page-heading"><div><p className="material-kicker">Учебные материалы</p><h1>Библиотека</h1></div>{teacher ? <Link className="button" href="/tasks/library/new"><Icon name="plus" size={19} />Создать урок</Link> : null}</header>
    <nav className="task-section-nav" aria-label="Формат материалов"><Link href="/tasks/library" className="active">Интерактивные упражнения</Link><Link href="/tasks/library/html">HTML-уроки</Link>{teacher?<><Link href="/tasks/library/pdf">Импорт PDF</Link><Link href="/tasks/library/bank">Банк упражнений</Link></>:null}</nav>
    <form className="material-search" onSubmit={find} role="search"><Icon name="search" size={21} /><input key={q} name="q" defaultValue={q} maxLength={200} aria-label="Название урока" placeholder="Найти урок по названию" /><button type="submit">Найти<Icon name="arrow" size={18} /></button></form>
    <button className="secondary material-filter-toggle" type="button" aria-expanded={filtersOpen} aria-controls="library-filters" onClick={() => setFiltersOpen((open) => !open)}>Фильтры<Icon name="chevron" size={17} /></button>
    <div className="material-library-layout">
      <aside id="library-filters" className={`material-filters material-collapsible-filters${filtersOpen ? " is-open" : ""}`} aria-label="Фильтры материалов"><div className="material-filter-heading"><h2>Фильтры</h2><Link className="inline-link" href="/tasks/library">Сбросить</Link></div>
        <label>Предмет<Select aria-label="Предмет" value="math" disabled><option value="math">Математика</option></Select></label>
        <label>Класс<Select aria-label="Класс" value={filter.grade} onChange={(e) => change({ grade: e.target.value })}><option value="">Все классы</option>{Array.from({ length: 11 }, (_, i) => <option key={i} value={i + 1}>{i + 1} класс</option>)}</Select></label>
        <label>Курс<Select aria-label="Курс" value={filter.course_id} onChange={(e) => change({ course_id: e.target.value, topic_id: "" })} disabled={!structure || Boolean(structure.error)}><option value="">Все курсы</option>{structure?.courses.map((course) => <option key={course.id} value={course.id}>{course.title}</option>)}</Select></label>
        <label>Тема<Select aria-label="Тема" value={filter.topic_id} onChange={(e) => change({ topic_id: e.target.value })} disabled={!structure || Boolean(structure.error)}><option value="">Все темы</option>{topics.map((topic) => <option key={topic.id} value={topic.id}>{topicLabel(topic, structure?.topics ?? [])}</option>)}</Select></label>
        <label>Тип подготовки<Select aria-label="Тип подготовки" value={filter.track} onChange={(e) => change({ track: e.target.value })}><option value="">Все направления</option>{Object.entries(trackLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</Select></label>
        <label>Сложность<Select aria-label="Сложность" value={filter.difficulty} onChange={(e) => change({ difficulty: e.target.value })}><option value="">Любая сложность</option>{difficultyLabels.slice(1).map((label, i) => <option key={label} value={i + 1}>{label}</option>)}</Select></label>
        {teacher ? <><label>Статус<Select aria-label="Статус" value={filter.status} onChange={(e) => change({ status: e.target.value })}><option value="">Все статусы</option>{Object.entries(statusLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</Select></label><label className="check-row"><input type="checkbox" checked={search.get("mine") === "1"} onChange={(e) => change({ mine: e.target.checked ? "1" : "" })} />Только мои материалы</label></> : null}
        {structure?.error ? <LibraryError error={structure.error} retry={() => setRetry((n) => n + 1)} /> : null}
      </aside>
      <div className="material-results" aria-live="polite">
        <div className="material-results-heading"><h2>Материалы</h2>{!loading && result.page ? <span>{countLabel(total, ["материал", "материала", "материалов"])}</span> : null}</div>
        {loading ? <LibraryLoading /> : result.error ? <LibraryError error={result.error} retry={() => setRetry((n) => n + 1)} /> : !result.page?.items.length ? <div className="material-empty"><Icon name="book" size={34} /><h2>{total ? "На этой странице нет материалов" : "Уроков не найдено"}</h2><p>{q || Object.values(filter).some(Boolean) ? "Попробуйте изменить запрос или фильтры." : "Доступные материалы появятся здесь после публикации."}</p>{total ? <button type="button" onClick={() => change({ page: "1" }, true)}>На первую страницу</button> : teacher ? <Link className="button secondary" href="/tasks/library/new">Создать урок</Link> : null}</div> : <div className="material-grid">{result.page.items.map((lesson) => {
          const topic = structure?.topics.find((item) => item.id === lesson.version.topic_id);
          return <Link className="material-card" href={`/tasks/library/${lesson.id}`} key={lesson.id}>
            <LessonCover track={lesson.version.track} course={structure?.courses.find((item) => item.id === lesson.course_id)?.title ?? "Математика"} />
            <div className="material-card-body"><div className="material-card-labels"><span>{classLabel(lesson.version.grade_min, lesson.version.grade_max)}</span>{lesson.status !== "published" ? <span className={`material-status material-status-${lesson.status}`}>{statusLabels[lesson.status]}</span> : null}</div><p className="material-topic">{topic?.title ?? trackLabels[lesson.version.track]}</p><h3>{lesson.version.title}</h3>{lesson.version.summary ? <p className="material-card-summary">{lesson.version.summary}</p> : null}<div className="material-card-meta"><span><Icon name="task" size={16} />{countLabel(lesson.version.exercises.length)}</span><span>{difficultyLabels[lesson.version.difficulty]}</span></div><div className="material-card-open">Открыть урок<Icon name="arrow" size={18} /></div></div>
          </Link>;
        })}</div>}
        {!loading && total > pageSize ? <nav className="material-pagination" aria-label="Страницы библиотеки"><button type="button" className="secondary" disabled={page <= 1} onClick={() => change({ page: String(page - 1) }, true)}><Icon name="arrow" className="back" size={17} />Назад</button><span>{page} / {Math.ceil(total / pageSize)}</span><button type="button" className="secondary" disabled={page * pageSize >= total} onClick={() => change({ page: String(page + 1) }, true)}>Далее<Icon name="arrow" size={17} /></button></nav> : null}
      </div>
    </div>
  </section>;
}
