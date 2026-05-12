import { test, expect } from '@playwright/test';

const MOCK_USER = { id: 1, name: 'Иван Тестов', email: 'ivan@test.com', role: 'client', phone: '+79991234567' };

test.describe('Регистрация нового пользователя', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/users/register', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );
    await page.route('**/api/users/login', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );
    await page.route('**/api/users/me', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) })
    );
  });

  test('форма регистрации открывается и переключается в режим регистрации', async ({ page }) => {
    await page.goto('/');

    // Открываем модальное окно
    await page.click('button.btn-primary:has-text("Вход")');
    await expect(page.locator('.auth-modal-overlay')).toHaveClass(/open/);

    // Переключаемся на вкладку «Регистрация»
    await page.click('.slider-option:has-text("Регистрация")');
    await expect(page.locator('#input_f1')).toBeVisible(); // поле «Имя» только в режиме регистрации
  });

  test('успешная регистрация закрывает модальное окно и входит в систему', async ({ page }) => {
    await page.goto('/');

    await page.click('button.btn-primary:has-text("Вход")');
    await page.click('.slider-option:has-text("Регистрация")');

    await page.fill('#input_f1', 'Иван Тестов');
    await page.fill('#input_f2', 'ivan@test.com');
    await page.fill('#input_f3', '+7 (999) 123-45-67');
    await page.fill('#input_f4', 'Password1');
    await page.fill('#input_f5', 'Password1');

    await page.click('button[type="submit"]:has-text("Зарегистрироваться")');

    // Модальное окно должно закрыться
    await expect(page.locator('.auth-modal-overlay')).not.toHaveClass(/open/);

    // В навбаре должна появиться кнопка «Выйти»
    await expect(page.locator('button:has-text("Выйти")')).toBeVisible({ timeout: 5000 });
  });

  test('валидация: отображаются ошибки при отправке пустой формы', async ({ page }) => {
    await page.goto('/');

    await page.click('button.btn-primary:has-text("Вход")');
    await page.click('.slider-option:has-text("Регистрация")');

    // Отправляем пустую форму
    await page.click('button[type="submit"]:has-text("Зарегистрироваться")');

    // Ошибки валидации должны появиться
    await expect(page.locator('.error-text').first()).toBeVisible();
    // Модальное окно остаётся открытым
    await expect(page.locator('.auth-modal-overlay')).toHaveClass(/open/);
  });
});
