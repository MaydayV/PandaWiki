'use client';
import { useStore } from '@/provider';
import { useMemo } from 'react';
import { getImagePath } from '@/utils/getImagePath';
import { useBasePath } from '@/hooks';
import { useI18n } from '@/i18n/useI18n';

import {
  Footer,
  WelcomeFooter as WelcomeFooterComponent,
} from '@panda-wiki/ui';

export const FooterProvider = ({
  showBrand = true,
  isDocPage = false,
  isWelcomePage = false,
}: {
  showBrand?: boolean;
  isDocPage?: boolean;
  isWelcomePage?: boolean;
}) => {
  const { mobile = false, catalogWidth, kbDetail } = useStore();
  const { t } = useI18n();
  const basePath = useBasePath();
  const docWidth = useMemo(() => {
    if (isWelcomePage) return 'full';
    return kbDetail?.settings?.theme_and_style?.doc_width || 'full';
  }, [kbDetail, isWelcomePage]);
  const footerSetting = kbDetail?.settings?.footer_settings;
  const customStyle = kbDetail?.settings?.web_app_custom_style;
  const brandLabel =
    kbDetail?.settings?.brand_settings?.powered_by_label ||
    t('brand.defaultCopyright');

  return (
    <Footer
      mobile={mobile}
      catalogWidth={catalogWidth}
      showBrand={showBrand}
      isDocPage={isDocPage}
      logo='https://release.baizhi.cloud/panda-wiki/icon.png'
      brandLabel={brandLabel}
      docWidth={docWidth}
      footerSetting={
        footerSetting
          ? {
              ...footerSetting,
              brand_logo: getImagePath(footerSetting?.brand_logo, basePath),
            }
          : undefined
      }
      customStyle={{
        ...customStyle,
        social_media_accounts: customStyle?.social_media_accounts?.map(
          (item: any) => ({
            ...item,
            icon: getImagePath(item.icon, basePath),
          }),
        ),
      }}
    />
  );
};

export const WelcomeFooter = ({
  showBrand = true,
}: {
  showBrand?: boolean;
}) => {
  const { mobile = false, catalogWidth, kbDetail } = useStore();
  const { t } = useI18n();
  const basePath = useBasePath();
  const footerSetting = kbDetail?.settings?.footer_settings;
  const customStyle = kbDetail?.settings?.web_app_custom_style;
  const brandLabel =
    kbDetail?.settings?.brand_settings?.powered_by_label ||
    t('brand.defaultCopyright');
  return (
    <WelcomeFooterComponent
      mobile={mobile}
      catalogWidth={catalogWidth}
      showBrand={showBrand}
      isDocPage={false}
      logo='https://release.baizhi.cloud/panda-wiki/icon.png'
      brandLabel={brandLabel}
      docWidth='full'
      footerSetting={
        footerSetting
          ? {
              ...footerSetting,
              brand_logo: getImagePath(footerSetting?.brand_logo, basePath),
            }
          : undefined
      }
      customStyle={{
        ...customStyle,
        social_media_accounts: customStyle?.social_media_accounts?.map(
          (item: any) => ({
            ...item,
            icon: getImagePath(item.icon, basePath),
          }),
        ),
      }}
    />
  );
};

export default Footer;
