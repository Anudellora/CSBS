import { test, expect } from '@playwright/test';

const MOCK_USER = { id: 1, name: 'Мария Петрова', email: 'maria@test.com', role: 'client', phone: '+79997654321' };

test.describe('Авторизация пользователя', () => {
  test('успешный вход устанавливает сессию и показывает личный кабинет', async ({ page }) => {
    await page.route('**/api/users/login', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );
    await page.route('**/api/users/me', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );

    await page.goto('/');

    // Открываем модальное окно
    await page.click('button.btn-primary:has-text("Вход")');
    await expect(page.locator('.auth-modal-overlay')).toHaveClass(/open/);

    // Форма должна быть в режиме «Войти» по умолчанию
    await expect(page.locator('.auth-form button[type="submit"]')).toContainText('Войти');

    await page.fill('#input_f2', 'maria@test.com');
    await page.fill('#input_f4', 'Password1');

    await page.click('.auth-form button[type="submit"]');

    // Модальное окно закрывается
    await expect(page.locator('.auth-modal-overlay')).not.toHaveClass(/open/);

    // Ссылка «Личный кабинет» появляется в десктопном навбаре
    await expect(page.locator('.nav-links-desktop a[href="/profile"]')).toBeVisible({ timeout: 5000 });

    // В localStorage установлен флаг аутентификации
    const isAuth = await page.evaluate(() => localStorage.getItem('isAuthenticated'));
    expect(isAuth).toBe('true');
  });

  test('неверные учётные данные показывают сообщение об ошибке', async ({ page }) => {
    await page.route('**/api/users/login', route =>
      route.fulfill({ status: 401, contentType: 'text/plain', body: 'Неверный email или пароль' })
    );

    await page.goto('/');

    await page.click('button.btn-primary:has-text("Вход")');
    await page.fill('#input_f2', 'wrong@test.com');
    await page.fill('#input_f4', 'WrongPass1');

    await page.click('.auth-form button[type="submit"]');

    // Ошибка отображается в форме
    await expect(page.locator('.form-error-banner')).toBeVisible();
    await expect(page.locator('.form-error-banner')).toContainText('Неверный email или пароль');

    // Пользователь остаётся на форме входа
    await expect(page.locator('.auth-modal-overlay')).toHaveClass(/open/);
  });

  test('выход из системы сбрасывает сессию', async ({ page }) => {
    await page.route('**/api/users/me', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );
    await page.route('**/api/users/logout', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    );

    // Входим через localStorage напрямую (обход UI для скорости)
    await page.goto('/');
    await page.evaluate((user) => {
      localStorage.setItem('isAuthenticated', 'true');
      localStorage.setItem('user', JSON.stringify(user));
      window.dispatchEvent(new Event('authChange'));
    }, MOCK_USER);
    await page.reload();

    await expect(page.locator('button:has-text("Выйти")')).toBeVisible();

    // Кликаем «Выйти»
    await page.click('button:has-text("Выйти")');

    // Кнопка «Вход/Регистрация» снова видна
    await expect(page.locator('button.btn-primary:has-text("Вход")')).toBeVisible({ timeout: 5000 });

    // localStorage очищен
    const isAuth = await page.evaluate(() => localStorage.getItem('isAuthenticated'));
    expect(isAuth).toBeNull();
  });
});
