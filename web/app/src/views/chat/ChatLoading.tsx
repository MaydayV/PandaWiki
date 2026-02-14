import LoadingIcon from '@/assets/images/loading.png';
import { alpha, Box, Stack } from '@mui/material';
import Image from 'next/image';
import { getAnswerStatus } from './constant';
import { useI18n } from '@/i18n/useI18n';

interface ChatLoadingProps {
  thinking: keyof ReturnType<typeof getAnswerStatus>;
  statusText?: ReturnType<typeof getAnswerStatus>;
  onClick?: () => void;
}

const ChatLoading = ({ thinking, statusText, onClick }: ChatLoadingProps) => {
  const { t } = useI18n();
  const answerStatus = statusText || getAnswerStatus(t);
  return (
    <Stack
      direction={onClick ? 'row-reverse' : 'row'}
      alignItems={'center'}
      gap={1}
      sx={{
        color: 'text.tertiary',
        fontSize: 12,
      }}
      onClick={() => onClick?.()}
    >
      <Stack
        direction={onClick ? 'row-reverse' : 'row'}
        alignItems={'center'}
        sx={{ position: 'relative' }}
      >
        <Image
          src={LoadingIcon.src}
          alt='loading'
          width={20}
          height={20}
          style={{ animation: 'loadingRotate 1s linear infinite' }}
        />
        <Box
          sx={{
            width: 6,
            height: 6,
            bgcolor: 'primary.main',
            borderRadius: '1px',
            position: 'absolute',
            top: 7,
            left: 7,
          }}
        ></Box>
      </Stack>
      <Box
        sx={theme => ({
          lineHeight: 1,
          color: alpha(theme.palette.text.primary, 0.5),
        })}
      >
        {answerStatus[thinking]}
      </Box>
    </Stack>
  );
};

export default ChatLoading;
