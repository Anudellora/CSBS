export function maskEmail(email) {
  if (!email || !email.includes('@')) return email;
  const [local, domain] = email.split('@');
  if (local.length <= 4) return local + '@' + domain;
  return local.substring(0, 3) + '***' + local.slice(-1) + '@' + domain;
}

export function maskPhone(phone) {
  if (!phone) return phone;
  const clean = phone.replace(/\D/g, '');
  if (clean.length < 5) return phone;
  const front = clean.substring(0, 3);
  const back = clean.slice(-1);
  return `+${front[0]} (${front.substring(1, 4)}*) ***-**-*${back}`;
}

export function categoryToBookingType(category) {
  if (!category) return 'desk';
  const c = category.toLowerCase();
  if (c.includes('перегов')) return 'room';
  if (c.includes('офис')) return 'office';
  return 'desk';
}
