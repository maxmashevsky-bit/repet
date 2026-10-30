import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { Script } from "node:vm";
import { lessonCatalog, findLesson } from "../features/lesson-library/catalog.ts";

test("workshop catalog has unique safe routes and valid self-contained HTML", () => {
  assert.equal(lessonCatalog.length, 20);
  assert.equal(new Set(lessonCatalog.map((lesson) => lesson.slug)).size, lessonCatalog.length);
  for (const lesson of lessonCatalog) {
    assert.match(lesson.slug, /^[a-z]+(?:-[a-z]+)*$/);
    assert.match(lesson.file, /^[a-z]+(?:-[a-z]+)*\.html$/);
    assert.equal(findLesson(lesson.slug), lesson);
    const html = readFileSync(new URL(`../lesson-content/${lesson.file}`, import.meta.url), "utf8");
    assert.match(html, /<!doctype html>/i);
    assert.match(html, /<\/head>/i);
    assert.match(html, /<title>[^<]+<\/title>/i);
    for (const [, attributes, script] of html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/gi)) {
      assert.doesNotMatch(attributes, /\bsrc\s*=/i);
      if (attributes.includes("application/json")) JSON.parse(script);
      else assert.doesNotThrow(() => new Script(script, { filename: lesson.file }));
    }
  }
  assert.equal(findLesson("../secret"), undefined);
  assert.equal(findLesson("missing-lesson"), undefined);
});
