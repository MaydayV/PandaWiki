'use client';
import { useStore } from '@/provider';
import { useBasePath } from '@/hooks';
import { Modal } from '@ctzhian/ui';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import MenuIcon from '@mui/icons-material/Menu';
import {
  Fab,
  FormControlLabel,
  Radio,
  RadioGroup,
  Stack,
  Tooltip,
  Zoom,
} from '@mui/material';
import { useParams, usePathname } from 'next/navigation';
import { useState } from 'react';
import { useI18n } from '@/i18n/useI18n';

const DocFab = () => {
  const pathname = usePathname();
  const { id: docId } = useParams() || {};
  const { kbDetail, mobile } = useStore();
  const [showActions, setShowActions] = useState(false);
  const [contentType, setContentType] = useState<'html' | 'md'>('html');
  const [openSelectContentTypeModal, setOpenSelectContentTypeModal] =
    useState(false);
  const basePath = useBasePath();
  const { t } = useI18n();
  if (mobile) return null;

  return (
    <>
      <Modal
        title={t('doc.createTypeTitle')}
        open={openSelectContentTypeModal}
        onCancel={() => {
          setOpenSelectContentTypeModal(false);
          setContentType('html');
        }}
        onOk={() => {
          setOpenSelectContentTypeModal(false);
          window.open(
            `${basePath}/editor?contentType=${contentType}`,
            '_blank',
          );
        }}
      >
        <RadioGroup
          value={contentType}
          onChange={e => setContentType(e.target.value as 'html' | 'md')}
        >
          <FormControlLabel
            value='html'
            control={<Radio size='small' />}
            label={t('doc.richText')}
          />
          <FormControlLabel
            value='md'
            control={<Radio size='small' />}
            label={t('doc.markdown')}
          />
        </RadioGroup>
      </Modal>
      <Stack
        gap={1}
        sx={{
          position: 'fixed',
          bottom: 70,
          right: 16,
          zIndex: 10000,
        }}
        onMouseLeave={() => setShowActions(false)}
      >
        {kbDetail?.settings.contribute_settings?.is_enable && (
          <>
            <Zoom
              in={showActions}
              style={{ transitionDelay: showActions ? '100ms' : '0ms' }}
            >
              <Tooltip title={t('doc.createDoc')} placement='left' arrow>
                <Fab
                  color='primary'
                  size='small'
                  onClick={() => {
                    setOpenSelectContentTypeModal(true);
                  }}
                >
                  <AddIcon />
                </Fab>
              </Tooltip>
            </Zoom>
            {pathname.startsWith(basePath + '/node/') && (
              <Zoom
                in={showActions}
                style={{ transitionDelay: showActions ? '40ms' : '0ms' }}
              >
                <Tooltip title={t('doc.editDoc')} placement='left' arrow>
                  <Fab
                    color='primary'
                    size='small'
                    onClick={() => {
                      window.open(`${basePath}/editor/${docId}`, '_blank');
                    }}
                  >
                    <EditIcon />
                  </Fab>
                </Tooltip>
              </Zoom>
            )}
            <Fab
              size='small'
              sx={{
                backgroundColor: 'background.paper2',
                color: 'text.secondary',
                '&:hover': { backgroundColor: 'background.paper2' },
              }}
              onMouseEnter={() => setShowActions(true)}
            >
              <MenuIcon
                sx={{
                  transition: 'transform 200ms',
                  transform: showActions ? 'rotate(90deg)' : 'rotate(0deg)',
                }}
              />
            </Fab>
          </>
        )}
      </Stack>
    </>
  );
};

export default DocFab;
