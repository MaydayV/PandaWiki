'use client';

import { useState, useRef, useEffect, useMemo } from 'react';
import { useSearchParams } from 'next/navigation';
import { Box, Paper, Fade, Stack, Fab, Zoom } from '@mui/material';
import { styled, alpha } from '@mui/material/styles';
import MarkDown2 from '@/components/markdown2';
import SSEClient from '@/utils/fetch';
import { message } from '@ctzhian/ui';
import { getShareV1AppWechatInfo } from '@/request/ShareChat';
import { IconCopy } from '@/components/icons';
import { postShareV1ChatFeedback } from '@/request/ShareChat';
import { copyText } from '@/utils';
import LoadingIcon from '@/assets/images/loading.png';
import Image from 'next/image';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import { useStore } from '@/provider';
import Feedback from '@/components/feedback';
import { ConstsSourceType, V1WechatAppInfoResp } from '@/request/types';
import { useBasePath } from '@/hooks';
import { useI18n } from '@/i18n/useI18n';
import {
  IconADiancaiWeixuanzhong2,
  IconDiancaiWeixuanzhong,
  IconDianzanXuanzhong1,
  IconDianzanWeixuanzhong,
} from '@panda-wiki/icons';

interface Message {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

// Styled Components
const StyledContainer = styled(Box)(({ theme }) => ({
  display: 'flex',
  flexDirection: 'column',
  height: '100vh',
  background: theme.palette.background.paper3,
  fontFamily: theme.typography.fontFamily,
}));

const StyledMessagesContainer = styled(Box)(({ theme }) => ({
  flex: 1,
  overflowY: 'auto',
  padding: theme.spacing(2.5),
  '&::-webkit-scrollbar': {
    width: 4,
  },
  '&::-webkit-scrollbar-track': {
    background: 'transparent',
  },
  '&::-webkit-scrollbar-thumb': {
    background: (theme.palette.mode === 'dark'
      ? 'rgba(255, 255, 255, 0.3)'
      : 'rgba(0, 0, 0, 0.3)') as any,
    borderRadius: 2,
  },
  '&::-webkit-scrollbar-thumb:hover': {
    background: (theme.palette.mode === 'dark'
      ? 'rgba(255, 255, 255, 0.5)'
      : 'rgba(0, 0, 0, 0.5)') as any,
  },
}));

const StyledMessages = styled(Box)(({ theme }) => ({
  display: 'flex',
  flexDirection: 'column',
  gap: theme.spacing(2),
}));

const StyledMessage = styled(Box, {
  shouldForwardProp: prop => prop !== 'isUser',
})<{ isUser: boolean }>(({ theme, isUser }) => ({
  display: 'flex',
  flexDirection: 'column',
  maxWidth: '100%',
  alignSelf: isUser ? 'flex-end' : 'flex-start',
  animation: 'fadeInUp 0.3s ease-out',
  marginBottom: theme.spacing(1),
  '@keyframes fadeInUp': {
    from: {
      opacity: 0,
      transform: 'translateY(20px)',
    },
    to: {
      opacity: 1,
      transform: 'translateY(0)',
    },
  },
}));

const StyledMessageContent = styled(Paper, {
  shouldForwardProp: prop => prop !== 'isUser',
})<{ isUser: boolean }>(({ theme, isUser }) => ({
  background: isUser ? '#95EA68' : theme.palette.background.paper,
  color: isUser ? '#000' : theme.palette.text.primary,
  padding: theme.spacing(1.5, 2),
  borderRadius: 4,
  boxShadow: '0 1px 2px rgba(0, 0, 0, 0.1)',
  wordWrap: 'break-word',
  maxWidth: '100%',
  position: 'relative',
  '&::before': isUser
    ? {
        content: '""',
        position: 'absolute',
        right: -7,
        top: '50%',
        transform: 'translateY(-50%)',
        width: 0,
        height: 0,
        borderLeft: '8px solid #95EA68',
        borderTop: '8px solid transparent',
        borderBottom: '8px solid transparent',
      }
    : {
        content: '""',
        position: 'absolute',
        left: -7,
        top: '50%',
        transform: 'translateY(-50%)',
        width: 0,
        height: 0,
        borderRight: `8px solid ${theme.palette.background.paper}`,
        borderTop: '8px solid transparent',
        borderBottom: '8px solid transparent',
      },
}));

const StyledMessageLine = styled('div')(() => ({
  margin: '0 0 8px 0',
  lineHeight: 1.5,
  fontSize: '14px',
  '&:last-child': {
    marginBottom: 0,
  },
}));

const StyledTyping = styled('div')(({ theme }) => ({
  display: 'inline-flex',
  alignItems: 'center',
  gap: 4,
  height: 16,
  verticalAlign: 'middle',
  '& span': {
    display: 'inline-block',
    width: 6,
    height: 6,
    borderRadius: '50%',
    background: theme.palette.text.primary,
    opacity: 0.2,
    animation: 'blink 1.4s infinite both',
  },
  '& span:nth-of-type(1)': {
    animationDelay: '0s',
  },
  '& span:nth-of-type(2)': {
    animationDelay: '0.2s',
  },
  '& span:nth-of-type(3)': {
    animationDelay: '0.4s',
  },
  '@keyframes blink': {
    '0%': { opacity: 0.2 },
    '20%': { opacity: 1 },
    '100%': { opacity: 0.2 },
  },
}));

const SOURCE_TO_API = {
  [ConstsSourceType.SourceTypeWechatBot]: getShareV1AppWechatInfo,
};

const ChatLoading = ({ onClick }: { onClick: () => void }) => {
  const { t } = useI18n();
  return (
    <Stack
      direction='row'
      alignItems={'center'}
      sx={{
        px: 1.5,
        py: 1,
        borderRadius: '18px',
        bgcolor: theme => alpha(theme.palette.background.paper, 0.9),
        position: 'fixed',
        bottom: 80,
        left: '50%',
        transform: 'translateX(-50%)',
        gap: 1,
        fontSize: 12,
        border: '1px solid',
        borderColor: 'divider',
        boxShadow: '0px 4px 10px 0px rgba(0,0,0,0.1)',
      }}
      onClick={onClick}
    >
      <Box sx={{ position: 'relative' }}>
        <Image
          src={LoadingIcon.src}
          alt='loading'
          width={20}
          height={20}
          style={{
            display: 'block',
            animation: 'loadingRotate 1s linear infinite',
          }}
        />
        <Box
          sx={{
            width: 6,
            height: 6,
            bgcolor: 'primary.main',
            borderRadius: '1px',
            position: 'absolute',
            top: 7,
            left: 7,
          }}
        />
      </Box>
      {t('chat.stopAnswer')}
    </Stack>
  );
};

const H5Chat = () => {
  const searchParams = useSearchParams();
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const sseClientRef = useRef<SSEClient<{
    type: string;
    content: string;
    chunk_result: Message[];
  }> | null>(null);
  const [appSetting, setAppSetting] = useState<V1WechatAppInfoResp | null>(
    null,
  );
  const [loading, setLoading] = useState(true);
  const [answer, setAnswer] = useState('');
  const [score, setScore] = useState(0);
  const [message_id, setMessageId] = useState('');
  const [open, setOpen] = useState(false);
  const [question, setQuestion] = useState('');
  const { kbDetail } = useStore();
  const { t } = useI18n();
  const basePath = useBasePath();

  const [showScrollTop, setShowScrollTop] = useState(false);

  const isFeedbackEnabled = useMemo(() => {
    return appSetting?.feedback_enable ?? false;
  }, [appSetting]);

  const disclaimerContent = useMemo(() => {
    return (
      appSetting?.disclaimer_content ??
      kbDetail?.settings?.disclaimer_settings?.content ??
      ''
    );
  }, [appSetting, kbDetail]);

  useEffect(() => {
    messagesContainerRef.current?.scrollTo({
      top: messagesContainerRef.current.scrollHeight,
      behavior: 'smooth',
    });
  }, [question, answer]);

  const handleScroll = () => {
    if (messagesContainerRef.current) {
      setShowScrollTop(messagesContainerRef.current.scrollTop > 100);
    }
  };

  useEffect(() => {
    const source = searchParams.get('source_type');
    const api = SOURCE_TO_API[source as keyof typeof SOURCE_TO_API];
    if (api) {
      api().then(res => {
        setAppSetting(res);
      });
    }
  }, [searchParams]);

  const handleScore = async (
    message_id: string,
    score: number,
    type?: string,
    content?: string,
  ) => {
    const data: any = {
      conversation_id: searchParams.get('id'),
      message_id,
      score,
    };
    if (type) data.type = type;
    if (content) data.feedback_content = content;
    await postShareV1ChatFeedback(data);
    setScore(score);
    message.success(t('chat.feedbackSuccess'));
  };

  const scrollToTop = () => {
    messagesContainerRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
  };

  useEffect(() => {
    const messagesContainer = messagesContainerRef.current;
    if (!messagesContainer) return;
    messagesContainer.addEventListener('scroll', handleScroll);
    return () => messagesContainer.removeEventListener('scroll', handleScroll);
  }, []);

  useEffect(() => {
    // 从URL获取q参数
    const id = searchParams.get('id');

    if (id) {
      // 设置消息
      setTimeout(() => {
        chatAnswer();
      });
    }
  }, [searchParams]);

  const handleSearchAbort = () => {
    sseClientRef.current?.unsubscribe();
    setLoading(false);
  };

  const chatAnswer = async () => {
    setLoading(true);
    if (sseClientRef.current) {
      sseClientRef.current.subscribe('', ({ type, content }) => {
        if (type === 'message_id') {
          setMessageId(prev => {
            return prev + content;
          });
        } else if (type === 'feedback_score') {
          setScore(+content);
        } else if (type === 'question') {
          setQuestion(prev => {
            return prev + content;
          });
        } else if (type === 'answer') {
          setAnswer(prev => {
            return prev + content;
          });
        } else if (type === 'error') {
          setLoading(false);
          setAnswer(prev => {
            if (content) {
              return (
                prev + `\n\n${t('chat.answerError')}: <error>${content}</error>`
              );
            }
            return prev + `\n\n${t('chat.answerRetry')}`;
          });
          if (content) message.error(content);
        } else if (type === 'done') {
          setLoading(false);
        }
      });
    }
  };

  useEffect(() => {
    const id = searchParams.get('id');

    sseClientRef.current = new SSEClient({
      url: `${basePath}/share/v1/app/wechat/service/answer?id=${id}`,
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      onCancel: () => {
        setLoading(false);
        setAnswer(() => {
          return '';
        });
      },
    });
    return () => {
      sseClientRef.current?.unsubscribe();
    };
  }, [searchParams, basePath]);

  return (
    <StyledContainer>
      {/* 聊天消息区域 */}
      <StyledMessagesContainer ref={messagesContainerRef}>
        <StyledMessages>
          <Fade in={true} timeout={300}>
            <StyledMessage isUser={true}>
              <StyledMessageContent isUser={true} elevation={1}>
                <StyledMessageLine>
                  {question
                    ? question
                    : loading && (
                        <StyledTyping>
                          <span />
                          <span />
                          <span />
                        </StyledTyping>
                      )}
                </StyledMessageLine>
              </StyledMessageContent>
            </StyledMessage>
          </Fade>
          {answer && (
            <Fade in={true} timeout={300}>
              <StyledMessage isUser={false}>
                <StyledMessageContent isUser={false} elevation={1}>
                  <MarkDown2 content={answer} />
                  {!loading && (
                    <Stack
                      direction='column'
                      alignItems='flex-start'
                      justifyContent='space-between'
                      gap={1}
                      sx={{
                        fontSize: 12,
                        color: 'text.tertiary',
                        mt: 1,
                      }}
                    >
                      <Box>{disclaimerContent}</Box>
                      <Stack direction='row' alignItems='center' gap={3}>
                        <IconCopy
                          sx={{ cursor: 'pointer', color: 'text.primary' }}
                          onClick={() => {
                            copyText(answer);
                          }}
                        />

                        {isFeedbackEnabled && (
                          <>
                            {score === 1 && (
                              <IconDianzanXuanzhong1
                                sx={{ cursor: 'pointer' }}
                              />
                            )}
                            {score !== 1 && (
                              <IconDianzanWeixuanzhong
                                sx={{ cursor: 'pointer' }}
                                onClick={() => {
                                  if (score === 0) handleScore(message_id, 1);
                                }}
                              />
                            )}
                            {score !== -1 && (
                              <IconDiancaiWeixuanzhong
                                sx={{ cursor: 'pointer' }}
                                onClick={() => {
                                  if (score === 0) {
                                    setOpen(true);
                                  }
                                }}
                              />
                            )}
                            {score === -1 && (
                              <IconADiancaiWeixuanzhong2
                                sx={{ cursor: 'pointer' }}
                              />
                            )}
                          </>
                        )}
                      </Stack>
                    </Stack>
                  )}
                </StyledMessageContent>
              </StyledMessage>
            </Fade>
          )}
        </StyledMessages>
      </StyledMessagesContainer>
      <Feedback
        open={open}
        onClose={() => setOpen(false)}
        onSubmit={handleScore}
        data={{ message_id: message_id }}
        tags={appSetting?.feedback_type}
      />
      {loading && <ChatLoading onClick={handleSearchAbort} />}
      <Zoom in={showScrollTop}>
        <Fab
          size='small'
          onClick={scrollToTop}
          sx={{
            backgroundColor: 'background.paper3',
            color: 'text.primary',
            position: 'fixed',
            bottom: 66,
            right: 16,
            zIndex: 1000,
          }}
        >
          <KeyboardArrowUpIcon sx={{ fontSize: 24 }} />
        </Fab>
      </Zoom>
    </StyledContainer>
  );
};

export default H5Chat;
