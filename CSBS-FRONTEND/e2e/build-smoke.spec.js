/**
 * Автотесты по PROD-сборке (smoke-тесты).
 * Запускаются против `vite build && vite preview` (порт 4173).
 * Проверяют, что production-артефакт рабочий: страницы грузятся,
 * навигация работает, UI отображается корректно.
 *
 * Запуск только этого набора:
 *   npx playwright test --project=build
 */

import { test, expect } from '@playwright/test';

// ─────────────────────────────────────────────────────────────────────────────
// Smoke 1: Главная страница доступна после сборки
// ─────────────────────────────────────────────────────────────────────────────
test('BUILD: главная страница возвращает 200 и содержит навбар', async ({ page }) => {
  const response = await page.goto('/');

  // HTTP-статус должен быть 200 (не 404 и не 500)
  expect(response.status()).toBe(200);

  // После сборки React-приложение монтируется в DOM
  await expect(page.locator('nav.navbar')).toBeVisible({ timeout: 10000 });

  // Логотип и бренд присутствуют
  await expect(page.locator('.brand-name')).toBeVisible();
});

// ─────────────────────────────────────────────────────────────────────────────
// Smoke 2: Страница бронирования доступна по прямой ссылке
// ─────────────────────────────────────────────────────────────────────────────
test('BUILD: страница /booking загружается и рендерит заголовок', async ({ page }) => {
  await page.goto('/booking');

  await expect(page).toHaveURL('/booking');
  // Заголовок страницы бронирования
  await expect(page.locator('h1')).toContainText('Бронирование', { timeout: 10000 });
});

// ─────────────────────────────────────────────────────────────────────────────
// Smoke 3: Страница ИИ-ассистента загружается
// ─────────────────────────────────────────────────────────────────────────────
test('BUILD: страница /ai-assistant загружается и рендерит чат-интерфейс', async ({ page }) => {
  await page.goto('/ai-assistant');

  await expect(page).toHaveURL('/ai-assistant');
  // Заголовок страницы чата — ищем внутри .chat-header
  await expect(page.locator('.chat-header h2')).toContainText('ИИ-Ассистент', { timeout: 10000 });
  // Контейнер чата должен присутствовать в DOM
  await expect(page.locator('.chat-container')).toBeVisible();
});

// ─────────────────────────────────────────────────────────────────────────────
// Smoke 4: Клиентская навигация между страницами работает без перезагрузки
// ─────────────────────────────────────────────────────────────────────────────
test('BUILD: навигация Главная → Бронирование → ИИ работает на клиенте', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('nav.navbar')).toBeVisible({ timeout: 10000 });

  // Переходим на Бронирование по ссылке в навбаре (клиентский роутинг)
  await page.click('a[href="/booking"]');
  await expect(page).toHaveURL('/booking');
  await expect(page.locator('.booking-form-card')).toBeVisible({ timeout: 10000 });

  // Переходим на ИИ-ассистент
  await page.click('a[href="/ai-assistant"]');
  await expect(page).toHaveURL('/ai-assistant');
  await expect(page.locator('.chat-container')).toBeVisible({ timeout: 10000 });
});

// ─────────────────────────────────────────────────────────────────────────────
// Smoke 5: Кнопка входа открывает модальное окно в prod-сборке
// ─────────────────────────────────────────────────────────────────────────────
test('BUILD: кнопка «Вход/Регистрация» открывает модальное окно авторизации', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('nav.navbar')).toBeVisible({ timeout: 10000 });

  // Кнопка открытия модального окна
  await page.click('button.btn-primary:has-text("Вход")');

  // Модальное окно открывается
  await expect(page.locator('.auth-modal-overlay')).toHaveClass(/open/);
  await expect(page.locator('.auth-modal')).toBeVisible();

  // Форма входа содержит поля email и пароль
  await expect(page.locator('#input_f2')).toBeVisible();
  await expect(page.locator('#input_f4')).toBeVisible();
});
