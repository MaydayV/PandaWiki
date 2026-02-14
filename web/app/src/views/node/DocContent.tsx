'use client';

import CommentInput, {
  CommentInputRef,
  ImageItem,
} from '@/components/commentInput';
import { DocWidth } from '@/constant';
import { useBasePath } from '@/hooks';
import { useStore } from '@/provider';
import { copyText } from '@/utils';
import {
  getShareV1CommentList,
  postShareV1Comment,
} from '@/request/ShareComment';
import { ConstsCopySetting, V1ShareNodeDetailResp } from '@/request/types';
import { findAdjacentDocuments } from '@/utils';
import { getImagePath } from '@/utils/getImagePath';
import { Editor, UseTiptapReturn } from '@ctzhian/tiptap';
import { Image, message } from '@ctzhian/ui';
import {
  Box,
  Button,
  Divider,
  IconButton,
  Stack,
  TextField,
  Tooltip,
  alpha,
} from '@mui/material';
import {
  IconFuzhi,
  IconMianbaoxie,
  IconWenjian,
  IconWenjianjia,
} from '@panda-wiki/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import FolderList from './folderList';
import { useI18n } from '@/i18n/useI18n';

dayjs.extend(relativeTime);

const DocContent = ({
  info,
  docWidth,
  editorRef,
  commentList: propsCommentList,
  characterCount,
}: {
  docWidth?: string;
  info?: V1ShareNodeDetailResp;
  editorRef: UseTiptapReturn;
  commentList?: any[];
  characterCount?: number;
}) => {
  const { mobile = false, authInfo, kbDetail, catalogWidth, tree } = useStore();
  const { t } = useI18n();
  const basePath = useBasePath();
  const params = useParams() || {};
  const [commentLoading, setCommentLoading] = useState(false);
  const docId = params.id as string;
  const [commentList, setCommentList] = useState<any[]>(propsCommentList ?? []);
  const [appDetail, setAppDetail] = useState<any>(kbDetail?.settings);
  const {
    control,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm({
    defaultValues: {
      content: '',
      name: '',
    },
  });

  const commentInputRef = useRef<CommentInputRef>(null);
  const [contentFocused, setContentFocused] = useState(false);
  const [commentImages, setCommentImages] = useState<ImageItem[]>([]);

  // 计算上一篇和下一篇文档
  const adjacentDocs = useMemo(() => {
    if (!tree || !docId || info?.type !== 2) {
      return undefined;
    }
    return findAdjacentDocuments(tree, docId);
  }, [tree, docId, info?.type]);

  const getComment = async () => {
    const res = await getShareV1CommentList({ id: docId });
    setCommentList(res.data ?? []);
  };

  useEffect(() => {
    if (
      docId &&
      info?.kb_id &&
      appDetail?.web_app_comment_settings?.is_enable
    ) {
      getComment();
    }
  }, [docId, info?.kb_id, appDetail?.web_app_comment_settings?.is_enable]);

  const onSubmit = handleSubmit(
    async (data: { content: string; name: string }) => {
      setCommentLoading(true);
      let token = '';

      try {
        const Cap = (await import('@cap.js/widget')).default;
        const cap = new Cap({
          apiEndpoint: `${basePath}/share/v1/captcha/`,
        });
        const solution = await cap.solve();
        token = solution.token;
      } catch (error) {
        message.error(t('chat.validationFailed'));
        console.log(error, 'error---------');
        setCommentLoading(false);
        return;
      }

      try {
        // 先上传所有图片
        let imageUrls: string[] = [];
        if (commentImages.length > 0 && commentInputRef.current) {
          imageUrls = await commentInputRef.current.uploadImages();
        }

        await postShareV1Comment({
          content: data.content,
          pic_urls: imageUrls,
          node_id: docId,
          user_name: data.name,
          captcha_token: token,
        });

        getComment();
        reset();
        commentInputRef.current?.clearImages();
        setCommentImages([]);
        message.success(
          appDetail?.web_app_comment_settings?.moderation_enable
            ? t('node.commentPending')
            : t('node.commentSuccess'),
        );
      } catch (error: any) {
        console.log(error.message || t('node.commentFailed'));
      } finally {
        setCommentLoading(false);
      }
    },
  );

  useEffect(() => {
    window.CAP_CUSTOM_WASM_URL =
      window.location.origin + `${basePath}/cap@0.0.6/cap_wasm.min.js`;
  }, []);

  if (!editorRef || !info) return null;

  const renderIp = (ip_address: any = {}) => {
    const { city = '', country = t('common.unknown'), province = '', ip } =
      ip_address;
    const isChina = country === '中国' || country === 'China';
    return (
      <>
        <Box>{ip}</Box>
        <Box sx={{ color: 'text.tertiary', fontSize: 12 }}>
          {isChina ? `${province}-${city}` : `${country}`}
        </Box>
      </>
    );
  };

  const onCopyDocMd = () => {
    let context = editorRef.getMarkdown() || '';
    // 如果设置了追加尾缀，则在复制内容后添加尾缀
    if (
      kbDetail?.settings?.copy_setting === ConstsCopySetting.CopySettingAppend
    ) {
      context += `\n\n-----------------------------------------\n内容来自 ${typeof window !== 'undefined' ? window.location.href : ''}`;
    }
    copyText(context);
  };

  return (
    <Box
      id='doc-content'
      sx={theme => ({
        wordBreak: 'break-all',
        color: 'text.primary',
        position: 'relative',
        zIndex: 1,
        '& ::selection': {
          backgroundColor: `${alpha(
            theme.palette.primary.main,
            0.2,
          )} !important`,
        },
        ...(docWidth === 'full' &&
          !mobile && {
            flexGrow: 1,
            width: 0,
          }),
        ...(docWidth !== 'full' &&
          !mobile && {
            width: DocWidth[docWidth as keyof typeof DocWidth].value,
            maxWidth: `calc(100% - ${catalogWidth}px - 240px - 192px)`,
          }),
        ...(mobile && {
          mx: 'auto',
          marginTop: 3,
          width: '100%',
          px: 3,
        }),
      })}
    >
      <Stack
        direction={'row'}
        alignItems={'flex-start'}
        gap={1}
        sx={{
          fontSize: 30,
          lineHeight: '36px',
          fontWeight: 'bold',
          color: 'text.primary',
          mb: '10px',
        }}
      >
        {info?.meta?.emoji ? (
          <Box sx={{ flexShrink: 0 }}>{info?.meta?.emoji}</Box>
        ) : info?.type === 1 ? (
          <IconWenjianjia sx={{ flexShrink: 0, mt: 0.5 }} />
        ) : (
          <IconWenjian sx={{ flexShrink: 0, mt: 0.5 }} />
        )}
        {info?.name}
      </Stack>
      <Stack
        direction='row'
        justifyContent='space-between'
        alignItems='center'
        sx={{ mb: 4 }}
      >
        <Stack
          direction='row'
          alignItems='center'
          gap={1}
          sx={{
            fontSize: 14,
            color: 'text.tertiary',
          }}
        >
          {info?.created_at && (
            <Box>
              {info?.creator_account && info?.creator_account === 'admin'
                ? t('node.admin')
                : info?.creator_account}{' '}
              {dayjs(info?.created_at).fromNow()}
              {t('node.created')}
            </Box>
          )}
          {info?.updated_at && info.updated_at.slice(0, 1) !== '0' && (
            <>
              <Box>·</Box>
              <Box>
                {info?.editor_account && info?.editor_account === 'admin'
                  ? t('node.admin')
                  : info?.editor_account}{' '}
                {dayjs(info.updated_at).fromNow()}
                {t('node.updated')}
              </Box>
            </>
          )}
          {!!characterCount && characterCount > 0 && (
            <>
              <Box>·</Box>
              <Box>{t('node.words', { count: characterCount })}</Box>
            </>
          )}
          {(info.pv ?? 0) > 0 && (
            <>
              <Box>·</Box>
              <Box>{t('node.views', { count: info.pv ?? 0 })}</Box>
            </>
          )}
        </Stack>
        {info?.type === 2 &&
          kbDetail?.settings?.copy_setting !==
            ConstsCopySetting.CopySettingDisabled && (
            <Tooltip title={t('node.copyMarkdownFormat')} arrow placement='top'>
              <IconButton size='small' onClick={onCopyDocMd}>
                <IconFuzhi sx={{ fontSize: 16 }} />
              </IconButton>
            </Tooltip>
          )}
      </Stack>

      {info?.meta?.summary && (
        <Box
          sx={{
            mb: 6,
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: '10px',
            bgcolor: 'background.paper3',
            p: '20px',
            fontSize: 14,
            lineHeight: '28px',
            backdropFilter: 'blur(5px)',
          }}
        >
          <Box sx={{ fontWeight: 'bold', mb: 2, lineHeight: '22px' }}>
            {t('node.contentSummary')}
          </Box>
          <Box>{info?.meta?.summary}</Box>
        </Box>
      )}
      <Box
        className='editor-container'
        sx={{
          mt: 6,
          '.tiptap.ProseMirror': {
            '.tableWrapper': {
              width:
                docWidth === 'full'
                  ? '100%'
                  : DocWidth[docWidth as keyof typeof DocWidth].value,
              overflowX: 'auto',
              ...(docWidth !== 'full' && {
                maxWidth: '100%',
              }),
              ...(mobile && {
                width: '100%',
              }),
            },
          },
        }}
      >
        {info.type === 2 && editorRef.editor && (
          <Editor editor={editorRef.editor} />
        )}
        {info.type === 1 && <FolderList list={info.list} />}
      </Box>
      {adjacentDocs && (adjacentDocs.prev || adjacentDocs.next) && (
        <Stack
          direction='row'
          justifyContent='space-between'
          alignItems='center'
          sx={{
            mt: 4,
            mb: appDetail?.web_app_comment_settings?.is_enable ? 0 : 2,
            gap: 2,
          }}
        >
          {adjacentDocs.prev ? (
            <Box
              component={Link}
              href={`${basePath}/node/${adjacentDocs.prev.id}`}
              sx={{
                flex: 1,
                display: 'flex',
                alignItems: 'center',
                gap: 1,
                color: 'text.tertiary',
                textDecoration: 'none',
                maxWidth: '48%',
                '&:hover': {
                  color: 'text.primary',
                },
                transition: 'color 0.2s ease-in-out',
              }}
            >
              <IconMianbaoxie
                sx={{
                  flexShrink: 0,
                  fontSize: 14,
                  transform: 'rotate(180deg)',
                }}
              />
              <Box
                sx={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  textAlign: 'left',
                }}
              >
                {t('node.prev')}
                {adjacentDocs.prev.name}
              </Box>
            </Box>
          ) : (
            <Box sx={{ flex: 1 }} />
          )}
          {adjacentDocs.next ? (
            <Box
              component={Link}
              href={`${basePath}/node/${adjacentDocs.next.id}`}
              sx={{
                flex: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'flex-end',
                gap: 1,
                color: 'text.tertiary',
                textDecoration: 'none',
                maxWidth: '48%',
                '&:hover': {
                  color: 'text.primary',
                },
                transition: 'color 0.2s ease-in-out',
              }}
            >
              <Box
                sx={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  textAlign: 'right',
                }}
              >
                {t('node.next')}
                {adjacentDocs.next.name}
              </Box>
              <IconMianbaoxie sx={{ flexShrink: 0, fontSize: 14 }} />
            </Box>
          ) : (
            <Box sx={{ flex: 1 }} />
          )}
        </Stack>
      )}
      {appDetail?.web_app_comment_settings?.is_enable && (
        <>
          {' '}
          <Divider sx={{ my: 4 }} />
          <Box sx={{ fontWeight: 700, fontSize: 18, mb: 3 }}>
            {t('node.commentTitle')}
          </Box>
          <Box
            sx={{
              p: 2,
              border: '1px solid',
              borderColor: contentFocused ? 'text.primary' : 'divider',
              borderRadius: 2,
              transition: 'all 0.2s ease-in-out',
            }}
          >
            <Controller
              name='content'
              control={control}
              rules={{
                required: t('node.commentRequired'),
              }}
              render={({ field }) => (
                <CommentInput
                  value={field.value}
                  onChange={field.onChange}
                  onImagesChange={setCommentImages}
                  ref={commentInputRef}
                  onFocus={() => {
                    setContentFocused(true);
                  }}
                  onBlur={() => {
                    setContentFocused(false);
                    field.onBlur?.();
                  }}
                  placeholder={t('node.commentPlaceholder')}
                  error={!!errors.content}
                  helperText={errors.content?.message}
                />
              )}
            />

            <Divider sx={{ my: 2 }} />
            <Stack
              direction='row-reverse'
              justifyContent='space-between'
              alignItems='center'
              sx={{ fontSize: 14, color: 'text.secondary' }}
            >
              <Button
                variant='contained'
                onClick={onSubmit}
                loading={commentLoading}
              >
                {t('node.send')}
              </Button>
              {!authInfo?.username && (
                <Controller
                  rules={{
                    required: t('node.nicknameRequired'),
                  }}
                  name='name'
                  control={control}
                  render={({ field }) => (
                    <TextField
                      {...field}
                      placeholder={t('node.nicknamePlaceholder')}
                      size='small'
                      sx={{
                        '.MuiOutlinedInput-notchedOutline': {
                          border: '1px solid',
                          borderColor: 'var(--mui-palette-divider) !important',
                        },
                      }}
                      error={!!errors.name}
                      helperText={errors.name?.message}
                    />
                  )}
                />
              )}
            </Stack>
          </Box>
          <Stack gap={1} sx={{ mt: 4 }}>
            {commentList.map((item, index) => (
              <React.Fragment key={item.id}>
                <Stack gap={1}>
                  <Box sx={{ fontSize: 14, fontWeight: 700 }}>
                    {item.info.user_name}
                  </Box>
                  <Box sx={{ fontSize: 14 }}>{item.content}</Box>
                  <Stack direction='row' gap={1}>
                    <Image.PreviewGroup>
                      {(item.pic_urls || []).map((url: string) => (
                        <Image
                          alt={url}
                          src={getImagePath(url, basePath)}
                          width={80}
                          height={80}
                          style={{
                            borderRadius: '4px',
                            objectFit: 'cover',
                            boxShadow: '0px 0px 3px 1px rgba(0,0,5,0.15)',
                            cursor: 'pointer',
                          }}
                          referrerPolicy='no-referrer'
                        />
                      ))}
                    </Image.PreviewGroup>
                  </Stack>
                  <Stack
                    direction='row'
                    justifyContent='flex-end'
                    alignItems='center'
                    gap={2}
                    sx={{
                      color: 'text.tertiary',
                      fontSize: 12,
                    }}
                  >
                    {renderIp(item.ip_address)}
                    <Box>{dayjs(item.created_at).fromNow()}</Box>
                  </Stack>
                </Stack>
                <Divider sx={{ my: 3, color: 'text.tertiary', fontSize: 14 }}>
                  {index !== commentList.length - 1 ? '' : t('node.noMore')}
                </Divider>
              </React.Fragment>
            ))}
          </Stack>
        </>
      )}
    </Box>
  );
};

export default DocContent;
