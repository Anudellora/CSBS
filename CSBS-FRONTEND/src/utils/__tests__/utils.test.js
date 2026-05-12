import { describe, test, expect } from 'vitest';
import { validateEmail, validatePassword } from '../validators';
import { maskEmail, maskPhone, categoryToBookingType } from '../formatters';

// ─────────────────────────────────────────────
// Unit test 1: validateEmail
// ─────────────────────────────────────────────
describe('validateEmail', () => {
  test('принимает корректный email', () => {
    expect(validateEmail('user@example.com')).toBe('');
    expect(validateEmail('ivan.petrov+tag@mail.ru')).toBe('');
  });

  test('отклоняет пустое значение и некорректный формат', () => {
    expect(validateEmail('')).toBe('Email обязателен');
    expect(validateEmail('notanemail')).toBe('Введите корректный email');
    expect(validateEmail('user@')).toBe('Введите корректный email');
    expect(validateEmail('@domain.com')).toBe('Введите корректный email');
  });
});

// ─────────────────────────────────────────────
// Unit test 2: validatePassword (режим входа)
// ─────────────────────────────────────────────
describe('validatePassword — режим входа', () => {
  test('при входе принимает любой непустой пароль без проверки сложности', () => {
    expect(validatePassword('weakpwd', false)).toBe('');
    expect(validatePassword('12345', false)).toBe('');
  });

  test('при входе отклоняет пустой пароль', () => {
    expect(validatePassword('', false)).toBe('Пароль обязателен');
  });
});

// ─────────────────────────────────────────────
// Unit test 3: validatePassword (режим регистрации)
// ─────────────────────────────────────────────
describe('validatePassword — режим регистрации', () => {
  test('отклоняет пароль короче 8 символов', () => {
    expect(validatePassword('Short1', true)).toBe('Пароль должен содержать минимум 8 символов');
  });

  test('требует хотя бы одну заглавную букву', () => {
    expect(validatePassword('password1', true)).toBe('Должна быть хотя бы одна заглавная буква');
  });

  test('требует хотя бы одну цифру', () => {
    expect(validatePassword('Password', true)).toBe('Должна быть хотя бы одна цифра');
  });

  test('принимает корректный пароль', () => {
    expect(validatePassword('Password1', true)).toBe('');
    expect(validatePassword('StrongPass99', true)).toBe('');
  });
});

// ─────────────────────────────────────────────
// Unit test 4: maskEmail / maskPhone
// ─────────────────────────────────────────────
describe('maskEmail', () => {
  test('маскирует длинный email, оставляя 3 символа и последний', () => {
    expect(maskEmail('username@example.com')).toBe('use***e@example.com');
  });

  test('не маскирует короткий local-part (≤ 4 символов)', () => {
    expect(maskEmail('ab@test.com')).toBe('ab@test.com');
  });

  test('возвращает исходное значение если нет @', () => {
    expect(maskEmail('notanemail')).toBe('notanemail');
    expect(maskEmail('')).toBe('');
  });
});

describe('maskPhone', () => {
  test('маскирует российский номер телефона', () => {
    // clean='79991234567', front='799', front[1..3]='99' → '+7 (99*) ***-**-*7'
    expect(maskPhone('+79991234567')).toBe('+7 (99*) ***-**-*7');
  });

  test('возвращает исходное значение для слишком короткого номера', () => {
    expect(maskPhone('123')).toBe('123');
  });

  test('возвращает исходное значение для пустой строки', () => {
    expect(maskPhone('')).toBe('');
    expect(maskPhone(null)).toBe(null);
  });
});

// ─────────────────────────────────────────────
// Unit test 5: categoryToBookingType
// ─────────────────────────────────────────────
describe('categoryToBookingType', () => {
  test('переговорная → room', () => {
    expect(categoryToBookingType('Переговорная')).toBe('room');
    expect(categoryToBookingType('переговорная комната')).toBe('room');
  });

  test('офис → office', () => {
    expect(categoryToBookingType('Офис')).toBe('office');
    expect(categoryToBookingType('Частный офис')).toBe('office');
  });

  test('всё остальное → desk', () => {
    expect(categoryToBookingType('Рабочее место')).toBe('desk');
    expect(categoryToBookingType('')).toBe('desk');
    expect(categoryToBookingType(null)).toBe('desk');
    expect(categoryToBookingType(undefined)).toBe('desk');
  });
});
