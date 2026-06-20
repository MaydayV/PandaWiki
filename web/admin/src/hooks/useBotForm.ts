import { useCallback, useEffect, useState } from 'react';
import {
  useForm,
  type UseFormReturn,
  type DefaultValues,
} from 'react-hook-form';
import { message } from '@ctzhian/ui';
import { DomainAppDetailResp } from '@/request/types';
import { getApiV1AppDetail, putApiV1App } from '@/request/App';
import { useAppSelector } from '@/store';

interface UseBotFormOptions<T extends Record<string, unknown>> {
  appType: string;
  defaultValues: DefaultValues<T>;
  /** Transform API response settings to form values */
  mapSettingsToForm: (settings: Record<string, unknown>) => Partial<T>;
  /** Transform form values to API request settings */
  mapFormToSettings: (data: T) => Record<string, unknown>;
}

interface UseBotFormReturn<T extends Record<string, unknown>> {
  isEdit: boolean;
  setIsEdit: (v: boolean) => void;
  detail: DomainAppDetailResp | null;
  control: UseFormReturn<T>['control'];
  handleSubmit: UseFormReturn<T>['handleSubmit'];
  reset: UseFormReturn<T>['reset'];
  errors: UseFormReturn<T>['formState']['errors'];
  getDetail: () => void;
  onSubmit: () => void;
  handleFieldChange: (callback?: () => void) => void;
}

export function useBotForm<T extends Record<string, unknown>>(
  options: UseBotFormOptions<T>,
): UseBotFormReturn<T> {
  const { appType, defaultValues, mapSettingsToForm, mapFormToSettings } =
    options;
  const [isEdit, setIsEdit] = useState(false);
  const [detail, setDetail] = useState<DomainAppDetailResp | null>(null);
  const { kb_id } = useAppSelector(state => state.config);

  const {
    control,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<T>({ defaultValues });

  const getDetail = useCallback(() => {
    if (!kb_id) return;
    getApiV1AppDetail({ kb_id, type: appType }).then(res => {
      setDetail(res);
      const formData = mapSettingsToForm(
        (res.settings as Record<string, unknown>) ?? {},
      );
      reset(formData as DefaultValues<T>);
    });
  }, [kb_id, appType, reset, mapSettingsToForm]);

  const onSubmit = handleSubmit(data => {
    if (!detail) return;
    putApiV1App(
      { id: detail.id! },
      {
        kb_id,
        settings: mapFormToSettings(data),
      },
    ).then(() => {
      message.success('保存成功');
      setIsEdit(false);
      getDetail();
      reset();
    });
  });

  const handleFieldChange = useCallback((callback?: () => void) => {
    setIsEdit(true);
    callback?.();
  }, []);

  useEffect(() => {
    getDetail();
  }, [getDetail]);

  return {
    isEdit,
    setIsEdit,
    detail,
    control,
    handleSubmit,
    reset,
    errors,
    getDetail,
    onSubmit,
    handleFieldChange,
  };
}
