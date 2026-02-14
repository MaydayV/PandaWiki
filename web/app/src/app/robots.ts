import { getShareV1AppWebInfo } from '@/request/ShareApp';
import { getBasePath } from '@/utils';
import type { MetadataRoute } from 'next';
import { headers } from 'next/headers';

const parseAbsoluteUrl = (value?: string) => {
  if (!value) return undefined;
  try {
    return new URL(value).toString();
  } catch {
    return undefined;
  }
};

const trimTrailingSlash = (value: string) => value.replace(/\/+$/, '');

const normalizePath = (value: string) =>
  value.startsWith('/') ? value : `/${value}`;

const getRequestOrigin = (headersList: Headers) => {
  const host = headersList.get('x-forwarded-host') || headersList.get('host');
  const protocol = headersList.get('x-forwarded-proto') || 'https';
  if (!host) return '';
  return `${protocol}://${host}`;
};

const resolveSiteBase = ({
  canonicalUrl,
  kbBaseUrl,
  headersList,
}: {
  canonicalUrl?: string;
  kbBaseUrl?: string;
  headersList: Headers;
}) => {
  const absoluteCanonical = parseAbsoluteUrl(canonicalUrl?.trim());
  if (absoluteCanonical) {
    return trimTrailingSlash(absoluteCanonical);
  }

  const absoluteKbBase = parseAbsoluteUrl(kbBaseUrl?.trim());
  if (absoluteKbBase) {
    return trimTrailingSlash(absoluteKbBase);
  }

  const origin = getRequestOrigin(headersList);
  if (!origin) {
    return '';
  }

  const relativeBasePath = canonicalUrl?.trim() || getBasePath(kbBaseUrl || '');
  if (!relativeBasePath) {
    return trimTrailingSlash(origin);
  }

  return trimTrailingSlash(
    `${trimTrailingSlash(origin)}${normalizePath(relativeBasePath)}`,
  );
};

const parseRobots = (value?: string) => {
  const normalized = value?.toLowerCase() || '';
  return {
    index: !normalized.includes('noindex'),
    follow: !normalized.includes('nofollow'),
  };
};

export default async function robots(): Promise<MetadataRoute.Robots> {
  const [kbDetail, headersList] = await Promise.all([
    getShareV1AppWebInfo(),
    headers(),
  ]);

  const seoSettings = kbDetail?.settings?.seo_settings || {};
  const siteBase = resolveSiteBase({
    canonicalUrl: seoSettings?.canonical_url,
    kbBaseUrl: kbDetail?.base_url,
    headersList,
  });
  const robotsFlags = parseRobots(seoSettings?.robots);
  const allowCrawl = robotsFlags.index && robotsFlags.follow;

  return {
    rules: {
      userAgent: '*',
      allow: allowCrawl ? '/' : undefined,
      disallow: allowCrawl ? undefined : '/',
    },
    sitemap: siteBase ? `${siteBase}/sitemap.xml` : undefined,
  };
}
