'use client';
import Emoji from '@/components/emoji';
import { postShareProV1FileUploadWithProgress } from '@/request/pro/otherCustomer';
import { V1NodeDetailResp } from '@/request/types';
import {
  Editor,
  EditorMarkdown,
  MarkdownEditorRef,
  TocList,
  useTiptap,
  UseTiptapReturn,
} from '@ctzhian/tiptap';
import { message } from '@ctzhian/ui';
import { Box, Stack, TextField } from '@mui/material';
import { IconAShijian2, IconZiti } from '@panda-wiki/icons';
import dayjs from 'dayjs';
import { useParams, useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useWrapContext } from '..';
import AIGenerate from './AIGenerate';
import ConfirmModal from './ConfirmModal';
import Header from './Header';
import Toc from './Toc';
import Toolbar from './Toolbar';
import IconPageview1 from '@panda-wiki/icons/IconPageview1';
import { useBasePath } from '@/hooks/useBasePath';
import { useI18n } from '@/i18n/useI18n';

interface WrapProps {
  detail: V1NodeDetailResp;
}

const Wrap = ({ detail: defaultDetail = {} }: WrapProps) => {
  const searchParams = useSearchParams();
  const contentType = searchParams.get('contentType') || 'html';
  const { nodeDetail, setNodeDetail, onSave } = useWrapContext();
  const { id } = useParams();
  const baseUrl = useBasePath();
  const { t } = useI18n();
  const markdownEditorRef = useRef<MarkdownEditorRef>(null);
  const [characterCount, setCharacterCount] = useState(0);
  const [headings, setHeadings] = useState<TocList>([]);
  const [fixedToc, setFixedToc] = useState(false);
  const [selectionText, setSelectionText] = useState('');
  const [aiGenerateOpen, setAiGenerateOpen] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);

  const isMarkdown = useMemo(() => {
    if (!id && contentType === 'md') {
      return true;
    }
    return defaultDetail.meta?.content_type === 'md';
  }, [defaultDetail.meta?.content_type, contentType]);

  const updateDetail = (value: V1NodeDetailResp) => {
    setNodeDetail({
      ...nodeDetail,
      updated_at: dayjs().format('YYYY-MM-DD HH:mm:ss'),
      status: 1,
      ...value,
    });
  };

  const handleUpload = async (
    file: File,
    onProgress?: (progress: { progress: number }) => void,
    abortSignal?: AbortSignal,
  ) => {
    const { key } = await postShareProV1FileUploadWithProgress(
      { file },
      {
        onprogress: ({ progress }) => {
          onProgress?.({ progress: progress / 100 });
        },
        abortSignal,
      },
    );
    return Promise.resolve('/static-file/' + key);
  };

  const handleTocUpdate = (toc: TocList) => {
    setHeadings(toc);
  };

  const handleError = (error: Error) => {
    if (error.message) {
      message.error(error.message);
    }
  };

  const handleUpdate = ({ editor }: { editor: UseTiptapReturn['editor'] }) => {
    setIsEditing(true);
    setCharacterCount((editor.storage as any).characterCount.characters());
  };

  const editorRef = useTiptap({
    immediatelyRender: false,
    editable: !isMarkdown,
    contentType: isMarkdown ? 'markdown' : 'html',
    baseUrl: baseUrl,
    content: defaultDetail?.content || '',
    exclude: ['invisibleCharacters', 'youtube', 'mention'],
    onCreate: ({ editor: tiptapEditor }) => {
      setCharacterCount(
        (tiptapEditor.storage as any).characterCount.characters(),
      );
    },
    onError: handleError,
    onUpload: handleUpload,
    onUpdate: handleUpdate,
    onTocUpdate: handleTocUpdate,
  });

  const handleAiGenerate = useCallback(() => {
    if (editorRef.editor) {
      const { from, to } = editorRef.editor.state.selection;
      const text = editorRef.editor.state.doc.textBetween(from, to, '\n');
      if (!text) {
        message.error(t('editor.selectTextFirst'));
        return;
      }
      setSelectionText(text);
      setAiGenerateOpen(true);
    }
  }, [editorRef.editor, t]);

  const checkRequiredFields = useCallback(
    (content?: string) => {
      if (!nodeDetail?.name?.trim()) {
        message.error(t('editor.enterDocNameFirst'));
        return false;
      }
      const contentToCheck =
        content !== undefined ? content : nodeDetail?.content;
      if (!contentToCheck?.trim()) {
        message.error(t('editor.enterDocContentFirst'));
        return false;
      }
      return true;
    },
    [nodeDetail, t],
  );

  const handleGlobalSave = useCallback(
    (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key === 's') {
        event.preventDefault();
        if (editorRef && editorRef.editor) {
          const value = editorRef.getContent();
          updateDetail({
            content: value,
          });
          if (checkRequiredFields(value)) {
            setConfirmModalOpen(true);
          }
        }
      }
    },
    [checkRequiredFields, editorRef, updateDetail],
  );
  useEffect(() => {
    document.addEventListener('keydown', handleGlobalSave);
    return () => {
      document.removeEventListener('keydown', handleGlobalSave);
    };
  }, [handleGlobalSave]);

  useEffect(() => {
    return () => {
      if (editorRef) editorRef.editor?.destroy();
    };
  }, []);

  return (
    <>
      <Box
        sx={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          zIndex: 10,
          bgcolor: 'background.default',
          transition: 'left 0.3s ease-in-out',
        }}
      >
        <Header
          detail={nodeDetail!}
          handleSave={async () => {
            let content = nodeDetail?.content || '';
            if (!isMarkdown) {
              content = editorRef.getContent();
              updateDetail({
                content: content,
              });
            }
            if (checkRequiredFields(content)) {
              setConfirmModalOpen(true);
            }
          }}
        />
        {!isMarkdown && (
          <Toolbar editorRef={editorRef} handleAiGenerate={handleAiGenerate} />
        )}
      </Box>
      <Box
        sx={{
          ...(fixedToc && {
            display: 'flex',
          }),
        }}
      >
        <Box
          sx={{
            width: `calc(100vw - 160px - ${fixedToc ? 292 : 0}px)`,
            p: isMarkdown ? '72px 80px' : '72px 80px 150px',
            mt: isMarkdown ? '56px' : '102px',
            mx: 'auto',
          }}
        >
          <Stack
            direction={'row'}
            alignItems={'center'}
            gap={1}
            sx={{ mb: 2, position: 'relative' }}
          >
            <Emoji
              type={2}
              sx={{ flexShrink: 0, width: 36, height: 36 }}
              iconSx={{ fontSize: 28 }}
              value={nodeDetail?.meta?.emoji}
              readOnly={!!id}
              onChange={value => {
                setNodeDetail({
                  ...nodeDetail,
                  meta: {
                    ...nodeDetail?.meta,
                    emoji: value,
                  },
                });
              }}
            />
            <TextField
              sx={{ flex: 1 }}
              value={nodeDetail?.name}
              placeholder={t('editor.docNamePlaceholder')}
              slotProps={{
                input: {
                  readOnly: !!id,
                  sx: {
                    fontSize: 28,
                    fontWeight: 'bold',
                    bgcolor: 'background.default',
                    '& input': {
                      p: 0,
                      lineHeight: '36px',
                      height: '36px',
                    },
                    '& fieldset': {
                      border: 'none !important',
                    },
                  },
                },
              }}
              onChange={e => {
                setNodeDetail({
                  ...nodeDetail,
                  name: e.target.value,
                });
              }}
            />
          </Stack>
          <Stack direction={'row'} alignItems={'center'} gap={2} sx={{ mb: 4 }}>
            {defaultDetail?.created_at && (
              <Stack
                direction={'row'}
                alignItems={'center'}
                gap={0.5}
                sx={{
                  fontSize: 12,
                  color: 'text.tertiary',
                  cursor: 'text',
                  ':hover': {
                    color: 'text.tertiary',
                  },
                }}
              >
                <IconAShijian2 />
                {dayjs(defaultDetail?.created_at).format(
                  'YYYY-MM-DD HH:mm:ss',
                )}{' '}
                {t('node.created')}
              </Stack>
            )}

            <Stack
              direction={'row'}
              alignItems={'center'}
              gap={0.5}
              sx={{ fontSize: 12, color: 'text.tertiary' }}
            >
              <IconZiti />
              {t('node.words', { count: characterCount })}
            </Stack>
            <Stack
              direction={'row'}
              alignItems={'center'}
              gap={0.5}
              sx={{ fontSize: 12, color: 'text.tertiary' }}
            >
              <IconPageview1 sx={{ fontSize: 12 }} />
              {t('node.views', { count: nodeDetail?.pv ?? 0 })}
            </Stack>
          </Stack>
          {editorRef.editor && (
            <Box sx={{ ...(fixedToc && { display: 'flex' }) }}>
              {isMarkdown ? (
                <EditorMarkdown
                  ref={markdownEditorRef}
                  editor={editorRef.editor}
                  value={nodeDetail?.content || defaultDetail?.content || ''}
                  placeholder={t('editor.docContentPlaceholder')}
                  onAceChange={value => {
                    updateDetail({
                      content: value,
                    });
                  }}
                  height='calc(100vh - 340px)'
                />
              ) : (
                <Box
                  sx={{
                    wordBreak: 'break-all',
                    '.tiptap.ProseMirror': {
                      overflowX: 'hidden',
                      minHeight: 'calc(100vh - 102px - 48px)',
                    },
                    '.tableWrapper': {
                      width: '100%',
                      overflowX: 'auto',
                    },
                  }}
                >
                  <Editor editor={editorRef.editor} />
                </Box>
              )}
            </Box>
          )}
        </Box>
        <Toc
          headings={headings}
          fixed={fixedToc}
          setFixed={setFixedToc}
          isMarkdown={isMarkdown}
          scrollToHeading={
            isMarkdown
              ? headingText =>
                  markdownEditorRef.current?.scrollToHeading(headingText)
              : undefined
          }
        />
      </Box>

      <AIGenerate
        open={aiGenerateOpen}
        selectText={selectionText}
        onClose={() => setAiGenerateOpen(false)}
        editorRef={editorRef}
      />

      <ConfirmModal
        open={confirmModalOpen}
        onCancel={() => setConfirmModalOpen(false)}
        onOk={async (reason: string, token: string) => {
          if (editorRef) {
            let value = nodeDetail?.content || '';
            if (!isMarkdown) {
              value = editorRef.getContent();
              updateDetail({
                content: value,
              });
            }
            await onSave(value, reason, token, isMarkdown ? 'md' : 'html');
            setConfirmModalOpen(false);
          }
        }}
      />
    </>
  );
};

export default Wrap;
