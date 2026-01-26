'use client';
import ErrorPng from '@/assets/images/500.png';
import NoPermissionImg from '@/assets/images/no-permission.png';
import NotFoundImg from '@/assets/images/404.png';
import BlockImg from '@/assets/images/block.png';
import { SxProps, Stack } from '@mui/material';
import Image from 'next/image';
import { useStore } from '@/provider';
import { useI18n } from '@/i18n/useI18n';

export default function Error({
  sx,
  error,
  reset,
}: {
  error: Partial<Error> & { digest?: string } & { code?: number | string };
  reset?: () => void;
  sx?: SxProps;
}) {
  const { mobile } = useStore();
  const { t } = useI18n();
  const CODE_MAP = {
    40003: {
      title: t('error.noPermission'),
      img: NoPermissionImg,
    },
    403: {
      title: t('error.siteClosed'),
      img: BlockImg,
    },
    40004: {
      title: t('error.notFound'),
      img: NotFoundImg,
    },
  };
  const DEFAULT_ERROR = {
    title: t('error.pageError', { code: '' }),
    img: ErrorPng,
  };
  const errorInfo =
    CODE_MAP[(error.code ?? error.message) as '40003'] || DEFAULT_ERROR;
  return (
    <Stack
      flex={1}
      sx={{
        height: '100%',
        ...(mobile && {
          width: '100%',
          marginLeft: 0,
        }),
        ...sx,
      }}
      justifyContent='center'
      alignItems='center'
    >
      <Image
        src={errorInfo.img.src}
        alt='404'
        width={380}
        height={255}
        style={{
          height: 'auto',
          ...(mobile && { width: 200 }),
        }}
      />
      <Stack
        gap={3}
        alignItems='center'
        sx={{ color: 'text.tertiary', fontSize: 14, mt: 3 }}
      >
        {errorInfo.title}
      </Stack>
    </Stack>
  );
}
