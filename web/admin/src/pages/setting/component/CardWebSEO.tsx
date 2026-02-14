import { SEOSetting } from '@/api';
import { TextField } from '@mui/material';
import { message } from '@ctzhian/ui';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { DomainAppDetailResp } from '@/request/types';
import { SettingCardItem, FormItem } from './Common';
import { useAppSelector } from '@/store';
import { putApiV1App } from '@/request/App';

interface CardWebSEOProps {
  id: string;
  data: DomainAppDetailResp;
  refresh: (value: SEOSetting) => void;
}

interface SEOFormValues extends SEOSetting {
  canonical_url: string;
  robots: string;
  og_image: string;
  twitter_card: string;
  json_ld: string;
}

const CardWebSEO = ({ data, id, refresh }: CardWebSEOProps) => {
  const [isEdit, setIsEdit] = useState(false);
  const { kb_id } = useAppSelector(state => state.config);
  const {
    handleSubmit,
    control,
    setValue,
    formState: { errors },
  } = useForm<SEOFormValues>({
    defaultValues: {
      desc: '',
      keyword: '',
      canonical_url: '',
      robots: '',
      og_image: '',
      twitter_card: '',
      json_ld: '',
    },
  });

  const onSubmit = handleSubmit((value: SEOFormValues) => {
    const { desc, keyword, ...advancedSEO } = value;
    const submitSEOSettings = {
      ...data.settings?.seo_settings,
      ...advancedSEO,
    };

    putApiV1App(
      { id },
      {
        kb_id,
        settings: {
          ...data.settings,
          desc,
          keyword,
          seo_settings: submitSEOSettings,
        },
      },
    ).then(() => {
      message.success('保存成功');
      refresh({ desc, keyword });
      setIsEdit(false);
    });
  });

  useEffect(() => {
    setValue('desc', data.settings?.desc || '');
    setValue('keyword', data.settings?.keyword || '');
    setValue('canonical_url', data.settings?.seo_settings?.canonical_url || '');
    setValue('robots', data.settings?.seo_settings?.robots || '');
    setValue('og_image', data.settings?.seo_settings?.og_image || '');
    setValue('twitter_card', data.settings?.seo_settings?.twitter_card || '');
    setValue('json_ld', data.settings?.seo_settings?.json_ld || '');
  }, [data]);

  return (
    <SettingCardItem title='SEO' isEdit={isEdit} onSubmit={onSubmit}>
      <FormItem label='网站描述'>
        <Controller
          control={control}
          name='desc'
          render={({ field }) => (
            <TextField
              fullWidth
              {...field}
              placeholder='请输入网站描述'
              error={!!errors.desc}
              helperText={errors.desc?.message}
              onChange={event => {
                setIsEdit(true);
                field.onChange(event);
              }}
            />
          )}
        />
      </FormItem>

      <FormItem label='关键词'>
        <Controller
          control={control}
          name='keyword'
          render={({ field }) => (
            <TextField
              fullWidth
              {...field}
              placeholder='请输入关键词，多个请用英文逗号分隔'
              error={!!errors.keyword}
              helperText={errors.keyword?.message}
              onChange={event => {
                setIsEdit(true);
                field.onChange(event);
              }}
            />
          )}
        />
      </FormItem>

      <FormItem label='Canonical URL'>
        <Controller
          control={control}
          name='canonical_url'
          render={({ field }) => (
            <TextField
              fullWidth
              {...field}
              placeholder='请输入规范链接（Canonical URL）'
              error={!!errors.canonical_url}
              helperText={errors.canonical_url?.message}
              onChange={event => {
                setIsEdit(true);
                field.onChange(event);
              }}
            />
          )}
        />
      </FormItem>

      <FormItem label='Robots'>
        <Controller
          control={control}
          name='robots'
          render={({ field }) => (
            <TextField
              fullWidth
              {...field}
              placeholder='例如：index,follow'
              error={!!errors.robots}
              helperText={errors.robots?.message}
              onChange={event => {
                setIsEdit(true);
                field.onChange(event);
              }}
            />
          )}
        />
      </FormItem>

      <FormItem label='OG 图片'>
        <Controller
          control={control}
          name='og_image'
          render={({ field }) => (
            <TextField
              fullWidth
              {...field}
              placeholder='请输入 Open Graph 分享图片地址'
              error={!!errors.og_image}
              helperText={errors.og_image?.message}
              onChange={event => {
                setIsEdit(true);
                field.onChange(event);
              }}
            />
          )}
        />
      </FormItem>

      <FormItem label='Twitter Card'>
        <Controller
          control={control}
          name='twitter_card'
          render={({ field }) => (
            <TextField
              fullWidth
              {...field}
              placeholder='例如：summary_large_image'
              error={!!errors.twitter_card}
              helperText={errors.twitter_card?.message}
              onChange={event => {
                setIsEdit(true);
                field.onChange(event);
              }}
            />
          )}
        />
      </FormItem>

      <FormItem label='JSON-LD' sx={{ alignItems: 'flex-start' }}>
        <Controller
          control={control}
          name='json_ld'
          render={({ field }) => (
            <TextField
              fullWidth
              multiline
              rows={4}
              {...field}
              placeholder='请输入 JSON-LD 结构化数据（JSON 格式）'
              error={!!errors.json_ld}
              helperText={errors.json_ld?.message}
              onChange={event => {
                setIsEdit(true);
                field.onChange(event);
              }}
            />
          )}
        />
      </FormItem>
    </SettingCardItem>
  );
};
export default CardWebSEO;
