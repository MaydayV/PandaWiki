import Card from '@/components/Card';
import { getApiV1StatFunnel } from '@/request/Stat';
import { V1StatFunnelData } from '@/request/types';
import { useAppSelector } from '@/store';
import { Box, Stack } from '@mui/material';
import { useEffect, useMemo, useState } from 'react';
import { ActiveTab, TimeList } from '.';

const FunnelConversion = ({ tab }: { tab: ActiveTab }) => {
  const { kb_id = '' } = useAppSelector(state => state.config);
  const [funnel, setFunnel] = useState<V1StatFunnelData | null>(null);

  useEffect(() => {
    if (!kb_id) return;
    getApiV1StatFunnel({ kb_id, day: tab }).then(res => {
      setFunnel(res?.funnel || null);
    });
  }, [kb_id, tab]);

  const visits = funnel?.visits || funnel?.page_visit_count || 0;
  const sessions = funnel?.sessions || 0;
  const conversations = funnel?.conversations || 0;
  const conversionRate = funnel?.conversion_rate || 0;

  const rows = useMemo(
    () => [
      {
        label: '访问次数',
        value: visits,
        ratio: 1,
      },
      {
        label: '访问用户数',
        value: sessions,
        ratio: visits > 0 ? sessions / visits : 0,
      },
      {
        label: '问答次数',
        value: conversations,
        ratio: sessions > 0 ? conversations / sessions : 0,
      },
    ],
    [conversations, sessions, visits],
  );

  return (
    <Card
      sx={{
        p: 2,
        height: '100%',
        boxShadow: '0px 4px 8px rgba(0, 0, 0, 0.1)',
      }}
    >
      <Stack
        direction={'row'}
        alignItems={'center'}
        justifyContent={'space-between'}
        sx={{ mb: 2 }}
      >
        <Box sx={{ fontSize: 16, fontWeight: 'bold' }}>漏斗概览</Box>
        <Box sx={{ fontSize: 12, color: 'text.tertiary' }}>
          {TimeList.find(it => it.value === tab)?.label}
        </Box>
      </Stack>
      <Stack gap={1.5}>
        {rows.map((item, idx) => (
          <Box key={item.label}>
            <Stack
              direction={'row'}
              alignItems={'center'}
              justifyContent={'space-between'}
              sx={{ fontSize: 12, mb: 0.5 }}
            >
              <Box>{item.label}</Box>
              <Box sx={{ fontWeight: 700 }}>{item.value}</Box>
            </Stack>
            <Box
              sx={{
                height: 8,
                borderRadius: '4px',
                bgcolor: 'background.paper3',
              }}
            >
              <Box
                sx={{
                  height: 8,
                  borderRadius: '4px',
                  width: `${Math.min(Math.max(item.ratio * 100, 2), 100)}%`,
                  background:
                    idx === 2
                      ? 'linear-gradient( 90deg, #2C62F6 0%, #50A3FF 100%)'
                      : 'linear-gradient( 90deg, #3248F2 0%, #9E68FC 100%)',
                }}
              />
            </Box>
          </Box>
        ))}
      </Stack>
      <Card
        sx={{
          mt: 2,
          p: 1.5,
          bgcolor: 'background.paper3',
        }}
      >
        <Stack direction={'row'} alignItems={'center'} justifyContent={'space-between'}>
          <Box sx={{ fontSize: 12, color: 'text.tertiary' }}>整体转化率</Box>
          <Box sx={{ fontSize: 18, fontWeight: 700 }}>
            {(conversionRate * 100).toFixed(2)}%
          </Box>
        </Stack>
      </Card>
    </Card>
  );
};

export default FunnelConversion;
