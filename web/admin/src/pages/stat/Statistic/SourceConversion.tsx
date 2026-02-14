import Nodata from '@/assets/images/nodata.png';
import Card from '@/components/Card';
import { getApiV1StatFunnel } from '@/request/Stat';
import { V1StatSourceItem } from '@/request/types';
import { useAppSelector } from '@/store';
import { Box, Stack } from '@mui/material';
import { useEffect, useState } from 'react';
import { ActiveTab, TimeList } from '.';

const SourceConversion = ({ tab }: { tab: ActiveTab }) => {
  const { kb_id = '' } = useAppSelector(state => state.config);
  const [list, setList] = useState<V1StatSourceItem[]>([]);
  const [max, setMax] = useState(0);

  useEffect(() => {
    if (!kb_id) return;
    getApiV1StatFunnel({ kb_id, day: tab }).then(res => {
      const data = (res?.sources || [])
        .slice()
        .sort((a, b) => {
          if ((b.conversion_rate || 0) === (a.conversion_rate || 0)) {
            return (b.visits || 0) - (a.visits || 0);
          }
          return (b.conversion_rate || 0) - (a.conversion_rate || 0);
        })
        .slice(0, 7);
      setList(data);
      setMax(Math.max(...data.map(item => item.visits || 0), 0));
    });
  }, [kb_id, tab]);

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
        <Box sx={{ fontSize: 16, fontWeight: 'bold' }}>来源转化排行</Box>
        <Box sx={{ fontSize: 12, color: 'text.tertiary' }}>
          {TimeList.find(it => it.value === tab)?.label}
        </Box>
      </Stack>
      {list.length > 0 ? (
        <Stack gap={1.5}>
          {list.map(it => (
            <Box key={it.referer_host}>
              <Stack
                direction={'row'}
                alignItems={'center'}
                justifyContent={'space-between'}
                sx={{ fontSize: 12 }}
              >
                <Stack direction={'row'} alignItems={'center'} gap={1}>
                  <Box>{it.referer_host || '-'}</Box>
                  {it.estimated ? (
                    <Box
                      sx={{
                        px: '6px',
                        py: '1px',
                        borderRadius: '10px',
                        fontSize: 10,
                        color: '#BC8A00',
                        bgcolor: '#FFF7E0',
                      }}
                    >
                      估算
                    </Box>
                  ) : null}
                </Stack>
                <Stack direction={'row'} alignItems={'center'} gap={1.5}>
                  <Box sx={{ color: 'text.tertiary' }}>{it.visits || 0} 次</Box>
                  <Box sx={{ fontWeight: 700 }}>
                    {((it.conversion_rate || 0) * 100).toFixed(2)}%
                  </Box>
                </Stack>
              </Stack>
              <Box
                sx={{
                  height: 6,
                  mt: '6px',
                  borderRadius: '3px',
                  bgcolor: 'background.paper3',
                }}
              >
                <Box
                  sx={{
                    height: 6,
                    borderRadius: '3px',
                    background:
                      'linear-gradient( 90deg, #36A6F8 0%, #2C62F6 100%)',
                    width: `${max > 0 ? ((it.visits || 0) / max) * 100 : 0}%`,
                  }}
                />
              </Box>
            </Box>
          ))}
        </Stack>
      ) : (
        <Stack
          alignItems={'center'}
          justifyContent={'center'}
          sx={{ fontSize: 12, color: 'text.disabled' }}
        >
          <img src={Nodata} width={100} />
          暂无数据
        </Stack>
      )}
    </Card>
  );
};

export default SourceConversion;
