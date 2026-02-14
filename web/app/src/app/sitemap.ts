import { getShareV1AppWebInfo } from '@/request/ShareApp';
import { getShareV1NodeList } from '@/request/ShareNode';
import { DomainNodeStatus, DomainNodeType } from '@/request/types';
import { getBasePath } from '@/utils';
import type { MetadataRoute } from 'next';
import { headers } from 'next/headers';

type ShareNodeItem = {
  id?: string;
  status?: DomainNodeStatus;
  type?: DomainNodeType;
  updated_at?: string;
};

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

const toNodeUrl = (siteBase: string, nodeId: string) =>
  `${trimTrailingSlash(siteBase)}/node/${encodeURIComponent(nodeId)}`;

const toValidDate = (value?: string) => {
  if (!value) return undefined;
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? undefined : parsed;
};

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const [kbDetail, nodeListResult, headersList] = await Promise.all([
    getShareV1AppWebInfo(),
    getShareV1NodeList(),
    headers(),
  ]);

  const seoSettings = kbDetail?.settings?.seo_settings || {};
  const siteBase = resolveSiteBase({
    canonicalUrl: seoSettings?.canonical_url,
    kbBaseUrl: kbDetail?.base_url,
    headersList,
  });

  if (!siteBase) {
    return [];
  }

  const nodes: ShareNodeItem[] = Array.isArray(nodeListResult)
    ? nodeListResult
    : [];
  const publishedDocs = nodes.filter(
    node =>
      node?.id &&
      node.type === DomainNodeType.NodeTypeDocument &&
      node.status === DomainNodeStatus.NodeStatusReleased,
  );
  const uniquePublishedDocs = Array.from(
    new Map(
      publishedDocs.map(node => [node.id as string, node] as const),
    ).values(),
  );

  return [
    {
      url: `${siteBase}/`,
    },
    ...uniquePublishedDocs.map(node => ({
      url: toNodeUrl(siteBase, node.id as string),
      lastModified: toValidDate(node.updated_at),
    })),
  ];
}
