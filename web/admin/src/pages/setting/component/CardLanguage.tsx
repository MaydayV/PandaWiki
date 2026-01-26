import { putApiV1App } from '@/request/App';
import { DomainAppDetailResp } from '@/request/types';
import { useAppSelector } from '@/store';
import { message } from '@ctzhian/ui';
import { MenuItem, Select } from '@mui/material';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';

import { FormItem, SettingCardItem } from './Common';

const LANGUAGE_OPTIONS = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
  { value: 'auto', label: '跟随浏览器' },
];

const CardLanguage = ({
  data,
  refresh,
}: {
  data: DomainAppDetailResp;
  refresh: (value: { language: string }) => void;
}) => {
  const [isEdit, setIsEdit] = useState(false);
  const { kb_id } = useAppSelector(state => state.config);
  const { control, handleSubmit, setValue } = useForm({
    defaultValues: {
      language: 'en-US',
    },
  });

  const onSubmit = handleSubmit(value => {
    putApiV1App(
      { id: data.id! },
      { settings: { ...data.settings, language: value.language }, kb_id },
    ).then(() => {
      refresh({ language: value.language });
      message.success('保存成功');
      setIsEdit(false);
    });
  });

  useEffect(() => {
    const language = (data.settings as { language?: string })?.language;
    setValue('language', language || 'en-US');
  }, [data]);

  return (
    <SettingCardItem title='网站语言' isEdit={isEdit} onSubmit={onSubmit}>
      <FormItem label='界面语言'>
        <Controller
          control={control}
          name='language'
          render={({ field }) => (
            <Select
              fullWidth
              {...field}
              onChange={event => {
                field.onChange(event.target.value);
                setIsEdit(true);
              }}
            >
              {LANGUAGE_OPTIONS.map(option => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </Select>
          )}
        />
      </FormItem>
    </SettingCardItem>
  );
};

export default CardLanguage;
