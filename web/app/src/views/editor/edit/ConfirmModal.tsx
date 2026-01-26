'use effect';
import React, { useEffect, useState } from 'react';
import { Modal, message } from '@ctzhian/ui';
import { Box, TextField, Typography, styled, FormLabel } from '@mui/material';
import { IconCorrection } from '@panda-wiki/icons';
import { useBasePath } from '@/hooks';
import { useI18n } from '@/i18n/useI18n';

interface ConfirmModalProps {
  open: boolean;
  onCancel: () => void;
  onOk: (reason: string, token: string) => Promise<void>;
}

const StyledInfoBox = styled(Box)(({ theme }) => ({
  display: 'flex',
  alignItems: 'flex-start',
  gap: theme.spacing(2),
  padding: theme.spacing(2),
  backgroundColor: theme.palette.background.paper2,
  borderRadius: theme.spacing(1.5),
  border: `1px solid ${theme.palette.divider}`,
  marginBottom: theme.spacing(2),
}));

const StyledIconBox = styled(Box)(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 40,
  height: 40,
  borderRadius: '50%',
  backgroundColor: theme.palette.primary.main,
  color: theme.palette.primary.contrastText,
  flexShrink: 0,
}));

const StyledContentBox = styled(Box)(({ theme }) => ({
  flex: 1,
  '& .title': {
    fontSize: 16,
    fontWeight: 600,
    color: theme.palette.text.primary,
    marginBottom: theme.spacing(0.5),
  },
  '& .description': {
    fontSize: 14,
    lineHeight: 1.5,
    color: theme.palette.text.secondary,
  },
}));

export const StyledFormLabel = styled(FormLabel)(({ theme }) => ({
  display: 'block',
  color: theme.palette.text.primary,
  fontSize: 14,
  fontWeight: 400,
  marginBottom: theme.spacing(1),
  [theme.breakpoints.down('sm')]: {
    fontSize: 14,
  },
}));

const ConfirmModal = ({ open, onCancel, onOk }: ConfirmModalProps) => {
  const basePath = useBasePath();
  const { t } = useI18n();
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState(false);

  useEffect(() => {
    setReason('');
    setReasonError(false);
  }, [open]);

  const handleOk = async () => {
    if (!reason) {
      setReasonError(true);
      return;
    }
    let token = '';
    const Cap = (await import(`@cap.js/widget`)).default;
    const cap = new Cap({
      apiEndpoint: `${basePath}/share/v1/captcha/`,
    });
    try {
      const solution = await cap.solve();
      token = solution.token;
    } catch (error) {
      message.error(t('chat.validationFailed'));
      console.log(error, 'error---------');
      return;
    }
    return onOk(reason, token);
  };

  return (
    <Modal
      open={open}
      onCancel={onCancel}
      title={t('editor.submitConfirmTitle')}
      okText={t('feedback.submit')}
      onOk={handleOk}
    >
      <StyledInfoBox>
        <StyledIconBox>
          <IconCorrection sx={{ fontSize: 20 }} />
        </StyledIconBox>
        <StyledContentBox>
          <Typography className='title'>
            {t('editor.contributionTitle')}
          </Typography>
          <Typography className='description'>
            {t('editor.contributionDescription')}
          </Typography>
        </StyledContentBox>
      </StyledInfoBox>

      <StyledFormLabel required>{t('editor.updateNote')}</StyledFormLabel>
      <TextField
        fullWidth
        multiline
        rows={3}
        placeholder={t('editor.updateNotePlaceholder')}
        value={reason}
        helperText={reasonError ? t('editor.updateNoteRequired') : ''}
        error={reasonError}
        onChange={e => setReason(e.target.value)}
        sx={{
          '& .MuiOutlinedInput-root': {
            borderRadius: 2,
          },
        }}
      />
    </Modal>
  );
};

export default ConfirmModal;
