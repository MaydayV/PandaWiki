import { message } from '@ctzhian/ui';
import { Box, FormControlLabel, Radio, RadioGroup } from '@mui/material';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { DomainAppDetailResp } from '@/request/types';
import { SettingCardItem, FormItem } from './Common';
import { useAppSelector } from '@/store';
import { putApiV1App } from '@/request/App';

interface CardWebStatsProps {
  id: string;
  data: DomainAppDetailResp;
  refresh: (value: { pv_enable?: boolean }) => void;
}

interface StatsFormData {
  pv_enable: 1 | 2;
}

const CardWebStats = ({ data, id, refresh }: CardWebStatsProps) => {
  const [isEdit, setIsEdit] = useState(false);
  const { kb_id } = useAppSelector(state => state.config);
  const { handleSubmit, control, setValue } = useForm<StatsFormData>({
    defaultValues: {
      pv_enable: 1,
    },
  });

  const onSubmit = handleSubmit((value: StatsFormData) => {
    const submitValue = {
      pv_enable: value.pv_enable === 1,
    };
    putApiV1App(
      { id },
      { kb_id, settings: { ...data.settings, stats_setting: submitValue } },
    ).then(() => {
      message.success('保存成功');
      refresh(submitValue);
      setIsEdit(false);
    });
  });

  useEffect(() => {
    const pvEnable = data.settings?.stats_setting?.pv_enable;
    setValue('pv_enable', pvEnable === false ? 2 : 1);
  }, [data]);

  return (
    <SettingCardItem title='统计分析' isEdit={isEdit} onSubmit={onSubmit}>
      <FormItem label='文档浏览量'>
        <Controller
          control={control}
          name='pv_enable'
          render={({ field }) => (
            <RadioGroup
              row
              {...field}
              onChange={e => {
                field.onChange(+e.target.value as 1 | 2);
                setIsEdit(true);
              }}
            >
              <FormControlLabel
                value={1}
                control={<Radio size='small' />}
                label={<Box sx={{ width: 100 }}>展示</Box>}
              />
              <FormControlLabel
                value={2}
                control={<Radio size='small' />}
                label={<Box sx={{ width: 100 }}>隐藏</Box>}
              />
            </RadioGroup>
          )}
        />
      </FormItem>
    </SettingCardItem>
  );
};

export default CardWebStats;
