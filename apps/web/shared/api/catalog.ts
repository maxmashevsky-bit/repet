import { request } from "./client";

export type CatalogTrack = "school" | "olympiad" | "oge" | "ege";
export type CatalogStatus = "draft" | "published" | "archived";
export type CatalogPage<T> = { items: T[]; total: number; limit: number; offset: number };
export type CourseInput = { title: string; description: string; track: CatalogTrack; grade_min: number; grade_max: number; difficulty: number; visibility: "linked_students" | "organization" };
export type Course = CourseInput & { id: string; author_id: string; author_name: string; status: CatalogStatus };
export type TopicInput = { title: string; description: string; parent_id: string | null; position: number };
export type Topic = TopicInput & { id: string; course_id: string; status: CatalogStatus };
export type LearningExample = { title:string; explanation:string; min:number; max:number; steps:{text:string;value:number}[] };
export type LessonInput = { topic_id: string; title: string; summary: string; content: string; track: CatalogTrack; grade_min: number; grade_max: number; difficulty: number; example?:LearningExample|null };
export type ExerciseMedia = { asset_id: string; alt: string };
export type Exercise = { id: string; kind: "text" | "numeric" | "single_choice" | "multiple_choice" | "long_text" | "number_line" | "grid_select" | "matching" | "ordering" | "geometry"; prompt: string; options: string[]; difficulty: number; position: number; config?: Record<string, unknown>; hint_count?: number; media?: ExerciseMedia[] };
export type AnswerKey = { exercise_id: string; answer: string; explanation: string; hints?: string[] };
export type ExerciseInput = Omit<Exercise, "id" | "hint_count"> & { answer_key: { answer: string; explanation: string; hints?: string[] } };
export type LessonVersion = LessonInput & { id: string; number: number; exercises: Exercise[]; created_at: string; published_at: string | null };
export type LearningLesson = { id: string; course_id: string; author_id: string; author_name: string; status: CatalogStatus; has_unpublished_changes: boolean; version: LessonVersion };
export type CatalogQuery = { q?: string; track?: string; grade?: string; difficulty?: string; course_id?: string; topic_id?: string; author_id?: string; status?: string; limit?: number; offset?: number };

const base = "/api/v1/catalog";
function query(values: CatalogQuery = {}) {
  const params = new URLSearchParams();
  Object.entries(values).forEach(([key, value]) => { if (value !== undefined && value !== "") params.set(key, String(value)); });
  return params.size ? `?${params}` : "";
}
function write<T>(path: string, body: object, method = "POST") {
  return request<T>(base + path, { method, body: JSON.stringify(body) });
}

export const catalogApi = {
  courses: (filter: CatalogQuery = {}, signal?: AbortSignal) => request<CatalogPage<Course>>(base + "/courses" + query(filter), { signal }),
  course: (id: string, signal?: AbortSignal) => request<Course>(`${base}/courses/${encodeURIComponent(id)}`, { signal }),
  createCourse: (input: CourseInput) => write<Course>("/courses", input),
  publishCourse: (id: string) => write<Course>(`/courses/${encodeURIComponent(id)}/publish`, {}),
  topics: (id: string, filter: CatalogQuery = {}, signal?: AbortSignal) => request<CatalogPage<Topic>>(`${base}/courses/${encodeURIComponent(id)}/topics${query(filter)}`, { signal }),
  createTopic: (id: string, input: TopicInput) => write<Topic>(`/courses/${encodeURIComponent(id)}/topics`, input),
  publishTopic: (id: string) => write<Topic>(`/topics/${encodeURIComponent(id)}/publish`, {}),
  lessons: (filter: CatalogQuery = {}, signal?: AbortSignal) => request<CatalogPage<LearningLesson>>(base + "/lessons" + query(filter), { signal }),
  lesson: (id: string, signal?: AbortSignal) => request<LearningLesson>(`${base}/lessons/${encodeURIComponent(id)}`, { signal }),
  createLesson: (id: string, input: LessonInput) => write<LearningLesson>(`/courses/${encodeURIComponent(id)}/lessons`, input),
  updateLesson: (id: string, input: LessonInput) => write<LearningLesson>(`/lessons/${encodeURIComponent(id)}`, input, "PUT"),
  publishLesson: (id: string) => write<LearningLesson>(`/lessons/${encodeURIComponent(id)}/publish`, {}),
  versions: (id: string, signal?: AbortSignal) => request<CatalogPage<LessonVersion>>(`${base}/lessons/${encodeURIComponent(id)}/versions?limit=1`, { signal }),
  version: (id: string, number: number, signal?: AbortSignal) => request<LearningLesson>(`${base}/lessons/${encodeURIComponent(id)}/versions/${number}`, { signal }),
  keys: (id: string, signal?: AbortSignal) => request<{ items: AnswerKey[] }>(`${base}/lessons/${encodeURIComponent(id)}/answer-keys`, { signal }),
  saveExercise: (id: string, input: ExerciseInput, exerciseID?: string) => write<Exercise>(`/lessons/${encodeURIComponent(id)}/exercises${exerciseID ? `/${encodeURIComponent(exerciseID)}` : ""}`, input, exerciseID ? "PUT" : "POST")
};

export async function allCatalogPages<T>(load: (offset: number) => Promise<CatalogPage<T>>): Promise<T[]> {
  const result: T[] = [];
  let offset = 0;
  for (;;) {
    const page = await load(offset);
    result.push(...page.items);
    offset += page.items.length;
    if (offset >= page.total) return result;
    if (!page.items.length || offset > 100000) throw new Error("Список изменился. Обновите страницу.");
  }
}

export async function loadLibraryStructure(signal?: AbortSignal, authorID?: string) {
  const courses = await allCatalogPages((offset) => catalogApi.courses({ limit: 100, offset, author_id: authorID }, signal));
  const topics: Topic[] = [];
  for (let i = 0; i < courses.length; i += 4) {
    const batch = await Promise.all(courses.slice(i, i + 4).map((course) => allCatalogPages((offset) => catalogApi.topics(course.id, { limit: 100, offset }, signal))));
    batch.forEach((items) => topics.push(...items));
  }
  return { courses, topics };
}
