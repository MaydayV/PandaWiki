export const getAnswerStatus = (t: (key: any) => string) => ({
  1: t('chat.searching'),
  2: t('chat.thinking'),
  3: t('chat.answering'),
  4: '',
});
