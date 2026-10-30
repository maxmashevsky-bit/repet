import test from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const { chromium, request, expect: baseExpect } = require(process.env.REPET_PLAYWRIGHT_MODULE || "playwright/test");
const expect = baseExpect.configure({ timeout: 15000 });
const baseURL = process.env.REPET_E2E_URL || "http://127.0.0.1:3003";
const artifacts = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../tmp/task04-visuals");

test("visual exercises: real mouse, touch and keyboard interactions with server checking", { timeout: 240000 }, async (t) => {
  await mkdir(artifacts, { recursive: true });
  const browser = await chromium.launch({ headless: true, ...(process.env.REPET_CHROME_PATH ? { executablePath: process.env.REPET_CHROME_PATH } : {}) });
  const pages = [];
  t.after(async () => { try { for (const [index, page] of pages.entries()) { t.diagnostic(page.url()); await page.screenshot({ path: path.join(artifacts, `final-${index}.png`), fullPage: true }); } } finally { await browser.close(); } });
  const teacherAPI = await request.newContext({ baseURL }); const studentAPI = await request.newContext({ baseURL });
  t.after(() => Promise.all([teacherAPI.dispose(), studentAPI.dispose()]));
  async function call(api, method, url, data, status = 200) { const res = await api.fetch(url, { method, ...(data ? { data } : {}) }); assert.equal(res.status(), status, `${method} ${url}: ${await res.text()}`); return res.json(); }
  const suffix = crypto.randomUUID();
  const register = (api, role) => call(api, "POST", "/api/v1/auth/register", { email: `${role}-${suffix}@example.test`, display_name: role === "tutor" ? "Автор визуальных задач" : "Ученик визуальных задач", password: "RepetTest123!", role, accept_terms: true }, 201);
  await register(teacherAPI, "tutor"); const { user: student } = await register(studentAPI, "student");
  const invitation = await call(teacherAPI, "POST", "/api/v1/invitations", { student_email: student.email }, 201);
  await call(studentAPI, "POST", `/api/v1/invitations/${invitation.token}/accept`, {}, 201);
  const base = "/api/v1/catalog";
  const course = await call(teacherAPI, "POST", `${base}/courses`, { title: "Математическая лаборатория", description: "", track: "school", grade_min: 3, grade_max: 6, difficulty: 2, visibility: "linked_students" }, 201);
  const topic = await call(teacherAPI, "POST", `${base}/courses/${course.id}/topics`, { title: "Модели и фигуры", description: "", parent_id: null, position: 0 }, 201);
  const lesson = await call(teacherAPI, "POST", `${base}/courses/${course.id}/lessons`, { topic_id: topic.id, title: "Исследуем математические объекты", summary: "Числа, клетки, связи и геометрия", content: "Решите задания.", track: "school", grade_min: 3, grade_max: 6, difficulty: 2 }, 201);
  const points = [{ id: "a", label: "A", x: 0, y: 0 }, { id: "b", label: "B", x: 4, y: 0 }, { id: "c", label: "C", x: 2, y: 4 }];
  const cases = [
    ["number_line", "Поставьте точку с координатой 2.", { min: -5, max: 5, step: 1 }, "2"],
    ["grid_select", "Выделите левую верхнюю и правую нижнюю клетки.", { rows: 3, columns: 3 }, '["0:0","2:2"]'],
    ["matching", "Соедините выражения с их значениями.", { left: [{ id: "a", label: "2 + 2" }, { id: "b", label: "3 + 3" }], right: [{ id: "x", label: "4" }, { id: "y", label: "6" }] }, '["a:x","b:y"]'],
    ["ordering", "Расположите числа по возрастанию.", { items: [{ id: "a", label: "3" }, { id: "b", label: "1" }, { id: "c", label: "2" }] }, '["b","c","a"]'],
    ["geometry", "Поставьте точку с координатами (2; 3).", { mode: "point", rows: 5, columns: 5 }, '["2:3"]'],
    ["geometry", "Постройте отрезки AB и BC.", { mode: "connect", rows: 5, columns: 5, points }, '["a:b","b:c"]'],
    ["geometry", "Выберите треугольник.", { mode: "select", rows: 5, columns: 5, points, objects: [{ id: "seg", label: "Отрезок AB", kind: "segment", point_ids: ["a", "b"] }, { id: "tri", label: "Треугольник", kind: "polygon", point_ids: ["a", "b", "c"] }] }, '["tri"]']
  ];
  for (const [position, [kind, prompt, config, answer]] of cases.entries()) await call(teacherAPI, "POST", `${base}/lessons/${lesson.id}/exercises`, { kind, prompt, config, options: [], difficulty: 2, position, answer_key: { answer, explanation: "VISUAL-SECRET-KEY" } }, 201);
  await call(teacherAPI, "POST", `${base}/courses/${course.id}/publish`, {}); await call(teacherAPI, "POST", `${base}/topics/${topic.id}/publish`, {}); await call(teacherAPI, "POST", `${base}/lessons/${lesson.id}/publish`, {});
  const author = await browser.newContext({ storageState: await teacherAPI.storageState(), viewport: { width: 1440, height: 1000 } });
  const reader = await browser.newContext({ storageState: await studentAPI.storageState(), viewport: { width: 1440, height: 1000 } });
  const mobileContext = await browser.newContext({ storageState: await studentAPI.storageState(), viewport: { width: 375, height: 900 }, hasTouch: true, isMobile: true });
  const page = await author.newPage(); const learner = await reader.newPage(); const mobile = await mobileContext.newPage(); pages.push(page, learner, mobile);
  const errors = []; const keys = []; let submissions = 0;
  for (const target of pages) target.on("pageerror", (error) => errors.push(error.message));
  for (const target of [learner, mobile]) target.on("request", (req) => { if (req.url().includes("answer-keys")) keys.push(req.url()); if (req.method() === "POST" && req.url().endsWith("/attempts")) submissions++; });
  await page.goto(`${baseURL}/tasks/library/${lesson.id}/edit`);
  const editor = page.locator(".material-exercise-editor").first();
  await editor.getByLabel("Настройки задания (JSON)").fill('{"min":');
  await expect(editor.locator(".engine-error")).toContainText("Проверьте настройки");
  await editor.getByLabel("Настройки задания (JSON)").fill(JSON.stringify(cases[0][2]));
  await editor.getByLabel("Координата точки").fill("2");
  await page.getByRole("button", { name: "Сохранить черновик", exact: true }).click(); await expect(page.getByText("Черновик сохранён.", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Опубликовать", exact: true }).click(); await page.getByRole("button", { name: "Подтвердить публикацию", exact: true }).click(); await page.waitForURL(new RegExp(`/tasks/library/${lesson.id}$`));
  const url = `${baseURL}/tasks/library/${lesson.id}`;
  await learner.goto(url); await expect(learner.locator(".engine-exercise")).toHaveCount(7);
  const line = learner.locator('[data-exercise-kind="number_line"]'); const svg = line.locator("svg.visual-number-svg");
  const box = await svg.boundingBox(); assert.ok(box);
  await svg.click({ position: { x: box.width * 405.6 / 600, y: box.height * 64 / 120 } });
  await expect(line.getByLabel("Координата точки")).toHaveValue("2");
  const slider = line.getByLabel("Положение точки"); await slider.focus(); await slider.press("Home"); for (let i = 0; i < 7; i++) await slider.press("ArrowRight");
  await expect(line.getByLabel("Координата точки")).toHaveValue("2"); assert.equal(submissions, 0, "interaction must not call checking API");
  const check = async (region, expected = "Верно") => { await region.getByRole("button", { name: "Проверить ответ", exact: true }).click(); await expect(region.locator(".engine-status")).toHaveText(expected); };
  await check(line);
  const grid = learner.locator('[data-exercise-kind="grid_select"]'); const firstCell = grid.getByRole("gridcell", { name: "Клетка 1, 1", exact: true });
  await firstCell.click(); await firstCell.press("ArrowDown"); await learner.keyboard.press("ArrowDown"); await learner.keyboard.press("ArrowRight"); await learner.keyboard.press("ArrowRight"); await learner.keyboard.press("Space");
  await expect(grid.getByRole("gridcell", { name: "Клетка 3, 3", exact: true })).toHaveAttribute("aria-selected", "true"); await check(grid);
  const lastCell = grid.getByRole("gridcell", { name: "Клетка 3, 3", exact: true }); await lastCell.focus(); await lastCell.press("ArrowDown"); await expect(lastCell).toBeFocused(); await lastCell.press("ArrowRight"); await expect(lastCell).toBeFocused();
  const matching = learner.locator('[data-exercise-kind="matching"]'); await matching.getByRole("button", { name: "2 + 2", exact: true }).click(); await matching.getByRole("button", { name: "4", exact: true }).click();
  const secondMatch = matching.getByLabel("Соответствие для 3 + 3"); await secondMatch.selectOption("y");
  await matching.getByRole("button", { name: "3 + 3", exact: true }).focus(); await learner.keyboard.press("Enter"); await matching.getByRole("button", { name: "6", exact: true }).focus(); await learner.keyboard.press("Space");
  await expect(secondMatch).toHaveValue("y"); await expect(matching.locator("svg line")).toHaveCount(2); await check(matching);
  const ordering = learner.locator('[data-exercise-kind="ordering"]');
  await ordering.scrollIntoViewIfNeeded();
  const start = await ordering.locator('[data-order-id="a"]').boundingBox(); const end = await ordering.locator('[data-order-id="c"]').boundingBox(); assert.ok(start && end);
  await learner.mouse.move(start.x + start.width / 2, start.y + start.height / 2); await learner.mouse.down();
  await learner.mouse.move(start.x + start.width / 2 + 10, start.y + start.height / 2 + 10, { steps: 3 });
  await learner.mouse.move(end.x + end.width / 2, end.y + end.height / 2, { steps: 12 }); await learner.mouse.move(end.x + end.width / 2 + 1, end.y + end.height / 2 + 1); await learner.mouse.up();
  await expect(ordering.locator(".visual-order-label")).toHaveText(["1", "2", "3"]);
  await ordering.locator('[data-order-id="a"]').focus(); await learner.keyboard.press("ArrowUp"); await learner.keyboard.press("ArrowDown"); await check(ordering);
  const geometry = learner.locator('[data-exercise-kind="geometry"]'); const point = geometry.nth(0);
  const target = point.getByRole("gridcell", { name: "Точка (2; 3)", exact: true }); await target.click(); await target.press("ArrowRight"); await learner.keyboard.press("Space"); await check(point, "Неверно");
  await point.getByRole("gridcell", { name: "Точка (3; 3)", exact: true }).press("ArrowLeft"); await learner.keyboard.press("Space"); await check(point);
  const connect = geometry.nth(1); await connect.getByRole("button", { name: "A (0; 0)", exact: true }).click(); await connect.getByRole("button", { name: "B (4; 0)", exact: true }).click();
  await connect.getByLabel("Начало отрезка").selectOption("b"); await connect.getByLabel("Конец отрезка").selectOption("c"); await connect.getByRole("button", { name: "Добавить отрезок", exact: true }).click();
  await connect.getByRole("button", { name: "Удалить отрезок a:b", exact: true }).click();
  await connect.getByRole("button", { name: "A (0; 0)", exact: true }).focus(); await learner.keyboard.press("Enter"); await connect.getByRole("button", { name: "B (4; 0)", exact: true }).focus(); await learner.keyboard.press("Space");
  await expect(connect.locator(".visual-connection")).toHaveCount(2); await check(connect);
  const selection = geometry.nth(2); await selection.getByRole("button", { name: "Треугольник", exact: true }).click(); await selection.getByRole("radio", { name: "Отрезок AB", exact: true }).check();
  await selection.getByRole("radio", { name: "Треугольник", exact: true }).focus(); await learner.keyboard.press("Space"); await expect(selection.locator(".visual-geometry-object.selected polygon")).toHaveCount(1); await check(selection);
  await learner.reload(); await expect(learner.locator(".engine-correct")).toHaveCount(7);
  await learner.evaluate(() => window.scrollTo(0, 0)); await learner.screenshot({ path: path.join(artifacts, "visuals-desktop.png"), fullPage: true }); await learner.screenshot({ path: path.join(artifacts, "visuals-desktop-top.png") });
  await mobile.goto(url); await expect(mobile.locator(".engine-exercise")).toHaveCount(7);
  const touchLine = mobile.locator('[data-exercise-kind="number_line"]'); const touchSvg = touchLine.locator("svg.visual-number-svg"); const touchBox = await touchSvg.boundingBox(); assert.ok(touchBox);
  await touchSvg.tap({ position: { x: touchBox.width * 194.4 / 600, y: touchBox.height * 64 / 120 } }); await expect(touchLine.getByLabel("Координата точки")).toHaveValue("-2"); await check(touchLine, "Неверно"); await touchLine.getByLabel("Координата точки").fill("2"); await check(touchLine);
  const touchGrid = mobile.locator('[data-exercise-kind="grid_select"]'); await touchGrid.getByRole("gridcell", { name: "Клетка 1, 1", exact: true }).tap(); await expect(touchGrid.getByRole("gridcell", { name: "Клетка 1, 1", exact: true })).toHaveAttribute("aria-selected", "false"); await touchGrid.getByRole("gridcell", { name: "Клетка 1, 1", exact: true }).tap(); await check(touchGrid);
  const touchMatching = mobile.locator('[data-exercise-kind="matching"]'); await touchMatching.getByRole("button", { name: "2 + 2", exact: true }).tap(); await touchMatching.getByRole("button", { name: "4", exact: true }).tap(); await check(touchMatching);
  const touchOrder = mobile.locator('[data-exercise-kind="ordering"]'); await touchOrder.getByRole("button", { name: "Переместить вверх: 3", exact: true }).tap(); await touchOrder.getByRole("button", { name: "Переместить вниз: 3", exact: true }).tap(); await check(touchOrder);
  const touchGeometry = mobile.locator('[data-exercise-kind="geometry"]'); await touchGeometry.nth(0).getByRole("gridcell", { name: "Точка (2; 3)", exact: true }).tap(); await check(touchGeometry.nth(0));
  const touchConnect = touchGeometry.nth(1); await touchConnect.getByRole("button", { name: "A (0; 0)", exact: true }).tap(); await touchConnect.getByRole("button", { name: "C (2; 4)", exact: true }).tap(); await expect(touchConnect.locator(".visual-connection")).toHaveCount(3); await touchConnect.getByRole("button", { name: "Удалить отрезок a:c", exact: true }).tap(); await check(touchConnect);
  await touchGeometry.nth(2).getByRole("button", { name: "Треугольник", exact: true }).tap(); await check(touchGeometry.nth(2));
  assert.equal(await mobile.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, "mobile overflow");
  await mobile.evaluate(() => window.scrollTo(0, 0)); await mobile.screenshot({ path: path.join(artifacts, "visuals-mobile.png"), fullPage: true }); await mobile.screenshot({ path: path.join(artifacts, "visuals-mobile-top.png") });
  const data = JSON.stringify(await call(studentAPI, "GET", `${base}/lessons/${lesson.id}`)); assert.equal(data.includes("answer_key"), false); assert.equal(data.includes("VISUAL-SECRET-KEY"), false);
  assert.deepEqual(keys, []); assert.deepEqual(errors, []);
});
