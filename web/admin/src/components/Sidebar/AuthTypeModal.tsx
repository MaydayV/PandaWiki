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
    <Modal open={open} footer={null} title='关于乘风版' onCancel={onClose}>
      <Stack gap={1.5} sx={{ fontSize: 14, lineHeight: '28px' }}>
        <Stack direction='row' alignItems='center'>
          <Box sx={{ width: 120, flexShrink: 0 }}>当前版本</Box>
          <Box sx={{ fontWeight: 700 }}>{curVersion}</Box>
        </Stack>
        <Stack direction='row' alignItems='center'>
          <Box sx={{ width: 120, flexShrink: 0 }}>产品型号</Box>
          <Box>{versionInfo.label}</Box>
        </Stack>
        <Box sx={{ color: 'text.secondary', fontSize: 13, lineHeight: 1.8 }}>
          Fly Version（乘风版）取“借风而起，向远而行”之意。
          我们站在 PandaWiki 开源成果之上，把原有能力继续打磨为更适合真实业务落地的体系：
          更稳定的工程链路、更灵活的扩展边界、更可控的运维体验。
          <br />
          <br />
          乘风，不是离开巨人的肩膀，而是因为肩膀足够坚实，我们才能看得更远、飞得更高。
          感谢 PandaWiki 原作者与开源社区的长期投入与分享。
        </Box>
        <Stack direction='row' gap={1}>
          <Button
            size='small'
            onClick={() => {
              window.open('https://github.com/chaitin/PandaWiki', '_blank');
            }}
          >
            查看原版仓库
          </Button>
          <Button
            size='small'
            onClick={() => {
              window.open('https://github.com/MaydayV/PandaWiki', '_blank');
            }}
          >
            查看乘风版仓库
          </Button>
        </Stack>
      </Stack>
    </Modal>
  );
};

export default AuthTypeModal;
