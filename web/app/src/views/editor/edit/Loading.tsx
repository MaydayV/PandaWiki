'use client';
import { V1NodeDetailResp } from '@/request/types';
import { useTiptap } from '@ctzhian/tiptap';
import { useBasePath } from '@/hooks/useBasePath';
import { Box, Skeleton, Stack } from '@mui/material';
import Header from './Header';
import Toolbar from './Toolbar';
import { IconAShijian2, IconZiti } from '@panda-wiki/icons';

const LoadingEditorWrap = () => {
  const baseUrl = useBasePath();
  const editorRef = useTiptap({
    editable: false,
    content: '',
    exclude: ['invisibleCharacters', 'youtube', 'mention'],
    immediatelyRender: false,
    baseUrl: baseUrl,
  });

  return (
    <Box>
      <Box
        sx={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          zIndex: 2,
          transition: 'left 0.3s ease-in-out',
        }}
      >
        <Header detail={{} as V1NodeDetailResp} handleSave={() => {}} />
        {editorRef.editor && <Toolbar editorRef={editorRef} />}
      </Box>
      <Box>
        <Box
          sx={{
            p: '72px 72px 150px',
            mt: '102px',
            mx: 'auto',
            maxWidth: 892,
            minWidth: '386px',
          }}
        >
          <Stack direction={'row'} alignItems={'center'} gap={1} sx={{ mb: 2 }}>
            <Skeleton variant='text' width={36} height={36} />
            <Skeleton variant='text' width={300} height={36} />
          </Stack>
          <Stack direction={'row'} alignItems={'center'} gap={2} sx={{ mb: 4 }}>
            <Stack direction={'row'} alignItems={'center'} gap={0.5}>
              <IconAShijian2 sx={{ color: 'text.tertiary' }} />
              <Skeleton variant='text' width={130} height={24} />
            </Stack>
            <Stack direction={'row'} alignItems={'center'} gap={0.5}>
              <IconZiti sx={{ color: 'text.tertiary' }} />
              <Skeleton variant='text' width={80} height={24} />
            </Stack>
          </Stack>
          <Stack
            gap={1}
            sx={{
              minHeight: 'calc(100vh - 432px)',
            }}
          >
            <Skeleton variant='text' height={24} />
            <Skeleton variant='text' width={300} height={24} />
            <Skeleton variant='text' height={24} />
            <Skeleton variant='text' height={24} />
            <Skeleton variant='text' width={600} height={24} />
          </Stack>
        </Box>
      </Box>
    </Box>
  );
};

export default LoadingEditorWrap;
