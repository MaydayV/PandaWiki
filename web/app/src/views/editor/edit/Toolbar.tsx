'use client';
import { Box } from '@mui/material';
import {
  AiGenerate2Icon,
  EditorToolbar,
  UseTiptapReturn,
} from '@ctzhian/tiptap';
import { useI18n } from '@/i18n/useI18n';

interface ToolbarProps {
  editorRef: UseTiptapReturn;
  handleAiGenerate?: () => void;
}

const Toolbar = ({ editorRef, handleAiGenerate }: ToolbarProps) => {
  const { t } = useI18n();
  return (
    <Box
      sx={{
        width: 'auto',
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: '10px',
        bgcolor: 'background.default',
        px: 0.5,
        mx: 1,
      }}
    >
      {editorRef.editor && (
        <EditorToolbar
          editor={editorRef.editor}
          menuInToolbarMore={[
            {
              id: 'ai',
              label: t('editor.textPolish'),
              icon: <AiGenerate2Icon sx={{ fontSize: '1rem' }} />,
              onClick: handleAiGenerate,
            },
          ]}
        />
      )}
    </Box>
  );
};

export default Toolbar;
