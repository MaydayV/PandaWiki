import { useMemo } from 'react';

import { useStore } from '@/provider';
import { MESSAGES } from './messages';
import { DEFAULT_LOCALE, resolveLanguage } from './locale';

export type MessageKey = keyof (typeof MESSAGES)['zh-CN'];

type MessageVars = Record<string, string | number>;

export const useI18n = () => {
  const { kbDetail, widget } = useStore();
  const configuredLanguage =
    kbDetail?.settings?.language || widget?.settings?.language;
  const browserLanguage =
    typeof navigator !== 'undefined' ? navigator.language : undefined;
  const locale = resolveLanguage(
    configuredLanguage === 'auto' ? browserLanguage : configuredLanguage,
  );

  const t = useMemo(() => {
    const messages = MESSAGES[locale] || MESSAGES[DEFAULT_LOCALE];
    return (key: MessageKey, vars?: MessageVars) => {
      const template = messages[key] || MESSAGES[DEFAULT_LOCALE][key] || key;
      if (!vars) return template;
      return Object.keys(vars).reduce((result, name) => {
        return result.replaceAll(`{{${name}}}`, String(vars[name]));
      }, template);
    };
  }, [locale]);

  return { t, locale };
};
