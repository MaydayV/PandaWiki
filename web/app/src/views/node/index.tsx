'use client';

import DocFab from '@/components/docFab';
import ErrorComponent from '@/components/error';
import { DocWidth } from '@/constant';
import { useBasePath } from '@/hooks/useBasePath';
import useCopy from '@/hooks/useCopy';
import { useStore } from '@/provider';
import { ConstsCopySetting } from '@/request/types';
import { resolveCopyAppendSuffix } from '@/utils/copyAppend';
import { TocList, useTiptap } from '@ctzhian/tiptap';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import { Box, Fab, Skeleton, Zoom } from '@mui/material';
import { usePathname } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import DocAnchor from './DocAnchor';
import DocContent from './DocContent';
import { useI18n } from '@/i18n/useI18n';

const Doc = ({
  node,
  error,
}: {
  node?: any;
  error?: Partial<Error> & { digest?: string } & { code?: number | string };
}) => {
  const { kbDetail, mobile, catalogWidth } = useStore();
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [headings, setHeadings] = useState<TocList>([]);
  const [characterCount, setCharacterCount] = useState(0);
  const pathname = usePathname();
  const isMarkdown = useMemo(() => {
    return node?.meta?.content_type === 'md';
  }, [node?.meta?.content_type]);
  const baseUrl = useBasePath();
  const editorRef = useTiptap({
    content: node?.content || '',
    editable: false,
    contentType: isMarkdown ? 'markdown' : 'html',
    immediatelyRender: false,
    baseUrl: baseUrl,
    onTocUpdate: (toc: TocList) => {
      setHeadings(toc);
    },
    onBeforeCreate: () => {
      setLoading(true);
    },
    onCreate: ({ editor }) => {
      setLoading(false);
      setCharacterCount((editor.storage as any).characterCount.characters());
    },
  });

  const docWidth = useMemo(() => {
    return kbDetail?.settings?.theme_and_style?.doc_width || 'full';
  }, [kbDetail]);

  const [showScrollTop, setShowScrollTop] = useState(false);

  useCopy({
    mode:
      kbDetail?.settings?.copy_setting !== ConstsCopySetting.CopySettingDisabled
        ? 'allow'
        : 'disable',
    blockContextMenuWhenDisabled: false,
    suffix: resolveCopyAppendSuffix({
      copySetting: kbDetail?.settings?.copy_setting,
      copyAppendContent: kbDetail?.settings?.copy_append_content,
      contentFromLabel: t('node.contentFrom'),
      currentUrl: typeof window !== 'undefined' ? window.location.href : '',
    }),
  });

  const handleScroll = () => {
    const scrollContainer = document.querySelector(
      '#scroll-container',
    ) as HTMLElement | null;
    if (!scrollContainer) return;
    setShowScrollTop(scrollContainer.scrollTop > 300);
  };

  useEffect(() => {
    const scrollContainer = document.querySelector(
      '#scroll-container',
    ) as HTMLElement | null;
    if (!scrollContainer) return;
    scrollContainer.addEventListener('scroll', handleScroll);
    return () => scrollContainer.removeEventListener('scroll', handleScroll);
  }, []);

  const scrollToTop = () => {
    const scrollContainer = document.querySelector(
      '#scroll-container',
    ) as HTMLElement | null;
    scrollContainer?.scrollTo({ top: 0, behavior: 'smooth' });
  };

  useEffect(() => {
    if (node && editorRef) {
      requestAnimationFrame(() => {
        editorRef.setContent(
          node?.content || '',
          isMarkdown ? 'markdown' : 'html',
        );
      });
    }
  }, [node]);

  useEffect(() => {
    document.querySelector('#scroll-container')?.scrollTo({ top: 0 });
  }, [pathname]);

  return (
    <>
      {error ? (
        <Box
          sx={{
            height: '100%',
            ...(docWidth === 'full' &&
              !mobile && {
                flexGrow: 1,
              }),
            ...(docWidth !== 'full' &&
              !mobile && {
                width: DocWidth[docWidth as keyof typeof DocWidth].value + 336,
                maxWidth: `calc(100% - ${catalogWidth}px - 96px)`,
              }),
            ...(mobile && {
              mx: 'auto',
              marginTop: 3,
              width: '100%',
              px: 3,
            }),
          }}
        >
          <ErrorComponent error={error} />
        </Box>
      ) : (
        <>
          {loading ? (
            <Box
              sx={{
                ...(docWidth === 'full' &&
                  !mobile && {
                    flexGrow: 1,
                  }),
                ...(docWidth !== 'full' &&
                  !mobile && {
                    width: DocWidth[docWidth as keyof typeof DocWidth].value,
                    maxWidth: `calc(100% - ${catalogWidth}px - 240px - 192px)`,
                  }),
                ...(mobile && {
                  mx: 'auto',
                  marginTop: 3,
                  width: '100%',
                  px: 3,
                }),
              }}
            >
              <Skeleton
                variant='rounded'
                width={'70%'}
                height={36}
                sx={{ mb: '10px' }}
              />
              <Skeleton
                variant='rounded'
                width={'50%'}
                height={20}
                sx={{ mb: 4 }}
              />
              {node.type === 2 && (
                <Box
                  sx={{
                    mb: 6,
                    border: '1px solid',
                    borderColor: 'divider',
                    borderRadius: '10px',
                    bgcolor: 'background.paper3',
                    p: '20px',
                    fontSize: 14,
                    lineHeight: '28px',
                    backdropFilter: 'blur(5px)',
                  }}
                >
                  <Box sx={{ fontWeight: 'bold', mb: 2, lineHeight: '22px' }}>
                    {t('node.contentSummary')}
                  </Box>
                  <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
                  <Skeleton variant='rounded' width={'30%'} height={16} />
                </Box>
              )}
              <Skeleton
                variant='rounded'
                width={'20%'}
                height={36}
                sx={{ m: '40px 0 20px' }}
              />
              <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
              <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
              <Skeleton
                variant='rounded'
                width={'70%'}
                height={16}
                sx={{ mb: 2 }}
              />
              <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
              <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
              <Skeleton
                variant='rounded'
                width={'90%'}
                height={16}
                sx={{ mb: 1 }}
              />
              <Skeleton
                variant='rounded'
                width={'35%'}
                height={36}
                sx={{ m: '40px 0 20px' }}
              />
              <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
              <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
              <Skeleton variant='rounded' height={16} sx={{ mb: 1 }} />
            </Box>
          ) : (
            <DocContent
              info={node}
              docWidth={docWidth}
              editorRef={editorRef}
              characterCount={characterCount}
            />
          )}
          {!mobile && <DocAnchor headings={headings} />}
          <DocFab />
          {!mobile && (
            <Zoom in={showScrollTop}>
              <Fab
                size='small'
                onClick={scrollToTop}
                sx={{
                  position: 'fixed',
                  bottom: 20,
                  right: 16,
                  zIndex: 10000,
                  backgroundColor: 'background.paper3',
                  color: 'text.primary',
                  '&:hover': {
                    backgroundColor: 'background.paper2',
                  },
                }}
              >
                <KeyboardArrowUpIcon sx={{ fontSize: 24 }} />
              </Fab>
            </Zoom>
          )}
        </>
      )}
    </>
  );
};

export default Doc;
