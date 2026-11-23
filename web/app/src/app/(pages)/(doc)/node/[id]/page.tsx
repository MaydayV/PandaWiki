import { LOCALE_COOKIE_NAME, resolveEffectiveLocale } from '@/i18n/locale';
import { getShareV1NodeDetail } from '@/request/ShareNode';
import { formatMeta } from '@/utils';
import Doc from '@/views/node';
import { ResolvingMetadata } from 'next';
import { cookies, headers } from 'next/headers';

export interface PageProps {
  params: Promise<{ id: string }>;
}

export async function generateMetadata(
  { params }: PageProps,
  parent: ResolvingMetadata,
) {
  const [{ id }, cookieStore, headersList] = await Promise.all([
    params,
    cookies(),
    headers(),
  ]);
  const cookieLocale = cookieStore.get(LOCALE_COOKIE_NAME)?.value;
  const locale = resolveEffectiveLocale({
    cookieLocale,
    acceptLanguage: headersList.get('accept-language') || '',
  });
  const deniedText = locale === 'zh-CN' ? '无权限访问' : 'Access denied';
  let node = {
    name: deniedText,
    meta: {
      summary: deniedText,
    },
  };
  try {
    // @ts-ignore
    node = (await getShareV1NodeDetail({
      id,
      ...(cookieLocale ? { lang: cookieLocale } : {}),
    })) as any;
  } catch (error) {
    console.log(error);
  }

  return await formatMeta(
    { title: node?.name, description: node?.meta?.summary },
    parent,
  );
}

const DocPage = async ({ params }: PageProps) => {
  const [{ id = '' }, cookieStore] = await Promise.all([params, cookies()]);
  const cookieLocale = cookieStore.get(LOCALE_COOKIE_NAME)?.value;
  let error: any = null;
  let node: any = null;
  try {
    // @ts-ignore
    node = await getShareV1NodeDetail({
      id,
      ...(cookieLocale ? { lang: cookieLocale } : {}),
    });
  } catch (err) {
    error = err;
  }
  return <Doc node={node} error={error} />;
};

export default DocPage;
