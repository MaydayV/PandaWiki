export type Locale = 'zh-CN' | 'en-US';

export const DEFAULT_LOCALE: Locale = 'en-US';

const normalizeLocaleToken = (value?: string): string => {
  if (!value) return '';
  return value.trim().toLowerCase();
};

export const resolveLanguage = (value?: string): Locale => {
  const normalized = normalizeLocaleToken(value);
  const token = normalized.split(',')[0];
  if (token.startsWith('en')) {
    return 'en-US';
  }
  if (token.startsWith('zh')) {
    return 'zh-CN';
  }
  return DEFAULT_LOCALE;
};

export const toDayjsLocale = (locale: Locale): string => {
  return locale === 'en-US' ? 'en' : 'zh-cn';
};
