import {
  DomainGetKBReleaseDocsResp,
  getApiV1KnowledgeBaseReleaseDocs,
} from '@/request/KnowledgeBase';
import { DomainKBReleaseListItemResp, DomainNodeType } from '@/request/types';
import { useAppSelector } from '@/store';
import { Modal, Table } from '@ctzhian/ui';
import { Box, Stack, Tab, Tabs } from '@mui/material';
import dayjs from 'dayjs';
import { useEffect, useMemo, useState } from 'react';

type InspectTab = 'docs' | 'diff';

interface VersionInspectProps {
  open: boolean;
  onClose: () => void;
  data: DomainKBReleaseListItemResp | null;
  defaultTab?: InspectTab;
}

const diffLabelMap: Record<string, { label: string; color: string }> = {
  added: { label: '新增', color: 'success.main' },
  removed: { label: '删除', color: 'error.main' },
  changed: { label: '修改', color: 'warning.main' },
  unchanged: { label: '未变更', color: 'text.tertiary' },
};

const VersionInspect = ({
  open,
  onClose,
  data,
  defaultTab = 'docs',
}: VersionInspectProps) => {
  const { kb_id } = useAppSelector(state => state.config);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState<InspectTab>(defaultTab);
  const [detail, setDetail] = useState<DomainGetKBReleaseDocsResp | null>(null);

  useEffect(() => {
    if (!open) return;
    setTab(defaultTab);
  }, [open, defaultTab]);

  useEffect(() => {
    if (!open) {
      setDetail(null);
      return;
    }
    if (!kb_id || !data?.id) {
      setDetail(null);
      return;
    }
    setLoading(true);
    getApiV1KnowledgeBaseReleaseDocs({
      kb_id,
      release_id: data.id,
    })
      .then(res => {
        setDetail(res);
      })
      .catch(() => {
        setDetail(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [open, kb_id, data?.id]);

  const docsColumns = [
    {
      dataIndex: 'name',
      title: '文档名称',
      render: (text: string, record: { type: number }) => (
        <Stack direction='row' alignItems='center' gap={1}>
          <Box>{text}</Box>
          <Box
            sx={{
              fontSize: 12,
              px: 0.75,
              borderRadius: 1,
              bgcolor: 'background.paper3',
              color: 'text.tertiary',
            }}
          >
            {record.type === DomainNodeType.NodeTypeFolder ? '文件夹' : '文档'}
          </Box>
        </Stack>
      ),
    },
    {
      dataIndex: 'updated_at',
      title: '更新时间',
      width: 180,
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      dataIndex: 'node_release_id',
      title: '文档版本 ID',
      width: 260,
      render: (text: string) => (
        <Box sx={{ color: 'text.tertiary', fontFamily: 'monospace' }}>{text}</Box>
      ),
    },
  ];

  const diffColumns = [
    {
      dataIndex: 'name',
      title: '文档名称',
    },
    {
      dataIndex: 'diff_type',
      title: '变更类型',
      width: 120,
      render: (value: string) => {
        const config = diffLabelMap[value] || {
          label: value,
          color: 'text.tertiary',
        };
        return (
          <Box
            sx={{
              display: 'inline-flex',
              px: 1,
              py: 0.25,
              borderRadius: 10,
              fontSize: 12,
              bgcolor: 'background.paper3',
              color: config.color,
            }}
          >
            {config.label}
          </Box>
        );
      },
    },
    {
      dataIndex: 'current_node_release_id',
      title: '当前版本',
      width: 220,
      render: (text: string) => (
        <Box sx={{ color: 'text.tertiary', fontFamily: 'monospace' }}>
          {text || '-'}
        </Box>
      ),
    },
    {
      dataIndex: 'previous_node_release_id',
      title: '上一版本',
      width: 220,
      render: (text: string) => (
        <Box sx={{ color: 'text.tertiary', fontFamily: 'monospace' }}>
          {text || '-'}
        </Box>
      ),
    },
  ];

  const stats = useMemo(() => {
    return (
      detail?.stats || {
        added: 0,
        removed: 0,
        changed: 0,
        unchanged: 0,
      }
    );
  }, [detail]);

  return (
    <Modal
      title={`版本详情：${data?.tag || '-'}`}
      open={open}
      width={900}
      onCancel={onClose}
      onOk={onClose}
      okText='关闭'
    >
      <Stack direction='row' gap={2} sx={{ color: 'text.tertiary', mb: 1 }}>
        <Box>版本号：{data?.tag || '-'}</Box>
        <Box>发布者：{data?.publisher_account || '-'}</Box>
        <Box>
          发布时间：
          {data?.created_at
            ? dayjs(data.created_at).format('YYYY-MM-DD HH:mm:ss')
            : '-'}
        </Box>
      </Stack>
      <Box sx={{ mb: 1, color: 'text.tertiary' }}>备注：{data?.message || '-'}</Box>
      <Tabs value={tab} onChange={(_, value) => setTab(value)}>
        <Tab value='docs' label='版本文档' />
        <Tab value='diff' label='版本对比' />
      </Tabs>
      {tab === 'diff' && !detail?.previous_release_id && (
        <Box sx={{ py: 1, color: 'text.tertiary' }}>
          当前版本没有上一历史版本，暂时无法对比。
        </Box>
      )}
      {tab === 'diff' && (
        <Stack direction='row' gap={1} sx={{ py: 1 }}>
          <Box sx={{ color: 'success.main' }}>新增 {stats.added}</Box>
          <Box sx={{ color: 'warning.main' }}>修改 {stats.changed}</Box>
          <Box sx={{ color: 'error.main' }}>删除 {stats.removed}</Box>
          <Box sx={{ color: 'text.tertiary' }}>未变更 {stats.unchanged}</Box>
        </Stack>
      )}
      <Table
        columns={tab === 'docs' ? docsColumns : diffColumns}
        dataSource={tab === 'docs' ? detail?.docs || [] : detail?.diff || []}
        rowKey='node_id'
        size='small'
        height='430px'
        renderEmpty={
          loading ? (
            <Box></Box>
          ) : (
            <Box sx={{ py: 6, textAlign: 'center', color: 'text.tertiary' }}>
              暂无数据
            </Box>
          )
        }
        sx={{
          overflow: 'hidden',
          '.MuiTableContainer-root': {
            height: '430px',
          },
        }}
      />
    </Modal>
  );
};

export default VersionInspect;
