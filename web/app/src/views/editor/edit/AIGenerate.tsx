import SSEClient from '@/utils/fetch';
import { useBasePath } from '@/hooks/useBasePath';
import { Editor, useTiptap, UseTiptapReturn } from '@ctzhian/tiptap';
import { Modal } from '@ctzhian/ui';
import { Box, Divider, Stack } from '@mui/material';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useI18n } from '@/i18n/useI18n';

interface AIGenerateProps {
  open: boolean;
  selectText: string;
  onClose: () => void;
  editorRef: UseTiptapReturn;
}

const AIGenerate = ({
  open,
  selectText,
  onClose,
  editorRef,
}: AIGenerateProps) => {
  const sseClientRef = useRef<SSEClient<string> | null>(null);
  const baseUrl = useBasePath();
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
  const [content, setContent] = useState('');

  const defaultEditor = useTiptap({
    editable: false,
    immediatelyRender: false,
  });

  const readEditor = useTiptap({
    editable: false,
    immediatelyRender: false,
    baseUrl: baseUrl,
  });

  const onGenerate = useCallback(() => {
    if (sseClientRef.current) {
      setLoading(true);
      sseClientRef.current.subscribe(
        JSON.stringify({
          text: selectText,
          action: 'rephrase',
          stream: true,
        }),
        data => {
          setContent(prev => {
            const newContent = prev + data;
            readEditor.editor?.commands.setContent(newContent);
            return newContent;
          });
        },
      );
    }
  }, [selectText, sseClientRef.current, readEditor.editor]);

  const onCancel = () => {
    sseClientRef.current?.unsubscribe();
    defaultEditor.editor.commands.setContent('');
    readEditor.editor.commands.setContent('');
    setContent('');
    onClose();
  };

  const onSubmit = () => {
    const { from, to } = editorRef.editor.state.selection;
    editorRef.editor.commands.insertContentAt({ from, to }, content);
    onCancel();
  };

  useEffect(() => {
    if (!open) return;
    sseClientRef.current = new SSEClient<string>({
      url: '/api/v1/creation/text',
      headers: {
        'Content-Type': 'application/json',
      },
      onComplete: () => setLoading(false),
      onError: () => setLoading(false),
    });
    if (selectText) {
      defaultEditor.editor.commands.setContent(selectText);
      setTimeout(() => {
        onGenerate();
      }, 60);
    }
  }, [selectText, open]);

  useEffect(() => {
    if (!defaultEditor.editor || !readEditor.editor) return;
    return () => {
      defaultEditor.editor.destroy();
      readEditor.editor.destroy();
      sseClientRef.current?.unsubscribe();
    };
  }, [defaultEditor, readEditor]);

  return (
    <Modal
      open={open}
      onCancel={onCancel}
      title={t('editor.textPolish')}
      okText={t('editor.replace')}
      width={1000}
      onOk={onSubmit}
      okButtonProps={{
        loading,
        disabled: content.length === 0,
      }}
    >
      <Stack
        direction={'row'}
        sx={{
          '.tiptap.ProseMirror': {
            padding: '0px',
          },
        }}
      >
        <Stack
          sx={{
            width: '50%',
            flex: 1,
          }}
        >
          <Box
            sx={{
              mb: 0.5,
              ml: 1,
              fontSize: 14,
              fontWeight: 'bold',
              color: 'text.tertiary',
            }}
          >
            {t('editor.originalText')}
          </Box>
          <Box
            sx={{
              borderRadius: '10px',
              p: 2,
              bgcolor: 'background.paper2',
              flex: 1,
            }}
          >
            <Editor editor={defaultEditor.editor} />
          </Box>
        </Stack>
        <Divider orientation='vertical' flexItem sx={{ mx: 2 }} />
        <Stack sx={{ width: '50%', flex: 1 }}>
          <Box
            sx={{
              mb: 0.5,
              ml: 1,
              fontSize: 14,
              fontWeight: 'bold',
              color: 'text.tertiary',
            }}
          >
            {t('editor.polishedText')}
          </Box>
          <Box
            sx={{
              borderRadius: '10px',
              p: 2,
              bgcolor: 'background.paper2',
              flex: 1,
            }}
          >
            <Editor editor={readEditor.editor} />
          </Box>
        </Stack>
      </Stack>
    </Modal>
  );
};

export default AIGenerate;
