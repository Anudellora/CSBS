const FORBIDDEN_CHARS = /[()[\]{}|`¬¦!«£$%^&*»<>:;#~_\-+=,@]/;
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateEmail(value) {
  if (!value) return 'Email обязателен';
  if (!EMAIL_REGEX.test(value)) return 'Введите корректный email';
  return '';
}

export function validatePassword(value, isRegister = false) {
  if (!value) return 'Пароль обязателен';
  if (!isRegister) return '';
  if (value.length < 8) return 'Пароль должен содержать минимум 8 символов';
  if (FORBIDDEN_CHARS.test(value)) return 'Пароль содержит недопустимые спецсимволы';
  if (!/(?=.*[A-Z])/.test(value)) return 'Должна быть хотя бы одна заглавная буква';
  if (!/(?=.*\d)/.test(value)) return 'Должна быть хотя бы одна цифра';
  return '';
}
