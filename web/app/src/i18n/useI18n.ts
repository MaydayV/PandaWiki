import { useContext, useMemo } from 'react';

import { StoreContext } from '@/provider';
import { MESSAGES } from './messages';
import { DEFAULT_LOCALE, Locale, resolveLanguage } from './locale';

export type MessageKey = keyof (typeof MESSAGES)['zh-CN'];

type MessageVars = Record<string, string | number>;
type TranslateFn = (key: MessageKey, vars?: MessageVars) => string;

export const LEGACY_DEFAULT_DISCLAIMER =
  '本回答由 PandaWiki 基于 AI 生成，仅供参考。';

const LEGACY_FEEDBACK_TAG_KEYS: Record<string, MessageKey> = {
  内容不准确: 'feedback.defaultInaccurate',
  没有帮助: 'feedback.defaultNotHelpful',
  其他: 'feedback.defaultOther',
};

export const localizeFeedbackTag = (
  tag: string,
  locale: Locale,
  t: TranslateFn,
): string => {
  if (locale === 'en-US' && LEGACY_FEEDBACK_TAG_KEYS[tag]) {
    return t(LEGACY_FEEDBACK_TAG_KEYS[tag]);
  }
  return tag;
};

export const localizeDisclaimer = (
  content: string | undefined,
  locale: Locale,
  t: TranslateFn,
  fallbackKey: MessageKey = 'widget.disclaimer',
): string => {
  if (!content) return '';
  if (locale === 'en-US' && content === LEGACY_DEFAULT_DISCLAIMER) {
    return t(fallbackKey);
  }
  return content;
};

export const useI18n = () => {
  const context = useContext(StoreContext);
  const configuredLanguage =
    context?.kbDetail?.settings?.language ||
    context?.widget?.settings?.language;
  const docLang =
    typeof document !== 'undefined' ? document.documentElement.lang : undefined;
  const locale: Locale =
    context?.locale ?? resolveLanguage(configuredLanguage || docLang);

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

  return {
    t,
    locale,
    setLocale: context?.setLocale,
  };
};
