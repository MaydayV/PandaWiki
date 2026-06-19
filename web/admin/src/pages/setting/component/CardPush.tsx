import {
  Box,
  FormControlLabel,
  Radio,
  RadioGroup,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { FormItem, SettingCardItem } from './Common';
import { DomainAppDetailResp } from '@/request/types';
import { getApiV1AppDetail, putApiV1App } from '@/request/App';
import { useAppSelector } from '@/store';

const defaultTemplate = `📚 知识库「{kb_name}」已更新
版本：{tag} | 发布说明：{message}
发布时间：{release_time}`;

const CardPush = () => {
  const [isEdit, setIsEdit] = useState(false);
  const [detail, setDetail] = useState<DomainAppDetailResp | null>(null);
  const [isEnabled, setIsEnabled] = useState(false);
  const { kb_id } = useAppSelector(state => state.config);
  const { control, handleSubmit, reset } = useForm({
    defaultValues: {
      kb_update_push_enabled: false,
      kb_update_push_chat_ids: '',
      kb_update_push_content: '',
    },
  });

  const getDetail = () => {
    if (!kb_id) return;
    getApiV1AppDetail({ kb_id, type: '1' }).then(res => {
      setDetail(res);
      // @ts-expect-error 新增字段，swagger 类型待更新
      setIsEnabled(res.settings?.kb_update_push_enabled ?? false);
      reset({
        // @ts-expect-error 新增字段，swagger 类型待更新
        kb_update_push_enabled: res.settings?.kb_update_push_enabled ?? false,
        // @ts-expect-error 新增字段，swagger 类型待更新
        kb_update_push_chat_ids: res.settings?.kb_update_push_chat_ids ?? '',
        // @ts-expect-error 新增字段，swagger 类型待更新
        kb_update_push_content: res.settings?.kb_update_push_content ?? '',
      });
    });
  };

  const onSubmit = handleSubmit(data => {
    if (!detail) return;
    putApiV1App(
      { id: detail.id! },
      {
        kb_id,
        settings: {
          // @ts-expect-error 新增字段，swagger 类型待更新
          kb_update_push_enabled: data.kb_update_push_enabled,
          kb_update_push_chat_ids: data.kb_update_push_chat_ids,
          kb_update_push_content: data.kb_update_push_content,
        },
      },
    ).then(() => {
      message.success('保存成功');
      setIsEdit(false);
      getDetail();
      reset();
    });
  });

  useEffect(() => {
    getDetail();
  }, [kb_id]);

  return (
    <SettingCardItem title='知识库更新推送' isEdit={isEdit} onSubmit={onSubmit}>
      <FormItem label='知识库更新推送'>
        <Controller
          control={control}
          name='kb_update_push_enabled'
          render={({ field }) => (
            <RadioGroup
              row
              {...field}
              onChange={e => {
                field.onChange(e.target.value === 'true');
                setIsEnabled(e.target.value === 'true');
                setIsEdit(true);
              }}
            >
              <FormControlLabel
                value={true}
                control={<Radio size='small' />}
                label={<Box sx={{ width: 100 }}>启用</Box>}
              />
              <FormControlLabel
                value={false}
                control={<Radio size='small' />}
                label={<Box sx={{ width: 100 }}>禁用</Box>}
              />
            </RadioGroup>
          )}
        />
      </FormItem>

      {isEnabled && (
        <>
          <FormItem label='推送目标群聊' required>
            <Controller
              control={control}
              name='kb_update_push_chat_ids'
              render={({ field }) => (
                <TextField
                  {...field}
                  fullWidth
                  placeholder='飞书或钉钉群聊 Webhook URL，多个用逗号分隔'
                  onChange={e => {
                    field.onChange(e.target.value);
                    setIsEdit(true);
                  }}
                />
              )}
            />
          </FormItem>

          <FormItem label='推送消息模板'>
            <Controller
              control={control}
              name='kb_update_push_content'
              render={({ field }) => (
                <TextField
                  {...field}
                  fullWidth
                  multiline
                  rows={4}
                  placeholder={defaultTemplate}
                  onChange={e => {
                    field.onChange(e.target.value);
                    setIsEdit(true);
                  }}
                />
              )}
            />
            <Typography
              variant='caption'
              color='text.secondary'
              sx={{ mt: 0.5 }}
            >
              支持变量：{'{kb_name}'} {'{tag}'} {'{message}'} {'{release_time}'}
              。留空使用默认模板。
            </Typography>
          </FormItem>
        </>
      )}
    </SettingCardItem>
  );
};

export default CardPush;
