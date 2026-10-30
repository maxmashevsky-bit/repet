"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { catalogApi, allCatalogPages, type LearningLesson, type Course, type Topic } from "../../shared/api/catalog";
import { useWorkspace } from "../app-shell/AppShell";
import { TaskSectionNav } from "../tasks/TaskSectionNav";
import { Icon } from "../ui/Icon";
import { LessonExercises } from "../exercise-engine/LessonExercises";
import { LearningPlayer } from "../learning-player/LearningPlayer";
import { ApiError } from "../../shared/api/client";
import { learningApi, type LearningState } from "../../shared/api/learning";
import { parseLearningState } from "../exercise-engine/learning-schema";
import { BankRegister } from "../exercise-bank/BankRegister";
import { classLabel, countLabel, difficultyLabels, LessonArticle, LessonCover, LibraryError, LibraryLoading, statusLabels, topicLabel, trackLabels } from "./LibraryUI";

export function MaterialLessonPage({ lessonID }: { lessonID: string }) {
  const { dashboard } = useWorkspace();
  const [retry, setRetry] = useState(0);
  const key = `${lessonID}:${retry}:${dashboard.user.id}`;
  const [state, setState] = useState<{ key: string; error?: unknown; lesson?: LearningLesson; draft?: LearningLesson; course?: Course; topic?: string; learning?: LearningState }>({ key: "" });
  useEffect(() => {
    const controller = new AbortController();
    async function load() {
      const draft = await catalogApi.lesson(lessonID, controller.signal);
      let lesson = draft;
      let learning: LearningState | undefined;
      if(dashboard.user.role==="student"){
        try{learning=parseLearningState(await learningApi.state(lessonID,"current",controller.signal));lesson={...draft,version:learning.version};}catch(error){if(!(error instanceof ApiError)||error.status!==404)throw error;}
      }
      if (draft.author_id === dashboard.user.id && draft.status === "published") {
        const versions = await catalogApi.versions(lessonID, controller.signal);
        if (!versions.items.length) throw new Error("Опубликованная версия не найдена.");
        lesson = await catalogApi.version(lessonID, versions.items[0].number, controller.signal);
      }
      const [course, topics] = await Promise.all([catalogApi.course(lesson.course_id, controller.signal), allCatalogPages<Topic>((offset) => catalogApi.topics(lesson.course_id, { limit: 100, offset }, controller.signal))]);
      const topic = topics.find((item) => item.id === lesson.version.topic_id);
      return { lesson, draft, course, learning, topic: topic ? topicLabel(topic, topics) : undefined };
    }
    load().then((data) => { if (!controller.signal.aborted) setState({ key, ...data }); }).catch((error: unknown) => { if (!controller.signal.aborted) setState({ key, error }); });
    return () => controller.abort();
  }, [lessonID, dashboard.user.id, dashboard.user.role, key]);
  const owner = state.draft?.author_id === dashboard.user.id && dashboard.user.role === "tutor";
  const lesson = state.lesson;
  return <section className="material-reader"><TaskSectionNav /><Link className="inline-link arrow-link material-back" href="/tasks/library"><Icon name="arrow" className="back" size={17} />В библиотеку</Link>
    {state.key !== key ? <LibraryLoading /> : state.error ? <LibraryError error={state.error} retry={() => setRetry((n) => n + 1)} /> : lesson ? <>
      <header className="material-page-heading"><div><p className="material-kicker">{state.course?.title}</p><h1>{lesson.version.title}</h1></div>{owner ? <Link className="button secondary" href={`/tasks/library/${lesson.id}/edit`}><Icon name="task" size={18} />Редактировать</Link> : null}</header>
      {owner && (lesson.status !== "published" || state.draft?.has_unpublished_changes) ? <div className="material-notice" role="status">{lesson.status === "archived" ? "Материал в архиве и недоступен ученикам." : lesson.status === "draft" ? "Черновик. Материал пока виден только вам." : "Есть изменения в черновике. Здесь показана опубликованная версия."}</div> : null}
      {owner?<BankRegister lesson={lesson} topic={state.topic??""}/>:null}
      {dashboard.user.role==="student"?<LearningPlayer key={state.learning?.attempt.id??lesson.id} lessonID={lesson.id} version={lesson.version} initial={state.learning}/>:<div className="material-reader-layout"><LessonArticle lesson={lesson.version} exercises={lesson.version.exercises} practice={<LessonExercises key={lesson.version.id} lessonID={lesson.id} versionID={lesson.version.id} exercises={lesson.version.exercises} mode={owner ? "author" : "reader"} />} /><aside className="material-facts"><LessonCover track={lesson.version.track} course={state.course?.title ?? "Математика"} /><h2>Об уроке</h2><dl><dt>Предмет</dt><dd>Математика</dd><dt>Тема</dt><dd>{state.topic ?? trackLabels[lesson.version.track]}</dd><dt>Класс</dt><dd>{classLabel(lesson.version.grade_min, lesson.version.grade_max)}</dd><dt>Сложность</dt><dd>{difficultyLabels[lesson.version.difficulty]}</dd><dt>Практика</dt><dd>{countLabel(lesson.version.exercises.length)}</dd><dt>Автор</dt><dd>{lesson.author_name}</dd><dt>Статус</dt><dd>{statusLabels[lesson.status]}</dd>{lesson.version.number > 0 ? <><dt>Версия</dt><dd>{lesson.version.number}</dd></> : null}</dl></aside></div>}
    </> : null}
  </section>;
}
