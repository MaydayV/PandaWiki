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
          乘风版基于 PandaWiki 开源项目持续演进，借助开源基础拓展了更多实用能力。
          当前版本已移除原商业授权体系相关的在线更新检测、授权激活和授权解绑流程。
          感谢原作者与开源社区的长期贡献，让我们能够站在巨人的肩膀上持续前行。
        </Box>
        <Stack direction='row' gap={1}>
          <Button
            size='small'
            onClick={() => {
              window.open('https://github.com/MaydayV/PandaWiki', '_blank');
            }}
          >
            打开乘风版仓库
          </Button>
          <Button
            size='small'
            onClick={() => {
              window.open('https://github.com/chaitin/PandaWiki', '_blank');
            }}
          >
            致谢原开源项目
          </Button>
        </Stack>
      </Stack>
    </Modal>
  );
};

export default AuthTypeModal;
