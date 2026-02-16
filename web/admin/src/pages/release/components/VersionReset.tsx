import Card from '@/components/Card';
import { postApiV1KnowledgeBaseReleaseRollback } from '@/request/KnowledgeBase';
import { DomainKBReleaseListItemResp } from '@/request/types';
import { useAppSelector } from '@/store';
import ArrowForwardIosIcon from '@mui/icons-material/ArrowForwardIos';
import ErrorIcon from '@mui/icons-material/Error';
import { Box, Stack, useTheme } from '@mui/material';
import { message, Modal } from '@ctzhian/ui';
import dayjs from 'dayjs';
import { useState } from 'react';

interface VersionResetProps {
  open: boolean;
  onClose: () => void;
  data: DomainKBReleaseListItemResp | null;
  refresh?: () => void;
}

const VersionReset = ({ open, onClose, data, refresh }: VersionResetProps) => {
  const theme = useTheme();
  const { kb_id } = useAppSelector(state => state.config);
  const [submitting, setSubmitting] = useState(false);
  if (!data) return null;

  const submit = () => {
    if (!kb_id || !data?.id || submitting) return;
    setSubmitting(true);
    postApiV1KnowledgeBaseReleaseRollback({
      kb_id,
      release_id: data.id,
    })
      .then(() => {
        message.success('版本回滚成功，文档已恢复到目标版本草稿');
        onClose();
        refresh?.();
      })
      .finally(() => {
        setSubmitting(false);
      });
  };

  return (
    <Modal
      title={
        <Stack direction='row' alignItems='center' gap={1}>
          <ErrorIcon sx={{ color: 'warning.main' }} />
          确认回滚以下版本？
        </Stack>
      }
      open={open}
      width={600}
      okText='回滚'
      okButtonProps={{ loading: submitting }}
      onCancel={onClose}
      onOk={submit}
    >
      <Card
        sx={{
          fontSize: 14,
          p: 1,
          px: 2,
          maxHeight: 'calc(100vh - 250px)',
          overflowY: 'auto',
          overflowX: 'hidden',
          bgcolor: 'background.paper3',
        }}
      >
        <Stack
          direction='row'
          alignItems={'center'}
          gap={2}
          sx={{
            borderBottom: '1px solid',
            borderColor: theme.palette.divider,
            py: 1,
          }}
        >
          <ArrowForwardIosIcon sx={{ fontSize: 12, mt: '4px' }} />
          <Box sx={{ width: '100%' }}>
            <Box sx={{ fontSize: 16, fontWeight: 500 }}>
              {data.tag || '-'}
            </Box>
            <Box sx={{ fontSize: 12, color: 'text.tertiary' }}>
              {data.message || '-'}
            </Box>
            <Box sx={{ fontSize: 12, color: 'text.tertiary', mt: 0.5 }}>
              发布时间：
              {data.created_at
                ? dayjs(data.created_at).format('YYYY-MM-DD HH:mm:ss')
                : '-'}
            </Box>
          </Box>
        </Stack>
      </Card>
    </Modal>
  );
};

export default VersionReset;
