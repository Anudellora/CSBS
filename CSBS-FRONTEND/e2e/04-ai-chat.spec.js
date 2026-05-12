import { test, expect } from '@playwright/test';

const MOCK_USER = { id: 2, name: 'Алекс Чат', email: 'alex@test.com', role: 'client', phone: '+79995550000' };

const MOCK_WORKSPACES = [
  { id: 5, name: 'A5', category: 'Рабочее место', location_name: 'Центр', location_id: 1, capacity: 1 },
  { id: 6, name: 'B1', category: 'Переговорная',  location_name: 'Центр', location_id: 1, capacity: 8 },
];

// Входим через localStorage, чтобы не зависеть от UI логина в каждом тесте
async function loginViaStorage(page) {
  await page.goto('/');
  await page.evaluate((user) => {
    localStorage.setItem('isAuthenticated', 'true');
    localStorage.setItem('user', JSON.stringify(user));
  }, MOCK_USER);
  await page.route('**/api/users/me', route =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
  );
}

test.describe('ИИ-Ассистент', () => {
  test('без авторизации чат отображает экран блокировки', async ({ page }) => {
    await page.goto('/ai-assistant');

    // Оверлей «Требуется авторизация» должен быть виден
    await expect(page.locator('text=Требуется авторизация')).toBeVisible({ timeout: 10000 });

    // Поле ввода должно быть заблокировано
    const chatInput = page.locator('input.chat-input');
    await expect(chatInput).toBeDisabled();
  });

  test('авторизованный пользователь видит форму ввода и приветственное сообщение', async ({ page }) => {
    await loginViaStorage(page);
    await page.goto('/ai-assistant');

    // Оверлей не отображается
    await expect(page.locator('text=Требуется авторизация')).not.toBeVisible();

    // Приветственное сообщение от ИИ есть в чате
    await expect(page.locator('.message-bubble').first()).toBeVisible({ timeout: 10000 });

    // Поле ввода активно
    const chatInput = page.locator('input.chat-input');
    await expect(chatInput).toBeEnabled();
  });

  test('отправка сообщения добавляет его в чат и получает ответ ИИ', async ({ page }) => {
    await loginViaStorage(page);

    await page.route('**/api/chat', route =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ reply: 'Вот доступные рабочие места!', action: 'list_workspaces', workspaces: MOCK_WORKSPACES }),
      })
    );

    await page.goto('/ai-assistant');

    const chatInput = page.locator('input.chat-input');
    await expect(chatInput).toBeEnabled({ timeout: 10000 });

    await chatInput.fill('Найди мне место на завтра');
    await page.click('button.btn-send');

    // Сообщение пользователя появляется в чате
    await expect(page.locator('.message-wrapper.user .message-content p')).toContainText('Найди мне место на завтра');

    // Ответ ИИ появляется в чате
    await expect(page.locator('.message-wrapper.ai .message-content p').last()).toContainText('Вот доступные рабочие места!', { timeout: 10000 });
  });

  test('карточки рабочих мест из ответа ИИ ведут на страницу бронирования', async ({ page }) => {
    await loginViaStorage(page);

    await page.route('**/api/chat', route =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ reply: 'Нашёл места:', action: 'list_workspaces', workspaces: MOCK_WORKSPACES }),
      })
    );

    await page.goto('/ai-assistant');

    const chatInput = page.locator('input.chat-input');
    await expect(chatInput).toBeEnabled({ timeout: 10000 });
    await chatInput.fill('Покажи свободные места');
    await page.click('button.btn-send');

    // Ждём появления карточки рабочего места
    const workspaceCard = page.locator('.workspace-card').first();
    await expect(workspaceCard).toBeVisible({ timeout: 10000 });

    // Клик по карточке переходит на /booking
    await workspaceCard.click();
    await expect(page).toHaveURL('/booking');
  });
});
