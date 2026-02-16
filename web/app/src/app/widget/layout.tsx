import StoreProvider from '@/provider';
import { getShareV1AppWidgetInfo } from '@/request/ShareApp';
import { darkThemeWidget, lightThemeWidget } from '@/theme';
import { ThemeProvider } from '@ctzhian/ui';
import React from 'react';

const Layout = async ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
  let widgetDetail: any;
  try {
    widgetDetail = await getShareV1AppWidgetInfo();
  } catch {
    widgetDetail = undefined;
  }
  const themeMode = widgetDetail?.settings?.widget_bot_settings?.theme_mode;

  let selectedTheme = lightThemeWidget;

  if (themeMode === 'dark') {
    selectedTheme = darkThemeWidget;
  }

  return (
    <ThemeProvider theme={selectedTheme}>
      <StoreProvider widget={widgetDetail}>{children}</StoreProvider>
    </ThemeProvider>
  );
};

export default Layout;
