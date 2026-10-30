import test from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { lessonCatalog } from "../features/lesson-library/catalog.ts";

const require = createRequire(import.meta.url);
const { chromium, request, expect: baseExpect } = require(process.env.REPET_PLAYWRIGHT_MODULE || "playwright/test");
const expect = baseExpect.configure({ timeout: 30000 });
const baseURL = process.env.REPET_E2E_URL || "http://127.0.0.1:3004";
const additions = lessonCatalog.slice(10);
const artifacts = fileURLToPath(new URL("../../../tmp/workshop-additions/", import.meta.url));

test("workshop: twenty layouts, ten trainers, protected HTML and mobile navigation", { timeout: 240000 }, async (t) => {
  await mkdir(artifacts, { recursive: true });
  const browser = await chromium.launch({ headless: true, ...(process.env.REPET_CHROME_PATH ? { executablePath: process.env.REPET_CHROME_PATH } : {}) });
  const teacher = await request.newContext({ baseURL });
  const student = await request.newContext({ baseURL });
  const anonymous = await request.newContext({ baseURL });
  t.after(() => Promise.all([teacher.dispose(), student.dispose(), anonymous.dispose()]));
  const suffix = crypto.randomUUID();
  for (const [api, role] of [[teacher, "tutor"], [student, "student"]]) {
    const response = await api.post("/api/v1/auth/register", { data: { email: `${role}-workshop-${suffix}@example.test`, display_name: "Проверка мастерской", password: "WorkshopTest123!", role, accept_terms: true } });
    assert.equal(response.status(), 201, await response.text());
  }
  const desktopContext = await browser.newContext({ storageState: await teacher.storageState(), viewport: { width: 1440, height: 1000 } });
  const mobileContext = await browser.newContext({ storageState: await teacher.storageState(), viewport: { width: 375, height: 900 }, isMobile: true, hasTouch: true });
  const page = await desktopContext.newPage(); const mobile = await mobileContext.newPage();
  const errors = [];
  for (const target of [page, mobile]) target.on("pageerror", (error) => errors.push(error.message));
  t.after(async () => { try { for (const [index, target] of [page, mobile].entries()) if (!target.isClosed()) await target.screenshot({ path: `${artifacts}final-${index}.png`, fullPage: true }); } finally { await browser.close(); } });
  await page.goto(`${baseURL}/teaching/lessons`);
  await expect(page.locator(".lesson-tile")).toHaveCount(lessonCatalog.length);
  await expect(page.locator(".library-count")).toHaveText(`Тем: ${lessonCatalog.length}`);
  await page.screenshot({ path: `${artifacts}catalog-desktop.png`, fullPage: true });
  await mobile.goto(`${baseURL}/teaching/lessons`);
  await expect(mobile.locator(".lesson-tile")).toHaveCount(lessonCatalog.length);
  assert.equal(await mobile.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  await mobile.screenshot({ path: `${artifacts}catalog-mobile.png`, fullPage: true });
  for (const lesson of lessonCatalog.slice(0, 10)) {
    for (const target of [page, mobile]) {
      await target.goto(`${baseURL}/teaching/lessons/${lesson.slug}`);
      const frame = target.frameLocator("iframe.lesson-frame");
      await expect(frame.locator("#repet-unified-theme")).toHaveCount(1);
      await expect.poll(() => frame.locator("body").evaluate(() => document.readyState)).toBe("complete");
      await expect(frame.locator("body")).toHaveCSS("background-color", "rgb(245, 247, 247)");
      assert.equal(await target.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1), false, `${lesson.slug}: outer overflow`);
      assert.equal(await frame.locator("body").evaluate(() => document.documentElement.scrollWidth > innerWidth + 1), false, `${lesson.slug}: iframe overflow`);
    }
    t.diagnostic(`${lesson.slug}: desktop/mobile layout and shared theme passed`);
  }
  for (const [index, lesson] of additions.entries()) {
    const contentURL = `/lesson-content/${lesson.slug}`;
    assert.equal((await anonymous.get(contentURL)).status(), 401);
    assert.equal((await student.get(contentURL)).status(), 403);
    const content = await teacher.get(contentURL);
    assert.equal(content.status(), 200);
    assert.equal(content.headers()["cache-control"], "private, no-store");
    assert.match(content.headers()["content-security-policy"], /connect-src 'none'/);
    await page.goto(`${baseURL}/teaching/lessons/${lesson.slug}`);
    await expect(page.getByRole("heading", { name: lesson.title, exact: true })).toBeVisible();
    await expect(page.locator(".lesson-reader-bar")).toContainText(`Урок ${index + 11} из ${lessonCatalog.length}`);
    const frame = page.frameLocator("iframe.lesson-frame");
    await expect(frame.locator("#repet-unified-theme")).toHaveCount(1);
    await expect.poll(() => frame.locator("body").evaluate(() => document.readyState)).toBe("complete");
    await expect(frame.locator("body")).toHaveCSS("background-color", "rgb(245, 247, 247)");
    const before = await frame.locator("body").innerText();
    await frame.locator("button:visible,a:visible").filter({ hasText: /тренаж[её]р/i }).first().click();
    await expect.poll(() => frame.locator("body").innerText()).not.toBe(before);
    const filter = frame.locator("select:visible").first();
    await expect(filter).toBeVisible();
    const option = await filter.locator("option").nth(1).evaluate((element) => element.value);
    assert.ok(option, lesson.slug);
    await filter.selectOption(option);
    await expect(frame.locator("select:visible").first()).toHaveValue(option);
    await mobile.goto(`${baseURL}/teaching/lessons/${lesson.slug}`);
    const mobileFrame = mobile.frameLocator("iframe.lesson-frame");
    await expect(mobileFrame.locator("#repet-unified-theme")).toHaveCount(1);
    await expect.poll(() => mobileFrame.locator("body").evaluate(() => document.readyState)).toBe("complete");
    await mobileFrame.locator("button:visible,a:visible").filter({ hasText: /тренаж[её]р/i }).first().tap();
    await expect(mobileFrame.locator("select:visible").first()).toBeVisible();
    assert.equal(await mobile.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `${lesson.slug}: outer overflow`);
    assert.equal(await mobileFrame.locator("body").evaluate(() => document.documentElement.scrollWidth > innerWidth + 1), false, `${lesson.slug}: iframe overflow`);
    await mobile.screenshot({ path: `${artifacts}${lesson.slug}-mobile.png`, fullPage: true });
    t.diagnostic(`${lesson.slug}: HTML ACL, desktop trainer/filter and touch navigation passed`);
  }
  assert.equal((await teacher.get("/lesson-content/missing-lesson")).status(), 404);
  assert.deepEqual(errors, []);
});
