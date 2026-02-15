import { putApiV1App } from '@/request/App';
import { DomainAppDetailResp, DomainNodeMetaSettings } from '@/request/types';
import { useAppSelector } from '@/store';
import { message } from '@ctzhian/ui';
import { FormControlLabel, Switch } from '@mui/material';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { FormItem, SettingCardItem } from './Common';

interface CardNodeMetaProps {
  id: string;
  data: DomainAppDetailResp;
  refresh: (value: DomainNodeMetaSettings) => void;
}

interface NodeMetaFormData {
  show_created_at: boolean;
  show_updated_at: boolean;
  show_word_count: boolean;
}

const CardNodeMeta = ({ id, data, refresh }: CardNodeMetaProps) => {
  const [isEdit, setIsEdit] = useState(false);
  const { kb_id } = useAppSelector(state => state.config);
  const { control, handleSubmit, setValue } = useForm<NodeMetaFormData>({
    defaultValues: {
      show_created_at: true,
      show_updated_at: true,
      show_word_count: true,
    },
  });

  const onSubmit = handleSubmit(value => {
    const submitValue: DomainNodeMetaSettings = {
      show_created_at: value.show_created_at,
      show_updated_at: value.show_updated_at,
      show_word_count: value.show_word_count,
    };
    putApiV1App(
      { id },
      {
        kb_id,
        settings: {
          ...data.settings,
          node_meta_settings: submitValue,
        },
      },
    ).then(() => {
      message.success('保存成功');
      refresh(submitValue);
      setIsEdit(false);
    });
  });

  useEffect(() => {
    const metaSettings = data.settings?.node_meta_settings;
    setValue('show_created_at', metaSettings?.show_created_at ?? true);
    setValue('show_updated_at', metaSettings?.show_updated_at ?? true);
    setValue('show_word_count', metaSettings?.show_word_count ?? true);
  }, [data, setValue]);

  return (
    <SettingCardItem title='文档元信息显示' isEdit={isEdit} onSubmit={onSubmit}>
      <FormItem label='显示创建时间'>
        <Controller
          control={control}
          name='show_created_at'
          render={({ field }) => (
            <FormControlLabel
              control={
                <Switch
                  size='small'
                  checked={!!field.value}
                  onChange={(_, checked) => {
                    field.onChange(checked);
                    setIsEdit(true);
                  }}
                />
              }
              label={field.value ? '开启' : '关闭'}
            />
          )}
        />
      </FormItem>

      <FormItem label='显示更新时间'>
        <Controller
          control={control}
          name='show_updated_at'
          render={({ field }) => (
            <FormControlLabel
              control={
                <Switch
                  size='small'
                  checked={!!field.value}
                  onChange={(_, checked) => {
                    field.onChange(checked);
                    setIsEdit(true);
                  }}
                />
              }
              label={field.value ? '开启' : '关闭'}
            />
          )}
        />
      </FormItem>

      <FormItem label='显示字数'>
        <Controller
          control={control}
          name='show_word_count'
          render={({ field }) => (
            <FormControlLabel
              control={
                <Switch
                  size='small'
                  checked={!!field.value}
                  onChange={(_, checked) => {
                    field.onChange(checked);
                    setIsEdit(true);
                  }}
                />
              }
              label={field.value ? '开启' : '关闭'}
            />
          )}
        />
      </FormItem>
    </SettingCardItem>
  );
};

export default CardNodeMeta;
