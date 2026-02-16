import { ConstsCopySetting } from '@/request/types';

export const DEFAULT_COPY_APPEND_CONTENT =
  '\n\n-----------------------------------------\n{{content_from}} {{url}}';

export const resolveCopyAppendSuffix = ({
  copySetting,
  copyAppendContent,
  contentFromLabel,
  currentUrl,
}: {
  copySetting?: ConstsCopySetting;
  copyAppendContent?: string;
  contentFromLabel: string;
  currentUrl: string;
}) => {
  if (copySetting !== ConstsCopySetting.CopySettingAppend) {
    return '';
  }

  const template =
    (copyAppendContent || '').trim() === ''
      ? DEFAULT_COPY_APPEND_CONTENT
      : (copyAppendContent || DEFAULT_COPY_APPEND_CONTENT);

  return template
    .replace(/\{\{\s*content_from\s*\}\}|\{content_from\}/g, contentFromLabel)
    .replace(/\{\{\s*url\s*\}\}|\{url\}/g, currentUrl);
};
