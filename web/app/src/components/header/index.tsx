'use client';

import Logo from '@/assets/images/logo.png';
import { Stack, Box, IconButton, alpha, Tooltip } from '@mui/material';
import { postShareProV1AuthLogout } from '@/request/pro/ShareAuth';
import { IconDengchu } from '@panda-wiki/icons';
import { useStore } from '@/provider';
import { useMemo, useState } from 'react';
import ErrorIcon from '@mui/icons-material/Error';
import { Modal } from '@ctzhian/ui';
import {
  Header as CustomHeader,
  WelcomeHeader as WelcomeHeaderComponent,
} from '@panda-wiki/ui';
import QaModal from '../QaModal';
import ThemeSwitch from './themeSwitch';
import { getImagePath } from '@/utils/getImagePath';
import { useBasePath } from '@/hooks';
import { useI18n } from '@/i18n/useI18n';
interface HeaderProps {
  isDocPage?: boolean;
  isWelcomePage?: boolean;
}

const LogoutButton = () => {
  const [open, setOpen] = useState(false);
  const { t } = useI18n();
  const handleLogout = () => {
    return postShareProV1AuthLogout().then(() => {
      // 使用当前页面的协议（http 或 https）
      const protocol = window.location.protocol;
      const host = window.location.host;
      window.location.href = `${protocol}//${host}/auth/login`;
    });
  };
  return (
    <>
      <Modal
        title={
          <Stack direction='row' alignItems='center' gap={1}>
            <ErrorIcon sx={{ fontSize: 24, color: 'warning.main' }} />
            <Box sx={{ mt: '2px' }}>{t('common.notice')}</Box>
          </Stack>
        }
        open={open}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        onCancel={() => setOpen(false)}
        onOk={handleLogout}
        closable={false}
      >
        <Box sx={{ pl: 4 }}>{t('auth.logoutConfirm')}</Box>
      </Modal>
      <Tooltip title={t('auth.logout')} arrow>
        <IconButton size='small' onClick={() => setOpen(true)}>
          <IconDengchu
            sx={theme => ({
              cursor: 'pointer',
              color: alpha(theme.palette.text.primary, 0.65),
              fontSize: 24,
              '&:hover': { color: theme.palette.primary.main },
            })}
          />
        </IconButton>
      </Tooltip>
    </>
  );
};

const Header = ({ isDocPage = false, isWelcomePage = false }: HeaderProps) => {
  const {
    mobile = false,
    kbDetail,
    catalogWidth,
    setQaModalOpen,
    authInfo,
  } = useStore();
  const { t } = useI18n();
  const basePath = useBasePath();
  const docWidth = useMemo(() => {
    if (isWelcomePage) return 'full';
    return kbDetail?.settings?.theme_and_style?.doc_width || 'full';
  }, [kbDetail, isWelcomePage]);

  const handleSearch = (value?: string, type: 'chat' | 'search' = 'chat') => {
    if (value?.trim()) {
      if (type === 'chat') {
        sessionStorage.setItem('chat_search_query', value.trim());
        setQaModalOpen?.(true);
      } else {
        sessionStorage.setItem('chat_search_query', value.trim());
      }
    }
  };

  return (
    <CustomHeader
      isDocPage={isDocPage}
      mobile={mobile}
      docWidth={docWidth}
      catalogWidth={catalogWidth}
      logo={getImagePath(kbDetail?.settings?.icon || Logo.src, basePath)}
      title={kbDetail?.settings?.title}
      placeholder={
        kbDetail?.settings?.web_app_custom_style?.header_search_placeholder ||
        t('common.searchPlaceholder')
      }
      qaLabel={t('qa.chatTab')}
      showSearch
      homePath={basePath || '/'}
      btns={
        kbDetail?.settings?.btns?.map((item: any) => ({
          ...item,
          url: getImagePath(item.url, basePath),
          icon: getImagePath(item.icon, basePath),
        })) || []
      }
      onSearch={handleSearch}
      onQaClick={() => setQaModalOpen?.(true)}
    >
      <Stack sx={{ ml: 2 }} direction='row' alignItems='center' gap={1}>
        <ThemeSwitch />
        {!!authInfo && <LogoutButton />}
      </Stack>
      <QaModal />
    </CustomHeader>
  );
};

export const WelcomeHeader = () => {
  const basePath = useBasePath();
  const { t } = useI18n();
  const {
    mobile = false,
    kbDetail,
    catalogWidth,
    setQaModalOpen,
    authInfo,
  } = useStore();
  const handleSearch = (value?: string, type: 'chat' | 'search' = 'chat') => {
    if (value?.trim()) {
      if (type === 'chat') {
        sessionStorage.setItem('chat_search_query', value.trim());
        setQaModalOpen?.(true);
      } else {
        sessionStorage.setItem('chat_search_query', value.trim());
      }
    }
  };
  return (
    <WelcomeHeaderComponent
      isDocPage={false}
      mobile={mobile}
      docWidth='full'
      catalogWidth={catalogWidth}
      logo={getImagePath(kbDetail?.settings?.icon || Logo.src, basePath)}
      title={kbDetail?.settings?.title}
      placeholder={
        kbDetail?.settings?.web_app_custom_style?.header_search_placeholder ||
        t('common.searchPlaceholder')
      }
      qaLabel={t('qa.chatTab')}
      showSearch
      homePath={basePath || '/'}
      btns={
        kbDetail?.settings?.btns?.map((item: any) => ({
          ...item,
          url: getImagePath(item.url, basePath),
          icon: getImagePath(item.icon, basePath),
        })) || []
      }
      onSearch={handleSearch}
      onQaClick={() => setQaModalOpen?.(true)}
    >
      {!!authInfo && (
        <Box sx={{ ml: 2 }}>
          <LogoutButton />
        </Box>
      )}
      <QaModal />
    </WelcomeHeaderComponent>
  );
};

export default Header;
