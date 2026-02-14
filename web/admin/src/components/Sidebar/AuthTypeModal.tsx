import { useVersionInfo } from '@/hooks';
import { Box, Stack, Button } from '@mui/material';
import { Modal } from '@ctzhian/ui';

interface AuthTypeModalProps {
  open: boolean;
  onClose: () => void;
  curVersion: string;
}

const AuthTypeModal = ({ open, onClose, curVersion }: AuthTypeModalProps) => {
  const versionInfo = useVersionInfo();

  return (
    <Modal open={open} footer={null} title='关于 PandaWiki' onCancel={onClose}>
      <Stack gap={1.5} sx={{ fontSize: 14, lineHeight: '28px' }}>
        <Stack direction='row' alignItems='center'>
          <Box sx={{ width: 120, flexShrink: 0 }}>当前版本</Box>
          <Box sx={{ fontWeight: 700 }}>{curVersion}</Box>
        </Stack>
        <Stack direction='row' alignItems='center'>
          <Box sx={{ width: 120, flexShrink: 0 }}>产品型号</Box>
          <Box>{versionInfo.label}</Box>
        </Stack>
        <Box sx={{ color: 'text.secondary', fontSize: 13 }}>
          当前系统为二开版本，已移除原版在线更新检测、授权激活和授权解绑流程。
        </Box>
        <Stack direction='row' gap={1}>
          <Button
            size='small'
            onClick={() => {
              window.open('https://github.com/MaydayV/PandaWiki', '_blank');
            }}
          >
            打开二开仓库
          </Button>
        </Stack>
      </Stack>
    </Modal>
  );
};

export default AuthTypeModal;
