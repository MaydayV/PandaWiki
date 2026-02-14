import ErrorComponent from '@/components/error';
import StoreProvider from '@/provider';
import { ThemeStoreProvider } from '@/provider/themeStore';
import { getShareV1AppWebInfo } from '@/request/ShareApp';
import { getShareProV1AuthInfo } from '@/request/pro/ShareAuth';
import Script from 'next/script';
import { Box } from '@mui/material';
import { AppRouterCacheProvider } from '@mui/material-nextjs/v16-appRouter';
import type { Metadata, Viewport } from 'next';
import localFont from 'next/font/local';
import { headers, cookies } from 'next/headers';
import { getSelectorsByUserAgent } from 'react-device-detect';
import { getBasePath, getImagePath } from '@/utils';
import { resolveLanguage } from '@/i18n/locale';
import './globals.css';

const gilory = localFont({
  variable: '--font-gilory',
  src: [
    {
      path: '../assets/fonts/gilroy-bold-700.otf',
      weight: '700',
    },
    {
      path: '../assets/fonts/gilroy-medium-500.otf',
      weight: '400',
    },
    {
      path: '../assets/fonts/gilroy-regular-400.otf',
      weight: '300',
    },
  ],
});

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
};

const parseAbsoluteUrl = (value?: string) => {
  if (!value) return undefined;
  try {
    return new URL(value).toString();
  } catch {
    return undefined;
  }
};

const parseUrl = (value?: string) => {
  if (!value) return undefined;
  try {
    return new URL(value);
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

const resolveCanonicalUrl = (canonicalUrl: string | undefined, siteBase: string) => {
  const canonicalValue = canonicalUrl?.trim();
  if (!canonicalValue) {
    return siteBase ? `${siteBase}/` : undefined;
  }

  const absoluteCanonical = parseAbsoluteUrl(canonicalValue);
  if (absoluteCanonical) {
    return absoluteCanonical;
  }

  if (!siteBase) {
    return undefined;
  }

  return `${trimTrailingSlash(siteBase)}${normalizePath(canonicalValue)}`;
};

const parseRobots = (value?: string) => {
  const normalized = value?.toLowerCase() || '';
  return {
    index: !normalized.includes('noindex'),
    follow: !normalized.includes('nofollow'),
  };
};

const resolveTwitterCard = (
  value?: string,
  hasImage?: boolean,
): 'summary' | 'summary_large_image' | 'player' | 'app' => {
  const normalized = value?.toLowerCase();
  if (
    normalized === 'summary' ||
    normalized === 'summary_large_image' ||
    normalized === 'player' ||
    normalized === 'app'
  ) {
    return normalized;
  }
  return hasImage ? 'summary_large_image' : 'summary';
};

const parseJsonLd = (value: unknown) => {
  if (typeof value !== 'string' || !value.trim()) return null;
  try {
    const parsed = JSON.parse(value);
    if (parsed && typeof parsed === 'object') {
      return JSON.stringify(parsed);
    }
  } catch {}
  return null;
};

export async function generateMetadata(): Promise<Metadata> {
  const headersList = await headers();
  const kbDetail: any = await getShareV1AppWebInfo();
  const basePath = getBasePath(kbDetail?.base_url || '');
  const seoSettings = kbDetail?.settings?.seo_settings || {};
  const icon = getImagePath(kbDetail?.settings?.icon || '', basePath);
  const ogImage = getImagePath(seoSettings?.og_image || '', basePath);
  const shareImage = ogImage || icon;
  const shareImages = shareImage ? [shareImage] : [];
  const siteBase = resolveSiteBase({
    canonicalUrl: seoSettings?.canonical_url,
    kbBaseUrl: kbDetail?.base_url,
    headersList,
  });
  const canonical = resolveCanonicalUrl(seoSettings?.canonical_url, siteBase);
  const metadataBase =
    parseUrl(siteBase ? `${siteBase}/` : undefined) ||
    parseUrl(process.env.TARGET || '');
  const robots = parseRobots(seoSettings?.robots);
  const twitterCard = resolveTwitterCard(seoSettings?.twitter_card, !!shareImage);

  return {
    metadataBase,
    title: kbDetail?.settings?.title || 'Panda-Wiki',
    description: kbDetail?.settings?.desc || '',
    keywords: kbDetail?.settings?.keyword || '',
    alternates: canonical
      ? {
          canonical,
        }
      : undefined,
    robots: {
      index: robots.index,
      follow: robots.follow,
    },
    icons: {
      icon: icon || `${basePath}/favicon.png`,
    },
    openGraph: {
      title: kbDetail?.settings?.title || 'Panda-Wiki',
      description: kbDetail?.settings?.desc || '',
      images: shareImages,
    },
    twitter: {
      card: twitterCard,
      title: kbDetail?.settings?.title || 'Panda-Wiki',
      description: kbDetail?.settings?.desc || '',
      images: shareImages,
    },
  };
}

const Layout = async ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
  const headersList = await headers();
  const userAgent = headersList.get('user-agent');
  const cookieStore = await cookies();
  const themeMode = (cookieStore.get('theme_mode')?.value || 'light') as
    | 'light'
    | 'dark';

  let error: any = null;

  const [kbDetailResolve, authInfoResolve] = await Promise.allSettled([
    getShareV1AppWebInfo(),
    getShareProV1AuthInfo({}),
  ]);

  const authInfo: any =
    authInfoResolve.status === 'fulfilled' ? authInfoResolve.value : undefined;
  const kbDetail: any =
    kbDetailResolve.status === 'fulfilled' ? kbDetailResolve.value : undefined;

  if (
    authInfoResolve.status === 'rejected' &&
    authInfoResolve.reason.code === 403
  ) {
    error = authInfoResolve.reason;
  }

  const { isMobile } = getSelectorsByUserAgent(userAgent || '') || {
    isMobile: false,
  };

  const basePath = getBasePath(kbDetail?.base_url || '');
  const acceptLanguage = headersList.get('accept-language') || '';
  const locale = resolveLanguage(
    kbDetail?.settings?.language === 'auto'
      ? acceptLanguage
      : kbDetail?.settings?.language,
  );
  const jsonLd = parseJsonLd(kbDetail?.settings?.seo_settings?.json_ld);

  return (
    <html lang={locale}>
      <Script
        id='base-path'
        dangerouslySetInnerHTML={{
          __html: `window._BASE_PATH_ = '${basePath}';`,
        }}
      />
      {jsonLd ? (
        <Script
          id='kb-json-ld'
          type='application/ld+json'
          dangerouslySetInnerHTML={{
            __html: jsonLd,
          }}
        />
      ) : null}
      <body
        className={`${gilory.variable} ${themeMode === 'dark' ? 'dark' : 'light'}`}
      >
        <AppRouterCacheProvider>
          <ThemeStoreProvider themeMode={themeMode}>
            <StoreProvider
              kbDetail={kbDetail}
              themeMode={themeMode || 'light'}
              mobile={isMobile}
              authInfo={authInfo}
            >
              <Box
                sx={{
                  bgcolor: 'background.paper',
                  height: error ? '100vh' : 'auto',
                }}
                id='app-theme-root'
              >
                {error ? <ErrorComponent error={error} /> : children}
              </Box>
            </StoreProvider>
          </ThemeStoreProvider>
        </AppRouterCacheProvider>
      </body>
    </html>
  );
};

export default Layout;
