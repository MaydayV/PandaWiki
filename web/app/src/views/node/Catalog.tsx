'use client';
import { NAV_BAR_HEIGHT } from '@/constant';
import { useStore } from '@/provider';
import { Box, Stack, SxProps, Tooltip } from '@mui/material';
import { IconMulu } from '@panda-wiki/icons';
import { useParams } from 'next/navigation';
import { useEffect, useRef } from 'react';
import CatalogFolder from './CatalogFolder';
import { useI18n } from '@/i18n/useI18n';

const Catalog = ({ sx }: { sx?: SxProps }) => {
  const params = useParams() || {};
  const id = params.id as string;
  const { t } = useI18n();
  const {
    kbDetail,
    mobile = false,
    catalogShow,
    setCatalogShow,
    catalogWidth,
    tree = [],
    navList = [],
  } = useStore();

  const docWidth = kbDetail?.settings?.theme_and_style?.doc_width || 'full';

  const listRef = useRef<HTMLDivElement>(null);
  const hasScrolledOnceRef = useRef(false);

  // 仅首次进入页面时自动滚动到当前文档在目录中的位置（刷新后会再次执行）
  // 切换文档时不滚动，避免打断用户浏览
  useEffect(() => {
    if (!id || !catalogShow || hasScrolledOnceRef.current) return;
    const scrollToActive = () => {
      const el = document.getElementById(`catalog-item-${id}`);
      const container = listRef.current;
      if (el && container) {
        const containerRect = container.getBoundingClientRect();
        const elementRect = el.getBoundingClientRect();
        const elementTop =
          elementRect.top - containerRect.top + container.scrollTop;
        const containerHeight = container.clientHeight;
        const elementHeight = el.offsetHeight;
        const scrollTop = elementTop - containerHeight / 2 + elementHeight / 2;
        container.scrollTo({ top: scrollTop, behavior: 'smooth' });
        hasScrolledOnceRef.current = true;
      }
    };
    const raf = requestAnimationFrame(() => {
      requestAnimationFrame(scrollToActive);
    });
    return () => cancelAnimationFrame(raf);
  }, [id, catalogShow, tree]);

  const hasNavBar = navList.length > 1;
  const stickyTop = hasNavBar ? 160 : 160 - NAV_BAR_HEIGHT;
  const stickyMaxHeight = hasNavBar
    ? `calc(100vh - 164px - ${NAV_BAR_HEIGHT}px)`
    : 'calc(100vh - 164px)';

  if (mobile) return null;

  return (
    <Stack
      flexShrink={0}
      alignItems={docWidth === 'full' ? 'flex-start' : 'flex-end'}
      sx={{
        position: 'sticky',
        top: stickyTop,
        maxHeight: stickyMaxHeight,
        zIndex: 9,
        fontSize: 14,
        width: catalogWidth,
        maxWidth: catalogWidth,
        minWidth: 24,
        overflow: 'hidden',
        transition: 'width 0.3s ease-in-out',
        ...(!catalogShow &&
          docWidth === 'full' && {
            width: 24,
          }),
        ...sx,
      }}
    >
      {!catalogShow ? (
        <Stack
          direction={'row'}
          justifyContent={'flex-end'}
          sx={{
            height: '22px',
            mb: 2,
            ...(docWidth === 'full' ? { ml: 1 } : { mr: 1 }),
          }}
        >
          <Tooltip title={catalogShow ? null : t('node.catalogExpand')} arrow>
            <IconMulu
              sx={{
                fontSize: 16,
                cursor: 'pointer',
                mr: 1,
                height: 22,
                lineHeight: '22px',
              }}
              onClick={() => setCatalogShow?.(!catalogShow)}
            />
          </Tooltip>
        </Stack>
      ) : (
        <Stack
          direction={'row'}
          alignItems={'center'}
          justifyContent={'space-between'}
          gap={1}
          sx={{
            width: '100%',
            mb: 2,
            pl: 2,
            pr: 1,
            height: '22px',
          }}
        >
          <Box
            sx={{
              fontWeight: 'bold',
              width: 'auto',
              maxWidth: 'calc(100% - 28px)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              wordBreak: 'keep-all',
            }}
          >
            {t('node.catalog')}
          </Box>
          <IconMulu
            sx={{ fontSize: 16, cursor: 'pointer' }}
            onClick={() => setCatalogShow?.(!catalogShow)}
          />
        </Stack>
      )}
      <Stack
        gap={0.5}
        sx={{
          maxHeight: 'calc(100vh - 202px)',
          overflowY: 'auto',
          overflowX: 'hidden',
          width: '100%',
          transition: 'width 0.3s ease-in-out',
          ...(!catalogShow && {
            width: 0,
          }),
        }}
        ref={listRef}
      >
        {tree.map(item => (
          <CatalogFolder key={item.id} item={item} />
        ))}
      </Stack>
    </Stack>
  );
};

export default Catalog;
