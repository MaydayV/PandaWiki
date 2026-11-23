import { ITreeItem } from '@/api';
import Cascader from '@/components/Cascader';
import { VersionCanUse } from '@/components/VersionMask';
import { BUSINESS_VERSION_PERMISSION } from '@/constant/version';
import VersionPublish from '@/pages/release/components/VersionPublish';
import { getApiV1NodeDetail, postApiV1Node } from '@/request';
import { postApiProV1NodeTranslationAuto } from '@/request/NodeTranslation';
import { V1NodeDetailResp } from '@/request/types';
import { useAppSelector } from '@/store';
import { addOpacityToColor, getShortcutKeyText } from '@/utils';
import { Ellipsis, message, Modal } from '@ctzhian/ui';
import TranslateIcon from '@mui/icons-material/Translate';
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Chip,
  FormControlLabel,
  IconButton,
  MenuItem,
  Select,
  Skeleton,
  Stack,
  styled,
  Switch,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import {
  IconBaocun,
  IconDaochu,
  IconGengduo,
  IconMuluzhankai,
} from '@panda-wiki/icons';
import dayjs from 'dayjs';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useOutletContext } from 'react-router';
import { WrapContext } from '..';
import DocAddByCustomText from '../../component/DocAddByCustomText';
import DocDelete from '../../component/DocDelete';

interface HeaderProps {
  edit: boolean;
  detail: V1NodeDetailResp;
  updateDetail: (detail: V1NodeDetailResp) => void;
  handleSave: () => void;
  handleExport: (type: string) => void;
}

const Header = ({
  edit,
  detail,
  updateDetail,
  handleSave,
  handleExport,
}: HeaderProps) => {
  const theme = useTheme();
  const navigate = useNavigate();
  const firstLoad = useRef(true);
  const [wikiUrl, setWikiUrl] = useState<string>('');
  const wikiUrlRef = useRef(wikiUrl);

  useEffect(() => {
    wikiUrlRef.current = wikiUrl;
  }, [wikiUrl]);

  const { kb_id, nav_id, license, kbList } = useAppSelector(
    state => state.config,
  );

  const currentKb = useMemo(() => {
    return kbList?.find(item => item.id === kb_id);
  }, [kbList, kb_id]);

  const {
    catalogOpen,
    nodeDetail,
    setCatalogOpen,
    refreshCatalog,
    catalogData,
  } = useOutletContext<WrapContext>();

  const [renameOpen, setRenameOpen] = useState(false);
  const [delOpen, setDelOpen] = useState(false);
  const [publishOpen, setPublishOpen] = useState(false);
  const [translateOpen, setTranslateOpen] = useState(false);
  const [translating, setTranslating] = useState(false);
  const [sourceLanguage, setSourceLanguage] = useState('zh-CN');
  const [targetLanguages, setTargetLanguages] = useState<string[]>(['en-US']);
  const [overwriteTranslation, setOverwriteTranslation] = useState(true);
  const [publishAfterTranslate, setPublishAfterTranslate] = useState(true);

  const [showSaveTip, setShowSaveTip] = useState(false);

  const existingTranslations = useMemo(() => {
    const translations = (
      detail.meta as
        | (NonNullable<V1NodeDetailResp['meta']> & {
            translations?: Record<
              string,
              { name?: string; content?: string; summary?: string }
            >;
          })
        | undefined
    )?.translations;
    if (!translations) return [];
    return Object.keys(translations).filter(key => {
      const item = translations[key];
      return Boolean(
        item &&
          (item.name?.trim() || item.content?.trim() || item.summary?.trim()),
      );
    });
  }, [detail.meta]);

  const toggleTargetLanguage = (lang: string) => {
    setTargetLanguages(prev =>
      prev.includes(lang)
        ? prev.filter(item => item !== lang)
        : [...prev, lang],
    );
  };

  const handleAutoTranslate = async () => {
    if (!kb_id || !detail.id) return;
    if (targetLanguages.length === 0) {
      message.warning('请至少选择一个目标语言');
      return;
    }
    setTranslating(true);
    try {
      const res = await postApiProV1NodeTranslationAuto({
        kb_id,
        node_id: detail.id,
        source_language: sourceLanguage,
        target_languages: targetLanguages,
        overwrite: overwriteTranslation,
        publish_to_latest_release: publishAfterTranslate,
      });
      if (res?.translated_languages && res.translated_languages.length > 0) {
        const latest = await getApiV1NodeDetail({ id: detail.id, kb_id });
        if (latest) {
          updateDetail({
            ...latest,
            ...(res.published ? { status: 2 } : {}),
          });
        } else if (res.published) {
          updateDetail({ status: 2 });
        }
        message.success(
          res.published
            ? `翻译完成并已发布（${res.translated_languages.join(', ')}）`
            : `翻译完成（${res.translated_languages.join(', ')}）`,
        );
      } else {
        message.info('目标语言已有译文或与源语言相同，未重新生成');
      }
      setTranslateOpen(false);
    } finally {
      setTranslating(false);
    }
  };

  const isBusiness = useMemo(() => {
    return BUSINESS_VERSION_PERMISSION.includes(license.edition!);
  }, [license]);

  useEffect(() => {
    if (currentKb?.access_settings?.base_url) {
      setWikiUrl(currentKb.access_settings.base_url);
      return;
    }
    const host = currentKb?.access_settings?.hosts?.[0] || '';
    if (host === '') return;
    const { ssl_ports = [], ports = [] } = currentKb?.access_settings || {};

    if (ssl_ports) {
      if (ssl_ports.includes(443)) setWikiUrl(`https://${host}`);
      else if (ssl_ports.length > 0)
        setWikiUrl(`https://${host}:${ssl_ports[0]}`);
    } else if (ports) {
      if (ports.includes(80)) setWikiUrl(`http://${host}`);
      else if (ports.length > 0) setWikiUrl(`http://${host}:${ports[0]}`);
    }
  }, [currentKb]);

  const handlePublish = useCallback(() => {
    if (nodeDetail?.status === 2 && !edit) {
      message.info('当前已是最新版本！');
    } else {
      handleSave();
      setTimeout(() => {
        setPublishOpen(true);
      }, 200);
    }
  }, [nodeDetail, edit]);

  const handleDeleteAndNavigate = async () => {
    // 深度优先遍历树，按照目录顺序收集所有文档
    const collectDocs = (items: ITreeItem[]): ITreeItem[] => {
      const docs: ITreeItem[] = [];
      // 先对 items 按 order 排序，与 Catalog 渲染顺序保持一致
      const sortedItems = [...items].sort(
        (a, b) => (a.order ?? 0) - (b.order ?? 0),
      );
      sortedItems.forEach(item => {
        if (item.type === 2) {
          // 是文档，添加到列表
          docs.push(item);
        }
        if (item.children && item.children.length > 0) {
          // 递归处理子节点
          docs.push(...collectDocs(item.children));
        }
      });
      return docs;
    };

    // 使用删除前的原始数据查找下一个文档
    const allDocs = collectDocs(catalogData);

    // 找到下一个文档
    let nextDoc = null;
    if (allDocs.length > 0) {
      // 找到当前文档在列表中的索引
      const currentIndex = allDocs.findIndex(
        (doc: ITreeItem) => doc.id === detail.id,
      );

      if (currentIndex !== -1) {
        // 如果当前文档不是最后一个，选择下一个文档
        if (currentIndex < allDocs.length - 1) {
          nextDoc = allDocs[currentIndex + 1];
        }
        // 如果当前文档是最后一个但不是唯一的，选择前一个文档
        else if (allDocs.length > 1) {
          nextDoc = allDocs[currentIndex - 1];
        }
      }
    }

    // 刷新目录数据（删除后更新目录显示）
    await refreshCatalog();

    // 导航到下一个文档或首页
    if (nextDoc) {
      // 有其他文档，导航到下一个文档
      navigate(`/doc/editor/${nextDoc.id}`);
    } else {
      // 没有其他文档，回到首页
      navigate('/');
    }
  };

  useEffect(() => {
    if (nodeDetail?.updated_at && !firstLoad.current) {
      setShowSaveTip(true);
      setTimeout(() => {
        setShowSaveTip(false);
      }, 1500);
    }
    firstLoad.current = false;
  }, [nodeDetail?.updated_at]);

  return (
    <Box sx={{ p: 1 }}>
      <Stack
        direction={'row'}
        alignItems={'center'}
        gap={1}
        justifyContent={'space-between'}
        sx={{ height: '40px' }}
      >
        {!catalogOpen && (
          <Stack
            alignItems='center'
            justifyContent='space-between'
            onClick={() => setCatalogOpen(true)}
            sx={{
              cursor: 'pointer',
              color: 'text.tertiary',
              ':hover': {
                color: 'text.primary',
              },
            }}
          >
            <IconMuluzhankai
              sx={{
                fontSize: 24,
              }}
            />
          </Stack>
        )}
        {detail.meta?.content_type === 'md' && (
          <Box
            component={'span'}
            sx={{
              fontSize: 10,
              color: 'white',
              background: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
              px: 1,
              flexShrink: 0,
              fontWeight: '500',
              borderRadius: '4px',
              height: '20px',
              lineHeight: '20px',
              display: 'inline-block',
            }}
          >
            MD
          </Box>
        )}
        <Stack sx={{ width: 0, flex: 1 }}>
          {detail?.name ? (
            <Ellipsis sx={{ fontSize: 14, fontWeight: 'bold' }}>
              <Box
                component='span'
                sx={{ cursor: 'pointer' }}
                // onClick={() => setRenameOpen(true)}
              >
                {detail.name}
              </Box>
            </Ellipsis>
          ) : (
            <Skeleton variant='text' width={300} height={24} />
          )}
          <Stack
            direction={'row'}
            alignItems={'center'}
            gap={0.5}
            sx={{ fontSize: 12, color: 'text.tertiary' }}
          >
            <IconBaocun sx={{ fontSize: 12 }} />
            {showSaveTip ? (
              '已保存'
            ) : nodeDetail?.updated_at ? (
              dayjs(nodeDetail.updated_at).format('YYYY-MM-DD HH:mm:ss')
            ) : (
              <Skeleton variant='text' width={100} height={24} />
            )}
          </Stack>
        </Stack>
        <Stack direction={'row'} gap={1}>
          <Cascader
            list={[
              {
                key: 'copy',
                textSx: { flex: 1 },
                label: <StyledMenuSelect>创建副本</StyledMenuSelect>,
                onClick: () => {
                  if (kb_id) {
                    postApiV1Node({
                      name: detail.name + ' [副本]',
                      content: detail.content,
                      kb_id,
                      nav_id: nav_id || detail.nav_id || '',
                      parent_id: detail.parent_id || undefined,
                      type: 2,
                      emoji: detail.meta?.emoji,
                    }).then(res => {
                      message.success('复制成功');
                      window.open(`/doc/editor/${res.id}`, '_blank');
                    });
                  }
                },
              },
              {
                key: 'front_doc',
                textSx: { flex: 1 },
                label: <StyledMenuSelect>前台查看</StyledMenuSelect>,
                onClick: () => {
                  if (detail.status !== 2 && !detail.publisher_id) {
                    message.warning('当前文档未发布，无法查看前台文档');
                    return;
                  }
                  window.open(
                    `${wikiUrlRef.current}/node/${detail.id}`,
                    '_blank',
                  );
                },
              },
              {
                key: 'version',
                textSx: { flex: 1 },
                label: (
                  <StyledMenuSelect disabled={!isBusiness}>
                    历史版本
                    <VersionCanUse permission={BUSINESS_VERSION_PERMISSION} />
                  </StyledMenuSelect>
                ),
                onClick: () => {
                  if (isBusiness) {
                    navigate(`/doc/editor/history/${detail.id}`);
                  }
                },
              },
              {
                key: 'rename',
                textSx: { flex: 1 },
                label: <StyledMenuSelect>重命名</StyledMenuSelect>,
                onClick: () => {
                  setRenameOpen(true);
                },
              },
              {
                key: 'delete',
                textSx: { flex: 1 },
                label: <StyledMenuSelect type='error'>删除</StyledMenuSelect>,
                onClick: () => {
                  setDelOpen(true);
                },
              },
            ]}
            context={
              <IconButton
                size='small'
                disabled={!detail.name}
                sx={{ flexShrink: 0 }}
              >
                <IconGengduo sx={{ fontSize: 14 }} />
              </IconButton>
            }
          />
          <Button
            size='small'
            variant='outlined'
            disabled={!detail.id || !detail.name}
            startIcon={<TranslateIcon sx={{ fontSize: 14 }} />}
            onClick={() => setTranslateOpen(true)}
          >
            AI 翻译
          </Button>
          <Cascader
            list={[
              {
                key: 'html',
                label: (
                  <Stack
                    direction={'row'}
                    alignItems={'center'}
                    gap={1}
                    sx={{
                      fontSize: 14,
                      px: 2,
                      lineHeight: '40px',
                      height: 40,
                      width: 140,
                      borderRadius: '5px',
                      cursor: 'pointer',
                      ':hover': {
                        bgcolor: addOpacityToColor(
                          theme.palette.primary.main,
                          0.1,
                        ),
                      },
                    }}
                  >
                    导出 HTML
                  </Stack>
                ),
                onClick: () => handleExport('html'),
              },
              {
                key: 'md',
                label: (
                  <Stack
                    direction={'row'}
                    alignItems={'center'}
                    gap={1}
                    sx={{
                      fontSize: 14,
                      px: 2,
                      lineHeight: '40px',
                      height: 40,
                      width: 140,
                      borderRadius: '5px',
                      cursor: 'pointer',
                      ':hover': {
                        bgcolor: addOpacityToColor(
                          theme.palette.primary.main,
                          0.1,
                        ),
                      },
                    }}
                  >
                    导出 Markdown
                  </Stack>
                ),
                onClick: () => handleExport('md'),
              },
            ]}
            context={
              <Button
                size='small'
                variant='outlined'
                disabled={!detail.name}
                startIcon={<IconDaochu sx={{ fontSize: 14 }} />}
              >
                导出
              </Button>
            }
          />
          <Cascader
            list={[
              {
                key: 'save',
                label: (
                  <Tooltip
                    title={<Box>{getShortcutKeyText(['ctrl', 's'])}</Box>}
                    placement='right'
                    arrow
                  >
                    <Stack
                      direction={'row'}
                      alignItems={'center'}
                      gap={1}
                      sx={{
                        fontSize: 14,
                        px: 2,
                        lineHeight: '40px',
                        height: 40,
                        width: 140,
                        borderRadius: '5px',
                        cursor: 'pointer',
                        ':hover': {
                          bgcolor: addOpacityToColor(
                            theme.palette.primary.main,
                            0.1,
                          ),
                        },
                      }}
                    >
                      保存
                    </Stack>
                  </Tooltip>
                ),
                onClick: handleSave,
              },
              {
                key: 'save_publish',
                label: (
                  <Stack
                    direction={'row'}
                    alignItems={'center'}
                    gap={1}
                    sx={{
                      fontSize: 14,
                      px: 2,
                      lineHeight: '40px',
                      height: 40,
                      width: 140,
                      borderRadius: '5px',
                      cursor: 'pointer',
                      ':hover': {
                        bgcolor: addOpacityToColor(
                          theme.palette.primary.main,
                          0.1,
                        ),
                      },
                    }}
                  >
                    保存并发布
                  </Stack>
                ),
                onClick: handlePublish,
              },
            ]}
            context={
              <Button
                size='small'
                variant='contained'
                disabled={!detail.name}
                startIcon={<IconBaocun sx={{ fontSize: 14 }} />}
              >
                保存
              </Button>
            }
          />
        </Stack>
      </Stack>
      <DocAddByCustomText
        type={detail.type}
        open={renameOpen}
        onClose={() => {
          setRenameOpen(false);
        }}
        data={detail}
        setDetail={updateDetail}
      />
      <VersionPublish
        open={publishOpen}
        defaultSelected={[detail.id!]}
        onClose={() => setPublishOpen(false)}
        refresh={() =>
          updateDetail({
            status: 2,
          })
        }
      />
      <DocDelete
        open={delOpen}
        onClose={() => {
          setDelOpen(false);
        }}
        onDeleted={handleDeleteAndNavigate}
        data={[
          {
            ...detail,
            emoji: detail.meta?.emoji || '',
            parent_id: '',
            summary: detail.meta?.summary || '',
            position: 0,
            status: 1,
          },
        ]}
      />
      <Modal
        open={translateOpen}
        onCancel={() => {
          if (!translating) setTranslateOpen(false);
        }}
        title='多语言 AI 翻译'
        okText={translating ? '翻译中...' : '开始翻译'}
        okButtonProps={{
          loading: translating,
          disabled: translating || targetLanguages.length === 0,
        }}
        cancelButtonProps={{ disabled: translating }}
        onOk={handleAutoTranslate}
      >
        <Stack spacing={2}>
          {edit && (
            <Alert severity='warning'>
              当前存在未保存的编辑内容，建议先点击“保存”后再执行 AI 翻译。
            </Alert>
          )}
          <Stack
            direction='row'
            spacing={1}
            alignItems='center'
            flexWrap='wrap'
          >
            <Typography variant='body2' color='text.secondary'>
              已有译文：
            </Typography>
            {existingTranslations.length > 0 ? (
              existingTranslations.map(lang => (
                <Chip key={lang} label={lang} size='small' variant='outlined' />
              ))
            ) : (
              <Typography variant='body2' color='text.secondary'>
                暂无
              </Typography>
            )}
          </Stack>
          <Box>
            <Typography variant='body2' sx={{ mb: 0.75, fontWeight: 500 }}>
              文档源语言
            </Typography>
            <Select
              size='small'
              fullWidth
              value={sourceLanguage}
              disabled={translating}
              onChange={e => setSourceLanguage(e.target.value)}
            >
              <MenuItem value='zh-CN'>简体中文 (zh-CN)</MenuItem>
              <MenuItem value='en-US'>English (en-US)</MenuItem>
              <MenuItem value='auto'>自动识别 (auto)</MenuItem>
            </Select>
          </Box>
          <Box>
            <Typography variant='body2' sx={{ mb: 0.5, fontWeight: 500 }}>
              目标翻译语言
            </Typography>
            <Stack direction='row' spacing={2}>
              <FormControlLabel
                control={
                  <Checkbox
                    size='small'
                    disabled={translating}
                    checked={targetLanguages.includes('en-US')}
                    onChange={() => toggleTargetLanguage('en-US')}
                  />
                }
                label='英文 (en-US)'
              />
              <FormControlLabel
                control={
                  <Checkbox
                    size='small'
                    disabled={translating}
                    checked={targetLanguages.includes('zh-CN')}
                    onChange={() => toggleTargetLanguage('zh-CN')}
                  />
                }
                label='简体中文 (zh-CN)'
              />
            </Stack>
          </Box>
          <Stack>
            <FormControlLabel
              control={
                <Switch
                  size='small'
                  disabled={translating}
                  checked={overwriteTranslation}
                  onChange={e => setOverwriteTranslation(e.target.checked)}
                />
              }
              label='覆盖已有译文'
            />
            <FormControlLabel
              control={
                <Switch
                  size='small'
                  disabled={translating}
                  checked={publishAfterTranslate}
                  onChange={e => setPublishAfterTranslate(e.target.checked)}
                />
              }
              label='翻译完成后自动发布'
            />
          </Stack>
        </Stack>
      </Modal>
    </Box>
  );
};

const StyledMenuSelect = styled('div')<{
  disabled?: boolean;
  type?: 'default' | 'error';
}>(({ theme, disabled = false, type = 'default' }) => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between ',
  fontSize: 14,
  padding: theme.spacing(0, 2),
  lineHeight: '40px',
  height: 40,
  minWidth: 106,
  borderRadius: '5px',
  color: disabled
    ? theme.palette.text.secondary
    : type === 'error'
      ? theme.palette.error.main
      : theme.palette.text.primary,
  cursor: disabled ? 'not-allowed' : 'pointer',
  ':hover': {
    backgroundColor: disabled
      ? 'transparent'
      : type === 'error'
        ? addOpacityToColor(theme.palette.error.main, 0.1)
        : addOpacityToColor(theme.palette.primary.main, 0.1),
  },
}));

export default Header;
