import test from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { chromium, expect: baseExpect } = require(process.env.REPET_PLAYWRIGHT_MODULE || "playwright/test");
const expect = baseExpect.configure({ timeout: 30000 });
const baseURL = process.env.REPET_E2E_URL || "http://127.0.0.1:3007";

test("registration login buttons navigate by click, touch and keyboard", { timeout: 120000 }, async () => {
  const browser = await chromium.launch({ headless: true, ...(process.env.REPET_CHROME_PATH ? { executablePath: process.env.REPET_CHROME_PATH } : {}) });
  try {
    for (const mobile of [false, true]) {
      const context = await browser.newContext({ viewport: { width: mobile ? 375 : 1440, height: 900 }, hasTouch: mobile, isMobile: mobile });
      const page = await context.newPage();
      page.setDefaultNavigationTimeout(60000);
      await page.goto(`${baseURL}/register`, { waitUntil: "domcontentloaded" });
      const top = page.locator(".auth-top-action .button");
      await expect(top).toHaveAttribute("href", "/login");
      if (mobile) await top.tap();
      else await top.locator("span").click();
      await expect(page).toHaveURL(`${baseURL}/login`);
      await page.goto(`${baseURL}/register`, { waitUntil: "domcontentloaded" });
      const bottom = page.getByRole("link", { name: "Уже есть аккаунт? Войти", exact: true }).last();
      await bottom.focus();
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.keyboard.press("Enter");
      await expect(page).toHaveURL(`${baseURL}/login`);
      await context.close();
    }
  } finally {
    await browser.close();
  }
});
