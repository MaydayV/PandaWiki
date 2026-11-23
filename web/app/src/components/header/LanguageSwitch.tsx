'use client';

import { Locale } from '@/i18n/locale';
import { useI18n } from '@/i18n/useI18n';
import LanguageIcon from '@mui/icons-material/Language';
import { Button, Menu, MenuItem, Tooltip, alpha } from '@mui/material';
import { useRouter } from 'next/navigation';
import React, { useState } from 'react';

interface LanguageSwitchProps {
  compact?: boolean;
}

const LanguageSwitch: React.FC<LanguageSwitchProps> = ({ compact = false }) => {
  const { locale, setLocale, t } = useI18n();
  const router = useRouter();
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);

  const open = Boolean(anchorEl);
  const options: { value: Locale; label: string }[] = [
    { value: 'en-US', label: t('lang.enUS') },
    { value: 'zh-CN', label: t('lang.zhCN') },
  ];

  const handleSelect = (next: Locale) => {
    setAnchorEl(null);
    if (next === locale) return;
    setLocale?.(next);
    router.refresh();
  };

  return (
    <>
      <Tooltip title={t('lang.switch')} arrow>
        <Button
          size='small'
          variant='text'
          aria-label={t('lang.switch')}
          onClick={e => setAnchorEl(e.currentTarget)}
          startIcon={<LanguageIcon sx={{ fontSize: 16 }} />}
          sx={theme => ({
            minWidth: 'auto',
            px: compact ? 0.75 : 1,
            py: 0.25,
            fontSize: 12,
            fontWeight: 500,
            textTransform: 'none',
            color: alpha(theme.palette.text.primary, 0.75),
            '&:hover': {
              color: theme.palette.primary.main,
              bgcolor: alpha(theme.palette.primary.main, 0.08),
            },
          })}
        >
          {locale === 'zh-CN' ? '中文' : 'EN'}
        </Button>
      </Tooltip>
      <Menu
        anchorEl={anchorEl}
        open={open}
        onClose={() => setAnchorEl(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        {options.map(option => (
          <MenuItem
            key={option.value}
            selected={locale === option.value}
            onClick={() => handleSelect(option.value)}
            sx={{ fontSize: 13 }}
          >
            {option.label}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};

export default LanguageSwitch;
