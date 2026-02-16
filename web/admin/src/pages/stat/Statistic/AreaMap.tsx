import { TrendData } from '@/api';
import { getApiV1StatGeoCount } from '@/request/Stat';
import Nodata from '@/assets/images/nodata.png';
import Card from '@/components/Card';
import MapChart from '@/components/MapChart';
import { Countries } from '@/constant/area';
import { useAppSelector } from '@/store';
import { Box, Stack } from '@mui/material';
import { useEffect, useState } from 'react';
import { ActiveTab, TimeList } from '.';

const CountryNameCodeMap: Record<string, string> = Object.entries(Countries).reduce(
  (acc, [code, item]) => {
    acc[code] = code;
    acc[item.cn] = code;
    acc[item.en] = code;
    return acc;
  },
  {} as Record<string, string>,
);

const CountryAliasMap: Record<string, string> = {
  UK: 'GB',
  'U.K.': 'GB',
  USA: 'US',
  'U.S.A.': 'US',
};

const resolveCountryCode = (countryName: string) => {
  const normalized = countryName.trim();
  const upper = normalized.toUpperCase();
  return (
    CountryAliasMap[normalized] ||
    CountryAliasMap[upper] ||
    CountryNameCodeMap[normalized] ||
    CountryNameCodeMap[upper] ||
    ''
  );
};

const AreaMap = ({ tab }: { tab: ActiveTab }) => {
  const { kb_id } = useAppSelector(state => state.config);
  const [list, setList] = useState<TrendData[]>([]);
  const [mapList, setMapList] = useState<TrendData[]>([]);

  useEffect(() => {
    if (!kb_id) return;
    getApiV1StatGeoCount({ kb_id, day: tab }).then(res => {
      const displayCountMap = new Map<string, number>();
      const worldMapCount = new Map<string, number>();

      for (const [key, value] of Object.entries(res as Record<string, number>)) {
        const [country = ''] = key.split('|');
        const countryName = country.trim();
        const count = Number(value) || 0;
        if (!countryName || count <= 0 || countryName === '未知') continue;

        const countryCode = resolveCountryCode(countryName);
        const displayName = countryCode
          ? Countries[countryCode]?.cn || countryName
          : countryName;

        displayCountMap.set(
          displayName,
          (displayCountMap.get(displayName) || 0) + count,
        );

        if (countryCode) {
          worldMapCount.set(
            countryCode,
            (worldMapCount.get(countryCode) || 0) + count,
          );
        }
      }

      const toSortedList = (target: Map<string, number>) =>
        Array.from(target, ([name, count]) => ({ name, count })).sort(
          (a, b) => b.count - a.count,
        );

      setList(toSortedList(displayCountMap));
      setMapList(toSortedList(worldMapCount));
    });
  }, [kb_id, tab]);

  return (
    <Card
      sx={{
        flex: 1,
        bgcolor: 'background.paper3',
        position: 'relative',
      }}
    >
      <MapChart
        map='world'
        nameProperty='ISO_A2'
        data={mapList}
        tooltipText={'用户数量'}
        tooltipNameFormatter={name => Countries[name]?.cn || name}
      />
      <Box
        sx={{
          position: 'absolute',
          left: 16,
          top: 16,
          fontSize: 16,
          fontWeight: 'bold',
        }}
      >
        用户分布
      </Box>
      <Box
        sx={{
          position: 'absolute',
          top: 16,
          right: 232,
          fontSize: 12,
          width: 100,
          textAlign: 'right',
          color: 'text.tertiary',
        }}
      >
        {TimeList.find(item => item.value === tab)?.label || ''}
      </Box>
      <Card
        sx={{
          bgcolor: '#fff',
          p: 2,
          position: 'absolute',
          width: 200,
          height: 260,
          overflow: 'auto',
          right: 16,
          top: 16,
        }}
      >
        {list.length > 0 ? (
          <Stack gap={1.5}>
            {list.map(it => (
              <Stack
                direction='row'
                alignItems='center'
                justifyContent={'space-between'}
                gap={2}
                key={it.name}
                sx={{ fontSize: 12 }}
              >
                <Stack>{it.name}</Stack>
                <Box sx={{ fontWeight: 700 }}>{it.count}</Box>
              </Stack>
            ))}
          </Stack>
        ) : (
          <Stack
            alignItems={'center'}
            justifyContent={'center'}
            sx={{ height: '100%', fontSize: 12, color: 'text.disabled' }}
          >
            <img src={Nodata} width={100} />
            暂无数据
          </Stack>
        )}
      </Card>
    </Card>
  );
};

export default AreaMap;
