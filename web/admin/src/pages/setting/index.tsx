import Card from '@/components/Card';
import { useURLSearchParams } from '@/hooks';
import { getApiV1AppDetail } from '@/request/App';
import { getApiV1KnowledgeBaseDetail } from '@/request/KnowledgeBase';
import {
  DomainAppDetailResp,
  DomainKnowledgeBaseDetail,
} from '@/request/types';
import { useAppSelector } from '@/store';
import { Box, CircularProgress, Tab, Tabs, Typography } from '@mui/material';
import { useEffect, useRef, useState } from 'react';
import CardAI from './component/CardAI';
import CardFeedback from './component/CardFeedback';
import CardKB from './component/CardKB';
import CardRobot from './component/CardRobot';
import CardSecurity from './component/CardSecurity';
import CardWeb from './component/CardWeb';
import CardMCP from './component/CardMCP';
import CardGitHubSync from './component/CardGitHubSync';

const SettingTabs: { label: string; id: string }[] = [
  { label: '门户网站', id: 'portal-website' },
  { label: 'AI 机器人', id: 'robot' },
  { label: '问答设置', id: 'ai-setting' },
  { label: '反馈设置', id: 'feedback' },
  { label: '安全设置', id: 'security' },
  { label: '访问控制', id: 'backend-info' },
  { label: 'MCP 设置', id: 'mcp' },
  { label: 'GitHub 同步', id: 'github-sync' },
];

const Setting = () => {
  const { kb_id } = useAppSelector(state => state.config);
  const [searchParams, setSearchParams] = useURLSearchParams();
  const activeTab = searchParams.get('tab') || 'portal-website';
  const [kb, setKb] = useState<DomainKnowledgeBaseDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [url, setUrl] = useState<string>('');
  const [info, setInfo] = useState<DomainAppDetailResp>();
  const kbReqSeqRef = useRef(0);
  const infoReqSeqRef = useRef(0);

  const getInfo = () => {
    // 竞态守卫：记录请求序号，仅应用最新一次请求的响应
    const seq = ++infoReqSeqRef.current;
    getApiV1AppDetail({ kb_id: kb_id!, type: '1' }).then(res => {
      if (seq !== infoReqSeqRef.current) return;
      setInfo(res);
    });
  };

  const getKb = () => {
    if (!kb_id) return;
    // 竞态守卫：记录请求序号，仅应用最新一次请求的响应，防止旧 KB 响应晚到覆盖新数据
    const seq = ++kbReqSeqRef.current;
    setLoading(true);
    getApiV1KnowledgeBaseDetail({ id: kb_id })
      .then(res => {
        if (seq !== kbReqSeqRef.current) return;
        setKb(res);
      })
      .catch(() => {
        // 失败时保留旧数据；首载失败时 kb 仍为 null，配合下方错误态兜底
      })
      .finally(() => {
        if (seq === kbReqSeqRef.current) setLoading(false);
      });
    getInfo();
  };

  const setActiveTab = (tab: string) => {
    setSearchParams({ tab });
  };

  useEffect(() => {
    if (kb) {
      if (kb.access_settings!.base_url) {
        setUrl(kb.access_settings!.base_url);
        return;
      }

      let defaultUrl: string = '';
      const host = kb.access_settings?.hosts?.[0] || '';
      if (!host) return;

      if (
        kb.access_settings!.ssl_ports &&
        kb.access_settings!.ssl_ports.length > 0
      ) {
        defaultUrl = kb.access_settings!.ssl_ports.includes(443)
          ? `https://${host}`
          : `https://${host}:${kb.access_settings!.ssl_ports[0]}`;
      } else if (
        kb.access_settings!.ports &&
        kb.access_settings!.ports.length > 0
      ) {
        defaultUrl = kb.access_settings!.ports.includes(80)
          ? `http://${host}`
          : `http://${host}:${kb.access_settings!.ports[0]}`;
      }

      setUrl(defaultUrl);
    }
  }, [kb]);

  useEffect(() => {
    if (!kb_id) {
      setLoading(false);
      return;
    }
    // kb 切换时先清空旧 KB 数据，避免渲染旧 KB 的设置
    setKb(null);
    getKb();
  }, [kb_id]);

  if (!kb) {
    if (loading) {
      return (
        <Card sx={{ height: 'calc(100vh - 148px)' }}>
          <Box
            sx={{
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexDirection: 'column',
              gap: 1.5,
            }}
          >
            <CircularProgress size={24} />
            <Typography variant='body2' color='text.secondary'>
              正在加载设置...
            </Typography>
          </Box>
        </Card>
      );
    }
    return (
      <Card sx={{ height: 'calc(100vh - 148px)' }}>
        <Box
          sx={{
            height: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexDirection: 'column',
            gap: 1.5,
          }}
        >
          <Typography variant='body2' color='text.secondary'>
            {!kb_id ? '请先选择知识库' : '设置加载失败，请刷新重试'}
          </Typography>
        </Box>
      </Card>
    );
  }

  return (
    <Box
      sx={{
        position: 'relative',
      }}
    >
      <Card sx={{ mb: 2, display: 'flex', justifyContent: 'center' }}>
        <Tabs
          value={activeTab}
          onChange={(event, newValue) => setActiveTab(newValue as string)}
          aria-label='setting tabs'
        >
          {SettingTabs.map(tab => (
            <Tab key={tab.id} label={tab.label} value={tab.id} />
          ))}
        </Tabs>
      </Card>
      <Card
        sx={{
          height: 'calc(100vh - 148px)',
          overflow: 'auto',
        }}
      >
        {activeTab === 'backend-info' && <CardKB />}
        {activeTab === 'ai-setting' && <CardAI kb={kb} />}
        {activeTab === 'security' && (
          <CardSecurity data={info} kb={kb} refresh={getInfo} />
        )}
        {activeTab === 'feedback' && <CardFeedback kb={kb} />}
        {activeTab === 'robot' && <CardRobot kb={kb} url={url} />}
        {activeTab === 'portal-website' && <CardWeb kb={kb} refresh={getKb} />}
        {activeTab === 'mcp' && <CardMCP kb={kb} />}
        {activeTab === 'github-sync' && <CardGitHubSync />}
      </Card>
    </Box>
  );
};
export default Setting;
