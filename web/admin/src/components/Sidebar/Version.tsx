import { Box, Stack } from '@mui/material';
import { useState } from 'react';
import packageJson from '../../../package.json';
import AuthTypeModal from './AuthTypeModal';
import { useVersionInfo } from '@/hooks';

const Version = () => {
  const versionInfo = useVersionInfo();
  const curVersion = import.meta.env.VITE_APP_VERSION || packageJson.version;
  const [typeOpen, setTypeOpen] = useState(false);

  return (
    <>
      <Stack
        justifyContent={'center'}
        gap={0.5}
        sx={{
          borderTop: '1px solid',
          borderColor: 'divider',
          pt: 2,
          mt: 1,
          cursor: 'pointer',
          color: 'text.primary',
          fontSize: 12,
        }}
        onClick={() => setTypeOpen(true)}
      >
        <Stack direction={'row'} alignItems='center' gap={0.5}>
          <Box sx={{ width: 30, color: 'text.tertiary' }}>型号</Box>
          <img src={versionInfo.image} style={{ height: 13, marginTop: -1 }} />
          {versionInfo.label}
        </Stack>
        <Stack direction={'row'} gap={0.5}>
          <Box sx={{ width: 30, color: 'text.tertiary' }}>版本</Box>
          <Box sx={{ whiteSpace: 'nowrap' }}>{curVersion}</Box>
        </Stack>
      </Stack>
      <AuthTypeModal
        open={typeOpen}
        onClose={() => setTypeOpen(false)}
        curVersion={curVersion}
      />
    </>
  );
};

export default Version;
