import { test, expect } from '@playwright/test';

const MOCK_USER = { id: 3, name: 'Сергей Профиль', email: 'sergey@test.com', role: 'client', phone: '+79998887766' };

// Структура данных должна совпадать с тем, что ждёт компонент renderMyBookings()
const MOCK_RESERVATIONS = [
  {
    ID: 101,
    Workspace: { NameOrNumber: 'A1', Location: { Name: 'CSBS Центр', Address: 'ул. Тверская, 15' } },
    StartTime: '2026-05-06T09:00:00Z',
    EndTime: '2026-05-06T10:00:00Z',
    TariffID: 1,
    Status: 'подтверждено',
  },
];

test.describe('Страница профиля — контроль доступа', () => {
  test('неавторизованный пользователь перенаправляется на главную страницу', async ({ page }) => {
    await page.route('**/api/users/me', route =>
      route.fulfill({ status: 401, contentType: 'text/plain', body: 'Unauthorized' })
    );

    await page.goto('/profile');

    // Должен произойти редирект на главную страницу
    await expect(page).toHaveURL('/', { timeout: 10000 });
  });

  test('авторизованный пользователь видит профиль со своими данными', async ({ page }) => {
    await page.route('**/api/users/me', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );
    await page.route('**/api/reservations', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESERVATIONS) })
    );

    await page.goto('/');
    await page.evaluate((user) => {
      localStorage.setItem('isAuthenticated', 'true');
      localStorage.setItem('user', JSON.stringify(user));
    }, MOCK_USER);

    await page.goto('/profile');

    // Страница профиля должна отрендериться (не редирект)
    await expect(page).toHaveURL('/profile', { timeout: 10000 });

    // Имя пользователя отображается
    await expect(page.locator('text=Сергей Профиль')).toBeVisible({ timeout: 10000 });
  });

  test('вкладка «Мои бронирования» показывает историю броней пользователя', async ({ page }) => {
    await page.route('**/api/users/me', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );
    await page.route('**/api/reservations', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RESERVATIONS) })
    );

    await page.goto('/');
    await page.evaluate((user) => {
      localStorage.setItem('isAuthenticated', 'true');
      localStorage.setItem('user', JSON.stringify(user));
    }, MOCK_USER);

    await page.goto('/profile');
    await expect(page).toHaveURL('/profile', { timeout: 10000 });

    // Переходим на вкладку «Мои бронирования» через sidebar
    const bookingsTab = page.locator('button.profile-nav-btn', { hasText: 'Мои бронирования' });
    await expect(bookingsTab).toBeVisible({ timeout: 10000 });
    await bookingsTab.click();

    // Рабочее место из мока должно отображаться в таблице
    await expect(page.locator('text=A1')).toBeVisible({ timeout: 5000 });
  });
});
