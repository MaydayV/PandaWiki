import Logo from '@/assets/images/logo.png';

import { Box, Button, Stack, Typography, useTheme } from '@mui/material';
import { ConstsUserKBPermission } from '@/request/types';
import { Modal } from '@ctzhian/ui';
import { useState, useMemo, useEffect } from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import Avatar from '../Avatar';
import Version from './Version';
import { useAppSelector } from '@/store';
import {
  IconNeirongguanli,
  IconTongjifenxi1,
  IconJushou,
  IconGongxian,
  IconPaperFull,
  IconDuihualishi1,
  IconChilun,
  IconGroup,
  IconGithub,
} from '@panda-wiki/icons';

const MENUS = [
  {
    label: '文档',
    value: '/',
    pathname: 'document',
    icon: IconNeirongguanli,
    show: true,
    perms: [
      ConstsUserKBPermission.UserKBPermissionFullControl,
      ConstsUserKBPermission.UserKBPermissionDocManage,
    ],
  },
  {
    label: '统计',
    value: '/stat',
    pathname: 'stat',
    icon: IconTongjifenxi1,
    show: true,
    perms: [
      ConstsUserKBPermission.UserKBPermissionFullControl,
      ConstsUserKBPermission.UserKBPermissionDataOperate,
    ],
  },
  {
    label: '贡献',
    value: '/contribution',
    pathname: 'contribution',
    icon: IconGongxian,
    show: true,
    perms: [ConstsUserKBPermission.UserKBPermissionFullControl],
  },
  {
    label: '问答',
    value: '/conversation',
    pathname: 'conversation',
    icon: IconDuihualishi1,
    show: true,
    perms: [
      ConstsUserKBPermission.UserKBPermissionFullControl,
      ConstsUserKBPermission.UserKBPermissionDataOperate,
    ],
  },
  {
    label: '反馈',
    value: '/feedback',
    pathname: 'feedback',
    icon: IconJushou,
    show: true,
    perms: [
      ConstsUserKBPermission.UserKBPermissionFullControl,
      ConstsUserKBPermission.UserKBPermissionDataOperate,
    ],
  },
  {
    label: '发布',
    value: '/release',
    pathname: 'release',
    icon: IconPaperFull,
    show: true,
    perms: [
      ConstsUserKBPermission.UserKBPermissionFullControl,
      ConstsUserKBPermission.UserKBPermissionDocManage,
    ],
  },
  {
    label: '设置',
    value: '/setting',
    pathname: 'application-setting',
    icon: IconChilun,
    show: true,
    perms: [ConstsUserKBPermission.UserKBPermissionFullControl],
  },
];

const Sidebar = () => {
  const { pathname } = useLocation();
  const { kbDetail } = useAppSelector(state => state.config);
  const theme = useTheme();
  const [showQrcode, setShowQrcode] = useState(false);
  const navigate = useNavigate();
  const menus = useMemo(() => {
    return MENUS.filter(it => {
      return it.perms.includes(kbDetail.perm!);
    });
  }, [kbDetail]);

  useEffect(() => {
    const menu = menus.find(it => {
      if (it.value === '/') {
        return pathname === '/';
      }
      return pathname.startsWith(it.value);
    });

    if (!menu && menus.length > 0) {
      navigate(menus[0].value);
    }
  }, [pathname, menus]);

  return (
    <Stack
      sx={{
        width: 138,
        m: 2,
        zIndex: 999,
        p: 2,
        height: 'calc(100vh - 32px)',
        bgcolor: '#FFFFFF',
        borderRadius: '10px',
        position: 'fixed',
        top: 0,
        left: 0,
        overflow: 'auto',
      }}
    >
      <Stack
        direction={'row'}
        alignItems={'center'}
        justifyContent={'center'}
        sx={{ flexShrink: 0 }}
      >
        <Avatar src={Logo} sx={{ width: 30, height: 30 }} />
      </Stack>
      <Box
        sx={{
          fontSize: '16px',
          fontWeight: 'bold',
          color: 'text.primary',
          textAlign: 'center',
          lineHeight: '36px',
          borderBottom: `1px solid ${theme.palette.divider}`,
        }}
      >
        PandaWiki
      </Box>
      <Stack sx={{ py: 2, flexGrow: 1 }} gap={1}>
        {menus.map(it => {
          let isActive = false;
          if (it.value === '/') {
            isActive = pathname === '/';
          } else {
            isActive = pathname.includes(it.value);
          }
          if (!it.show) return null;
          const IconMenu = it.icon;
          return (
            <NavLink
              key={it.pathname}
              to={it.value}
              style={{
                zIndex: isActive ? 2 : 1,
              }}
            >
              <Button
                variant={isActive ? 'contained' : 'text'}
                color='dark'
                sx={{
                  width: '100%',
                  height: 50,
                  px: 2,
                  justifyContent: 'flex-start',
                  color: isActive ? '#FFFFFF' : 'text.primary',
                  fontWeight: isActive ? '500' : '400',
                  boxShadow: isActive
                    ? '0px 10px 25px 0px rgba(33,34,45,0.2)'
                    : 'none',
                  ':hover': {
                    boxShadow: isActive
                      ? '0px 10px 25px 0px rgba(33,34,45,0.2)'
                      : 'none',
                  },
                }}
              >
                <IconMenu
                  sx={{
                    fontSize: 14,
                    mr: 1,
                    color: isActive ? '#FFFFFF' : 'text.disabled',
                  }}
                />
                {it.label}
              </Button>
            </NavLink>
          );
        })}
      </Stack>
      <Stack gap={1} sx={{ flexShrink: 0 }}>
        <Button
          variant='outlined'
          color='dark'
          sx={{
            fontSize: 14,
            flexShrink: 0,
            fontWeight: 400,
            pr: 1.5,
            pl: 1.5,
            gap: 0.5,
            justifyContent: 'flex-start',
            textTransform: 'none',
            border: `1px solid ${theme.palette.divider}`,
            '.MuiButton-startIcon': {
              mr: '3px',
            },
            '&:hover': {
              color: 'primary.main',
            },
          }}
          startIcon={<IconGithub sx={{ fontSize: '14px !important' }} />}
          onClick={() =>
            window.open('https://github.com/chaitin/PandaWiki', '_blank')
          }
        >
          原版仓库
        </Button>
        <Button
          variant='outlined'
          color='dark'
          sx={{
            fontSize: 14,
            flexShrink: 0,
            fontWeight: 400,
            pr: 1.5,
            pl: 1.5,
            gap: 0.5,
            justifyContent: 'flex-start',
            border: `1px solid ${theme.palette.divider}`,
            '.MuiButton-startIcon': {
              mr: '3px',
            },
            '&:hover': {
              color: 'primary.main',
            },
          }}
          onClick={() => setShowQrcode(true)}
          startIcon={<IconGroup sx={{ fontSize: '14px !important' }} />}
        >
          在线支持
        </Button>
        <Version />
      </Stack>
      <Modal
        open={showQrcode}
        onCancel={() => setShowQrcode(false)}
        title='在线支持'
        footer={null}
        width={640}
      >
        <Box sx={{ p: 2 }}>
          <Stack spacing={2}>
            <Box
              sx={{
                p: 2,
                borderRadius: 2,
                background: 'linear-gradient(135deg, #eff6ff 0%, #eef2ff 100%)',
                border: '1px solid #c7d2fe',
              }}
            >
              <Typography sx={{ fontSize: 15, fontWeight: 700, mb: 1 }}>
                乘风版支持说明
              </Typography>
              <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
                乘风版基于 PandaWiki 开源项目进行深度二次开发，功能边界与技术实现已和原版存在差异。
                为了保护原作者宝贵时间，也为了你能更快拿到可执行的修复，请按下方指引反馈问题。
              </Typography>
            </Box>
            <Box
              sx={{
                p: 2,
                borderRadius: 2,
                border: '1px dashed #f59e0b',
                backgroundColor: '#fffaf0',
              }}
            >
              <Typography
                component='div'
                sx={{ fontSize: 13, color: 'text.secondary', lineHeight: 1.9 }}
              >
                1. 乘风版里遇到的报错、功能差异、升级问题，请提交到乘风版仓库。
                <br />
                2. 请不要拿乘风版问题去原版仓库提 issue，别让原作者“在线背锅”。
                <br />
                3. 如果你确认是原版问题，请先在原版环境复现后，再联系原版开发者。
              </Typography>
            </Box>

            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
              <Button
                fullWidth
                variant='contained'
                onClick={() =>
                  window.open('https://github.com/MaydayV/PandaWiki', '_blank')
                }
                sx={{ textTransform: 'none' }}
              >
                反馈乘风版问题
              </Button>
              <Button
                fullWidth
                variant='outlined'
                onClick={() =>
                  window.open('https://github.com/chaitin/PandaWiki', '_blank')
                }
                sx={{ textTransform: 'none' }}
              >
                致谢并查看原版仓库
              </Button>
            </Stack>
          </Stack>
        </Box>
      </Modal>
    </Stack>
  );
};

export default Sidebar;
