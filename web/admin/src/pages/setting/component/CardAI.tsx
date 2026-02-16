import { getApiProV1Prompt, postApiProV1Prompt } from '@/request/pro/Prompt';
import { getApiV1AppDetail, putApiV1App } from '@/request/App';
import { DomainAppDetailResp, DomainKnowledgeBaseDetail } from '@/request/types';
import { ALL_VERSION_PERMISSION } from '@/constant/version';
import { useAppSelector } from '@/store';
import { message, Modal } from '@ctzhian/ui';
import { Box, Slider, TextField } from '@mui/material';
import { useEffect, useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { FormItem, SettingCardItem } from './Common';
import { DomainCreatePromptReq } from '@/request/pro/types';

interface CardAIProps {
  kb: DomainKnowledgeBaseDetail;
}

const CardAI = ({ kb }: CardAIProps) => {
  const [isEdit, setIsEdit] = useState(false);
  const { license } = useAppSelector(state => state.config);
  const [webApp, setWebApp] = useState<DomainAppDetailResp>();

  const { control, handleSubmit, setValue, getValues } = useForm({
    defaultValues: {
      interval: 0,
      content: '',
      summary_content: '',
    },
  });

  const onSubmit = handleSubmit(async data => {
    await Promise.all([
      postApiProV1Prompt({
        kb_id: kb.id!,
        content: data.content,
        summary_content: data.summary_content,
      }),
      webApp?.id
        ? putApiV1App(
            { id: webApp.id },
            {
              kb_id: kb.id!,
              settings: {
                ...webApp.settings,
                conversation_setting: {
                  ...webApp.settings?.conversation_setting,
                  ask_interval_seconds: data.interval,
                },
              },
            },
          )
        : Promise.resolve(),
    ]);

    message.success('保存成功');
    setIsEdit(false);
    setWebApp(prev => {
      if (!prev) return prev;
      return {
        ...prev,
        settings: {
          ...prev.settings,
          conversation_setting: {
            ...prev.settings?.conversation_setting,
            ask_interval_seconds: data.interval,
          },
        },
      };
    });
  });

  const canEditPrompt = useMemo(() => {
    return ALL_VERSION_PERMISSION.includes(license.edition!);
  }, [license]);

  useEffect(() => {
    if (!kb.id || !ALL_VERSION_PERMISSION.includes(license.edition!))
      return;
    Promise.all([
      getApiProV1Prompt({ kb_id: kb.id! }),
      getApiV1AppDetail({ kb_id: kb.id!, type: '1' }),
    ]).then(([promptRes, appRes]) => {
      setValue('content', promptRes.content || '');
      setValue('summary_content', promptRes.summary_content || '');
      setValue(
        'interval',
        Math.max(
          0,
          Math.min(
            300,
            appRes.settings?.conversation_setting?.ask_interval_seconds ?? 0,
          ),
        ),
      );
      setWebApp(appRes);
    });
  }, [kb, canEditPrompt]);

  const onResetPrompt = (type: 'content' | 'summary_content' = 'content') => {
    Modal.confirm({
      title: '提示',
      content: `确定要重置为默认${type === 'content' ? '智能问答' : '智能摘要'}提示词吗？`,
      onOk: () => {
        let params: DomainCreatePromptReq = {
          kb_id: kb.id!,
          content: '',
          summary_content: getValues('summary_content'),
        };
        if (type === 'summary_content') {
          params = {
            kb_id: kb.id!,
            summary_content: '',
            content: getValues('content'),
          };
        }
        postApiProV1Prompt(params).then(() => {
          getApiProV1Prompt({ kb_id: kb.id! }).then(res => {
            setValue(type, res[type] || '');
          });
        });
      },
    });
  };

  return (
    <Box
      sx={{
        width: 1000,
        margin: 'auto',
        pb: 4,
      }}
    >
      <SettingCardItem title='智能问答' isEdit={isEdit} onSubmit={onSubmit}>
        <FormItem
          vertical
          permission={ALL_VERSION_PERMISSION}
          extra={
            <Box
              sx={{
                fontSize: 12,
                color: 'primary.main',
                display: 'block',
                cursor: 'pointer',
              }}
              onClick={() => onResetPrompt('content')}
            >
              重置为默认提示词
            </Box>
          }
          label='智能问答提示词'
        >
          <Controller
            control={control}
            name='content'
            render={({ field }) => (
              <TextField
                {...field}
                fullWidth
                disabled={!canEditPrompt}
                multiline
                rows={20}
                placeholder='智能问答提示词'
                onChange={e => {
                  field.onChange(e.target.value);
                  setIsEdit(true);
                }}
              />
            )}
          />
        </FormItem>
        <FormItem vertical label='连续提问时间间隔（秒）'>
          <Controller
            control={control}
            name='interval'
            render={({ field }) => (
              <Slider
                {...field}
                valueLabelDisplay='auto'
                valueLabelFormat={value => (value === 0 ? '关闭' : `${value}s`)}
                min={0}
                max={300}
                step={1}
                sx={{
                  width: 432,
                  '& .MuiSlider-thumb': {
                    width: 16,
                    height: 16,
                    borderRadius: '50%',
                    backgroundColor: '#fff',
                    border: '2px solid currentColor',
                    '&:focus, &:hover, &.Mui-active, &.Mui-focusVisible': {
                      boxShadow: 'inherit',
                    },
                    '&::before': {
                      display: 'none',
                    },
                  },
                  '& .MuiSlider-track': {
                    bgcolor: 'primary.main',
                  },
                  '& .MuiSlider-rail': {
                    bgcolor: 'text.disabled',
                  },
                  '& .MuiSlider-valueLabel': {
                    lineHeight: 1.2,
                    fontSize: 12,
                    fontWeight: 'bold',
                    background: 'unset',
                    p: 0,
                    width: 24,
                    height: 24,
                    borderRadius: '50% 50% 50% 0',
                    bgcolor: 'primary.main',
                    transformOrigin: 'bottom left',
                    transform: 'translate(50%, -100%) rotate(-45deg) scale(0)',
                    '&::before': { display: 'none' },
                    '&.MuiSlider-valueLabelOpen': {
                      transform:
                        'translate(50%, -100%) rotate(-45deg) scale(1)',
                    },
                    '& > *': {
                      transform: 'rotate(45deg)',
                    },
                  },
                }}
                onChange={(e, value) => {
                  const nextValue = Array.isArray(value) ? value[0] : value;
                  field.onChange(nextValue);
                  setIsEdit(true);
                }}
              />
            )}
          />
          <Box sx={{ fontSize: 12, color: 'text.secondary', mt: 1 }}>
            0 表示不限制。开启后，同一来源需等待设置秒数后才能继续提问。
          </Box>
        </FormItem>
        <FormItem
          vertical
          permission={ALL_VERSION_PERMISSION}
          extra={
            <Box
              sx={{
                fontSize: 12,
                color: 'primary.main',
                display: 'block',
                cursor: 'pointer',
              }}
              onClick={() => onResetPrompt('summary_content')}
            >
              重置为默认提示词
            </Box>
          }
          label='智能摘要提示词'
        >
          <Controller
            control={control}
            name='summary_content'
            render={({ field }) => (
              <TextField
                {...field}
                fullWidth
                disabled={!canEditPrompt}
                multiline
                rows={5}
                placeholder='智能摘要提示词'
                onChange={e => {
                  field.onChange(e.target.value);
                  setIsEdit(true);
                }}
              />
            )}
          />
        </FormItem>
      </SettingCardItem>
    </Box>
  );
};

export default CardAI;
