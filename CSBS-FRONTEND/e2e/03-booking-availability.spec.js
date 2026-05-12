import { test, expect } from '@playwright/test';

const MOCK_USER = { id: 1, name: 'Тест Booking', email: 'booking@test.com', role: 'client', phone: '+79991111111' };

const MOCK_TARIFFS = [
  { ID: 1, LocationID: 1, DurationMinutes: 60,  Price: 500 },
  { ID: 2, LocationID: 1, DurationMinutes: 240, Price: 1500 },
  { ID: 3, LocationID: 1, DurationMinutes: 480, Price: 2500 },
];

// Все тесты бронирования требуют авторизации (страница закрыта overlay без неё)
test.beforeEach(async ({ page }) => {
  await page.route('**/api/tariffs', route =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_TARIFFS) })
  );
  await page.route('**/api/users/me', route =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
  );

  // Устанавливаем авторизацию через localStorage перед каждым тестом
  await page.goto('/');
  await page.evaluate((user) => {
    localStorage.setItem('isAuthenticated', 'true');
    localStorage.setItem('user', JSON.stringify(user));
  }, MOCK_USER);
});

test.describe('Страница бронирования', () => {
  test('страница бронирования загружается с формой и картой зала', async ({ page }) => {
    await page.route('**/api/reservations/availability**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
    );

    await page.goto('/booking');

    // Форма бронирования должна быть видна
    await expect(page.locator('.booking-form-card')).toBeVisible({ timeout: 10000 });

    // Карта зала — класс .desk-map-col
    await expect(page.locator('.desk-map-col')).toBeVisible();

    // Переключатели типа рабочего места
    await expect(page.locator('.type-selector')).toBeVisible();
  });

  test('проверка доступности отправляет запрос после заполнения даты', async ({ page }) => {
    const availabilityRequests = [];

    await page.route('**/api/reservations/availability**', route => {
      availabilityRequests.push(route.request().url());
      route.fulfill({ status: 200, contentType: 'application/json', body: '[1, 3]' });
    });

    await page.goto('/booking');
    await expect(page.locator('.booking-form-card')).toBeVisible({ timeout: 10000 });

    // Завтрашняя дата
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    const dateStr = tomorrow.toISOString().split('T')[0];

    // DatePicker — кастомный компонент: открываем попап кликом по триггеру
    await page.locator('#date-from').click();

    // В попапе кликаем на первый доступный (не заблокированный) день
    await page.locator('.dp-cell:not([disabled]):not(.dp-other-month)').first().click();

    // Ждём debounce (500 мс) + сетевой запрос
    await page.waitForTimeout(700);

    expect(availabilityRequests.length).toBeGreaterThan(0);
    expect(availabilityRequests[0]).toContain('/api/reservations/availability');
  });

  test('переключение типа рабочего места меняет активную кнопку', async ({ page }) => {
    await page.route('**/api/reservations/availability**', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
    );

    await page.goto('/booking');
    await expect(page.locator('.booking-form-card')).toBeVisible({ timeout: 10000 });

    // По умолчанию активен «Рабочее место»
    const deskBtn = page.locator('.type-btn').filter({ hasText: 'Рабочее место' });
    const roomBtn = page.locator('.type-btn').filter({ hasText: 'Переговорная' });

    await expect(deskBtn).toHaveClass(/active/);

    // Переключаемся на «Переговорная»
    await roomBtn.click();
    await expect(roomBtn).toHaveClass(/active/);
    await expect(deskBtn).not.toHaveClass(/active/);
  });
});
