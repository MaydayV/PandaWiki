import { LOCALE_COOKIE_NAME, resolveEffectiveLocale } from '@/i18n/locale';
import StoreProvider from '@/provider';
import { getShareV1AppWidgetInfo } from '@/request/ShareApp';
import { darkThemeWidget, lightThemeWidget } from '@/theme';
import { ThemeProvider } from '@ctzhian/ui';
import { cookies, headers } from 'next/headers';
import React from 'react';

const Layout = async ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
  const [headersList, cookieStore] = await Promise.all([headers(), cookies()]);
  let widgetDetail: any;
  try {
    widgetDetail = await getShareV1AppWidgetInfo();
  } catch {
    widgetDetail = undefined;
  }
  const themeMode = widgetDetail?.settings?.widget_bot_settings?.theme_mode;
  const locale = resolveEffectiveLocale({
    cookieLocale: cookieStore.get(LOCALE_COOKIE_NAME)?.value,
    configuredLanguage: widgetDetail?.settings?.language,
    acceptLanguage: headersList.get('accept-language') || '',
  });

  let selectedTheme = lightThemeWidget;

  if (themeMode === 'dark') {
    selectedTheme = darkThemeWidget;
  }

  return (
    <ThemeProvider theme={selectedTheme}>
      <StoreProvider widget={widgetDetail} locale={locale}>
        {children}
      </StoreProvider>
    </ThemeProvider>
  );
};

export default Layout;
