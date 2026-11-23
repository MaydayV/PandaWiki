export type Locale = 'zh-CN' | 'en-US';

export const DEFAULT_LOCALE: Locale = 'en-US';
export const LOCALE_COOKIE_NAME = 'pw_locale';

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

export const parseLocaleCookie = (value?: string): Locale | undefined => {
  const token = normalizeLocaleToken(value).split(';')[0].trim();
  if (!token) return undefined;
  if (token.startsWith('zh')) return 'zh-CN';
  if (token.startsWith('en')) return 'en-US';
  return undefined;
};

export const resolveEffectiveLocale = ({
  cookieLocale,
  configuredLanguage,
  acceptLanguage,
}: {
  cookieLocale?: string;
  configuredLanguage?: string;
  acceptLanguage?: string;
}): Locale => {
  const fromCookie = parseLocaleCookie(cookieLocale);
  if (fromCookie) {
    return fromCookie;
  }
  if (normalizeLocaleToken(configuredLanguage) === 'auto') {
    return resolveLanguage(acceptLanguage);
  }
  if (configuredLanguage && configuredLanguage.trim()) {
    return resolveLanguage(configuredLanguage);
  }
  if (acceptLanguage && acceptLanguage.trim()) {
    return resolveLanguage(acceptLanguage);
  }
  return DEFAULT_LOCALE;
};

export const toDayjsLocale = (locale: Locale): string => {
  return locale === 'en-US' ? 'en' : 'zh-cn';
};
