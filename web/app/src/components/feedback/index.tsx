import { ConversationItem } from '@/assets/type';
import { useStore } from '@/provider';
import { Box, Stack, TextField } from '@mui/material';
import { Modal } from '@ctzhian/ui';
import { useState } from 'react';
import { useI18n } from '@/i18n/useI18n';

interface FeedbackProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (
    message_id: string,
    score: number,
    type: string,
    content?: string,
  ) => void;
  data: ConversationItem | { message_id: string } | null;
  tags?: string[];
}

const Feedback = ({
  open,
  onClose,
  onSubmit,
  data,
  tags: propsTags,
}: FeedbackProps) => {
  const { themeMode, kbDetail } = useStore();
  const { t } = useI18n();
  const [type, setType] = useState<string>('');
  const [content, setContent] = useState('');

  const tags: string[] =
    propsTags ??
    // @ts-ignore
    (kbDetail?.settings?.ai_feedback_settings?.ai_feedback_type || []);

  const handleCancel = () => {
    setContent('');
    setType('');
    onClose();
  };

  const handleSubmit = () => {
    if (!data) return;
    onSubmit(data.message_id, -1, type, content);
    handleCancel();
  };

  return (
    <Modal
      open={open}
      onCancel={handleCancel}
      title={t('feedback.title')}
      cancelText={t('common.cancel')}
      okText={t('feedback.submit')}
      onOk={handleSubmit}
      cancelButtonProps={{
        sx: {
          color: 'text.primary',
        },
      }}
    >
      <Stack
        direction='row'
        spacing={2}
        sx={{
          flexWrap: 'wrap',
          mb: 2,
        }}
      >
        {tags.map(tag => (
          <Box
            key={tag}
            sx={{
              py: 0.75,
              px: 2,
              fontSize: 12,
              borderRadius: '10px',
              border: '1px solid',
              borderColor: type === tag ? 'primary.main' : 'divider',
              cursor: 'pointer',
              color: type === tag ? 'primary.main' : 'text.primary',
              bgcolor:
                themeMode === 'dark'
                  ? 'background.paper3'
                  : 'background.default',
            }}
            onClick={() => {
              setType(tag);
            }}
          >
            {tag}
          </Box>
        ))}
      </Stack>
      <Box
        sx={{
          borderRadius: '10px',
          border: '1px solid',
          borderColor: 'divider',
          bgcolor:
            themeMode === 'dark' ? 'background.paper3' : 'background.default',
          p: 2,
        }}
      >
        <TextField
          fullWidth
          multiline
          rows={4}
          size='small'
          placeholder={t('feedback.placeholder')}
          value={content}
          sx={{
            '.MuiInputBase-root': {
              p: 0,
              overflow: 'hidden',
              transition: 'all 0.5s ease-in-out',
              bgcolor:
                themeMode === 'dark'
                  ? 'background.paper3'
                  : 'background.default',
            },
            textarea: {
              lineHeight: '26px',
              borderRadius: 0,
              transition: 'all 0.5s ease-in-out',
              '&::-webkit-scrollbar': {
                display: 'none',
              },
              '&::placeholder': {
                fontSize: 14,
              },
              scrollbarWidth: 'none',
              msOverflowStyle: 'none',
            },
            fieldset: {
              border: 'none',
            },
          }}
          onChange={e => setContent(e.target.value)}
        />
      </Box>
    </Modal>
  );
};

export default Feedback;
