'use client';

import {
  postShareProV1AuthCas,
  postShareProV1AuthDingtalk,
  postShareProV1AuthFeishu,
  postShareProV1AuthGithub,
  postShareProV1AuthLdap,
  postShareProV1AuthOauth,
  postShareProV1AuthWecom,
} from '@/request/pro/ShareAuth';
import { postShareV1AuthGithub } from '@/request/ShareAuth';
import {
  getShareV1AuthGet,
  postShareV1AuthLoginSimple,
} from '@/request/ShareAuth';
import { getShareV1NodeList } from '@/request/ShareNode';
import { clearCookie } from '@/utils/cookie';

import {
  IconKoulingrenzheng,
  IconLDAP,
  IconMima,
  IconZhanghao,
  IconFeishu,
} from '@panda-wiki/icons';

import Logo from '@/assets/images/logo.png';
import { FooterProvider } from '@/components/footer';
import { IconCAS, IconDingDing, IconQiyeweixin } from '@/components/icons';
import { IconGitHub1 } from '@panda-wiki/icons';
import { useStore } from '@/provider';
import {
  ConstsSourceType,
  ConstsAuthType,
  ConstsLicenseEdition,
} from '@/request/types';
import {
  Box,
  Button,
  IconButton,
  InputAdornment,
  Stack,
  TextField,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import Image from 'next/image';
import { useSearchParams } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useBasePath } from '@/hooks';
import { getImagePath } from '@/utils/getImagePath';
import { useI18n } from '@/i18n/useI18n';
function isWeComByUA() {
  if (typeof navigator === 'undefined') {
    return false;
  }
  const ua = navigator.userAgent.toLowerCase();
  // 1. 必须包含 MicroMessenger (表示微信/企业微信内核)
  // 2. 必须包含 wxwork 或 wecom (表示企业微信)
  return (
    ua.includes('micromessenger') &&
    (ua.includes('wxwork') || ua.includes('wecom'))
  );
}

export default function Login() {
  const searchParams = useSearchParams();
  const [password, setPassword] = useState('');
  const [username, setUsername] = useState('');
  const [loading, setLoading] = useState(false);
  const [authType, setAuthType] = useState<ConstsAuthType>();
  const [licenseEdition, setLicenseEdition] = useState<ConstsLicenseEdition>();
  const [sourceType, setSourceType] = useState<ConstsSourceType>();
  const { kbDetail, themeMode, mobile = false, setNodeList } = useStore();
  const basePath = useBasePath();
  const { t } = useI18n();

  const redirectUrl =
    typeof window !== 'undefined'
      ? `${window.location.origin}${basePath}${decodeURIComponent(searchParams.get('redirect') || '')}`
      : '';

  const handleLogin = async () => {
    if (!password.trim()) {
      message.error(t('login.enterAccessToken'));
      return;
    }
    setLoading(true);
    try {
      clearCookie();
      postShareV1AuthLoginSimple({
        password,
      }).then(() => {
        getShareV1NodeList().then(res => {
          setNodeList?.((res as any) ?? []);
          message.success(t('login.authSuccess'));
          window.open(redirectUrl, '_self');
        });
      });
    } catch (error) {
      message.error(t('login.authFail'));
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      if (
        authType === ConstsAuthType.AuthTypeEnterprise &&
        sourceType === ConstsSourceType.SourceTypeLDAP
      ) {
        // For LDAP auth, check if both username and password are filled before submitting
        if (username.trim() && password.trim()) {
          handleLDAPLogin();
        }
      } else {
        handleLogin();
      }
    }
  };

  const handleDingTalkLogin = () => {
    clearCookie();
    postShareProV1AuthDingtalk({
      redirect_url: redirectUrl,
    }).then(res => {
      window.location.href = res.url || '/';
    });
  };

  const handleFeishuLogin = () => {
    clearCookie();
    postShareProV1AuthFeishu({
      redirect_url: redirectUrl,
    }).then(res => {
      window.location.href = res.url || '/';
    });
  };

  const handleQiyeweixinLogin = () => {
    clearCookie();
    postShareProV1AuthWecom({
      redirect_url: redirectUrl,
      is_app: isWeComByUA(),
    }).then(res => {
      window.location.href = res.url || '/';
    });
  };

  const handleOAuthLogin = () => {
    clearCookie();
    postShareProV1AuthOauth({
      redirect_url: redirectUrl,
    }).then(res => {
      window.location.href = res.url || '/';
    });
  };

  const handleGitHubLogin = () => {
    clearCookie();
    if (licenseEdition === ConstsLicenseEdition.LicenseEditionFree) {
      postShareV1AuthGithub({
        redirect_url: redirectUrl,
      }).then(res => {
        window.location.href = res.url || '/';
      });
    } else {
      postShareProV1AuthGithub({
        redirect_url: redirectUrl,
      }).then(res => {
        window.location.href = res.url || '/';
      });
    }
  };

  const handleCASLogin = () => {
    postShareProV1AuthCas({
      redirect_url: redirectUrl,
    }).then(res => {
      window.location.href = res.url || '/';
    });
  };

  const handleLDAPLogin = () => {
    setLoading(true);
    try {
      postShareProV1AuthLdap({
        username,
        password,
      }).then(() => {
        getShareV1NodeList().then(res => {
          setNodeList?.((res as any) ?? []);
          message.success(t('login.authSuccess'));
          window.open(redirectUrl, '_self');
        });
      });
    } catch (error) {
      message.error(t('login.authFail'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    getShareV1AuthGet({}).then(res => {
      setAuthType(res?.auth_type);
      setSourceType(res?.source_type);
      setLicenseEdition(res?.license_edition);
      if (res?.auth_type === ConstsAuthType.AuthTypeNull) {
        window.open(redirectUrl, '_self');
      }
    });
  }, []);

  return (
    <>
      <Box
        sx={{
          minHeight: mobile ? `calc(100vh - 120px)` : `calc(100vh - 40px)`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor:
            themeMode === 'dark' ? 'background.default' : 'background.paper',
        }}
      >
        <Box
          sx={{
            width: '100%',
            maxWidth: 400,
            p: 4,
            backdropFilter: 'blur(2px)',
            bgcolor:
              themeMode === 'dark' ? 'background.paper' : 'background.default',
            borderRadius: '10px',
            ...(mobile && {
              m: 3,
            }),
          }}
        >
          <Stack alignItems='center'>
            <Stack alignItems='center' gap={1} sx={{ mb: 5 }}>
              {kbDetail?.settings?.icon ? (
                <img
                  src={getImagePath(kbDetail?.settings?.icon, basePath)}
                  alt='logo'
                  width={40}
                  height={40}
                />
              ) : (
                <Image src={Logo.src} width={40} height={40} alt='logo' />
              )}
              <Box
                sx={{ fontSize: 28, lineHeight: '36px', fontWeight: 'bold' }}
              >
                {kbDetail?.settings?.title}
              </Box>
            </Stack>
            {authType === ConstsAuthType.AuthTypeSimple && (
              <>
                <TextField
                  fullWidth
                  type='password'
                  value={password}
                  autoFocus
                  onChange={e => setPassword(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder={t('login.accessTokenPlaceholder')}
                  disabled={loading}
                  slotProps={{
                    input: {
                      startAdornment: (
                        <InputAdornment position='start'>
                          <IconKoulingrenzheng
                            sx={{ fontSize: 16, width: 24, height: 16 }}
                          />
                        </InputAdornment>
                      ),
                    },
                  }}
                  sx={{
                    borderRadius: '10px',
                    overflow: 'hidden',
                    '& .MuiInputBase-input': {
                      p: 2,
                      lineHeight: '24px',
                      height: '24px',
                      fontFamily: 'Mono',
                    },
                    '& .MuiOutlinedInput-root': {
                      pr: '18px',
                      bgcolor: 'background.paper',
                      '& fieldset': {
                        borderRadius: '10px',
                        borderColor: 'divider',
                        px: 2,
                      },
                    },
                  }}
                />
                <Button
                  fullWidth
                  variant='contained'
                  onClick={handleLogin}
                  sx={{ mt: 5, height: '50px', fontSize: 16 }}
                  disabled={loading || !password.trim()}
                >
                  {loading ? t('login.verifying') : t('login.verifyAccess')}
                </Button>
              </>
            )}

            {authType === ConstsAuthType.AuthTypeEnterprise && (
              <>
                {sourceType === ConstsSourceType.SourceTypeDingTalk && (
                  <IconButton onClick={handleDingTalkLogin}>
                    <IconDingDing sx={{ fontSize: 40 }}></IconDingDing>
                  </IconButton>
                )}
                {sourceType === ConstsSourceType.SourceTypeFeishu && (
                  <IconButton onClick={handleFeishuLogin}>
                    <IconFeishu sx={{ fontSize: 40 }}></IconFeishu>
                  </IconButton>
                )}
                {sourceType === ConstsSourceType.SourceTypeWeCom && (
                  <IconButton onClick={handleQiyeweixinLogin}>
                    <IconQiyeweixin sx={{ fontSize: 28 }}></IconQiyeweixin>
                  </IconButton>
                )}
                {sourceType === ConstsSourceType.SourceTypeOAuth && (
                  <Button
                    fullWidth
                    variant='contained'
                    onClick={handleOAuthLogin}
                    sx={{ height: '50px', fontSize: 16 }}
                  >
                    {t('login.login')}
                  </Button>
                )}
                {sourceType === ConstsSourceType.SourceTypeGitHub && (
                  <Button
                    fullWidth
                    variant='contained'
                    onClick={handleGitHubLogin}
                    startIcon={<IconGitHub1 />}
                    sx={{ height: '50px', fontSize: 16 }}
                  >
                    {t('login.login')}
                  </Button>
                )}

                {sourceType === ConstsSourceType.SourceTypeCAS && (
                  <IconButton onClick={handleCASLogin}>
                    <IconCAS sx={{ fontSize: 40 }}></IconCAS>
                  </IconButton>
                )}

                {sourceType === ConstsSourceType.SourceTypeLDAP && (
                  <Stack spacing={2} width='100%'>
                    {(() => {
                      const textFieldSx = {
                        borderRadius: '10px',
                        overflow: 'hidden',
                        '& .MuiInputBase-input': {
                          p: 2,
                          lineHeight: '24px',
                          height: '24px',
                          fontFamily: 'Mono',
                        },
                        '& .MuiOutlinedInput-root': {
                          pr: '18px',
                          bgcolor: 'background.paper',
                          '& fieldset': {
                            borderRadius: '10px',
                            borderColor: 'divider',
                            px: 2,
                          },
                        },
                      };

                      return (
                        <>
                          <TextField
                            fullWidth
                            type='text'
                            value={username}
                            autoFocus
                            onChange={e => setUsername(e.target.value)}
                            placeholder={t('login.username')}
                            disabled={loading}
                            slotProps={{
                              input: {
                                startAdornment: (
                                  <InputAdornment position='start'>
                                    <IconZhanghao
                                      sx={{
                                        fontSize: 16,
                                        width: 24,
                                        height: 16,
                                      }}
                                    />
                                  </InputAdornment>
                                ),
                              },
                            }}
                            sx={textFieldSx}
                          />
                          <TextField
                            fullWidth
                            type='password'
                            value={password}
                            onChange={e => setPassword(e.target.value)}
                            onKeyDown={handleKeyDown}
                            placeholder={t('login.password')}
                            disabled={loading}
                            slotProps={{
                              input: {
                                startAdornment: (
                                  <InputAdornment position='start'>
                                    <IconMima
                                      sx={{
                                        fontSize: 16,
                                        width: 24,
                                        height: 16,
                                      }}
                                    />
                                  </InputAdornment>
                                ),
                              },
                            }}
                            sx={textFieldSx}
                          />
                          <Button
                            fullWidth
                            variant='contained'
                            onClick={handleLDAPLogin}
                            sx={{
                              mt: 2,
                              height: '50px',
                              fontSize: 16,
                              borderRadius: '10px',
                              boxShadow: '0px 4px 12px rgba(0, 0, 0, 0.1)',
                            }}
                            startIcon={
                              <IconLDAP
                                sx={{
                                  fontSize: 16,
                                  width: 24,
                                  height: 16,
                                  color:
                                    loading ||
                                    !username.trim() ||
                                    !password.trim()
                                      ? ''
                                      : '#e73f3f',
                                }}
                              />
                            }
                            disabled={
                              loading || !username.trim() || !password.trim()
                            }
                          >
                            {loading ? t('login.verifying') : t('login.login')}
                          </Button>
                        </>
                      );
                    })()}
                  </Stack>
                )}
              </>
            )}

            <Box
              sx={{
                textAlign: 'center',
                color: 'text.disabled',
                fontSize: 14,
                lineHeight: '20px',
                mt: 2,
              }}
            >
              {t('login.authRequired')}
            </Box>
          </Stack>
        </Box>
      </Box>
      <FooterProvider showBrand={false} />
    </>
  );
}
