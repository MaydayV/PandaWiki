// GitHub MD 同步配置卡片（git 为唯一内容真源的同步模式）
import { useState, useEffect, useCallback } from 'react';
import {
  Button,
  Divider,
  FormControlLabel,
  InputAdornment,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';

import {
  getApiV1GithubSyncConfig,
  putApiV1GithubSyncConfig,
  postApiV1GithubSyncSync,
  type DomainGitHubSyncSetting,
} from '@/request/GitHubSync';
import { useAppSelector } from '@/store';

const defaultConfig = (kbId: string): DomainGitHubSyncSetting => ({
  kb_id: kbId,
  enabled: false,
  repo: '',
  branch: 'main',
  token: '',
  interval_minutes: 0,
  delete_orphans: false,
  auto_publish: true,
});

const CardGitHubSync = () => {
  const kb_id = useAppSelector(state => state.config.kb_id);
  const [config, setConfig] = useState<DomainGitHubSyncSetting>(defaultConfig(kb_id));
  const [hasToken, setHasToken] = useState(false);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const loadConfig = useCallback(async () => {
    if (!kb_id) return;
    try {
      const res = await getApiV1GithubSyncConfig({ kb_id });
      setConfig({ ...(res || defaultConfig(kb_id)) });
      setHasToken(Boolean((res as { has_token?: boolean })?.has_token));
    } catch {
      // 全局拦截器已提示
    }
  }, [kb_id]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const update = (patch: Partial<DomainGitHubSyncSetting>) =>
    setConfig((prev: DomainGitHubSyncSetting) => ({ ...prev, ...patch }));

  const handleSave = async () => {
    if (!config.repo || config.repo.split('/').length !== 2) {
      message.error('仓库格式必须为 owner/repo');
      return;
    }
    setSaving(true);
    try {
      await putApiV1GithubSyncConfig({ kb_id }, config);
      message.success('保存成功');
      loadConfig();
    } catch {
      // 全局拦截器已提示
    } finally {
      setSaving(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      const res = await postApiV1GithubSyncSync({ kb_id });
      const r = res as {
        created?: number;
        updated?: number;
        skipped?: number;
        deleted?: number;
        failed?: number;
        duration?: string;
      };
      message.success(
        `同步完成：新建 ${r.created ?? 0}，更新 ${r.updated ?? 0}，跳过 ${r.skipped ?? 0}，删除 ${r.deleted ?? 0}，失败 ${r.failed ?? 0}（${r.duration ?? ''}）`,
      );
      loadConfig();
    } catch {
      // 全局拦截器已提示
    } finally {
      setSyncing(false);
    }
  };

  return (
    <Stack spacing={2}>
      <Typography variant='h6'>GitHub 文章同步</Typography>
      <Typography variant='body2' color='text.secondary'>
        将 GitHub 私有仓库中的 Markdown 文章同步到当前知识库：目录结构自动映射为栏目，图片自动转存，变更自动发布并更新 AI 向量。开启后建议以仓库为唯一编辑入口，避免后台编辑与仓库冲突。
      </Typography>
      <Divider />

      <FormControlLabel
        control={
          <Switch
            checked={config.enabled}
            onChange={e => update({ enabled: e.target.checked })}
          />
        }
        label='启用同步'
      />

      <TextField
        label='仓库（owner/repo）'
        placeholder='my-name/my-wiki-content'
        size='small'
        value={config.repo}
        onChange={e => update({ repo: e.target.value })}
      />
      <TextField
        label='分支'
        size='small'
        value={config.branch}
        onChange={e => update({ branch: e.target.value })}
      />
      <TextField
        label={hasToken ? '访问令牌（已保存，留空保持不变）' : '访问令牌（GitHub PAT，需仓库读取权限）'}
        type='password'
        size='small'
        value={config.token}
        onChange={e => update({ token: e.target.value })}
        placeholder={hasToken ? '••••••••' : 'ghp_xxx / github_pat_xxx'}
      />
      <TextField
        label='自动同步间隔（分钟，0 = 仅手动）'
        size='small'
        type='number'
        value={config.interval_minutes}
        onChange={e => update({ interval_minutes: Number(e.target.value) })}
        InputProps={{ inputProps: { min: 0, max: 1440 } }}
      />
      <FormControlLabel
        control={
          <Switch
            checked={config.auto_publish}
            onChange={e => update({ auto_publish: e.target.checked })}
          />
        }
        label='同步后自动发布（触发 AI 向量更新）'
      />
      <FormControlLabel
        control={
          <Switch
            checked={config.delete_orphans}
            onChange={e => update({ delete_orphans: e.target.checked })}
          />
        }
        label='删除仓库中已移除的文章（默认仅新增/更新）'
      />

      <Divider />
      <Typography variant='body2' color='text.secondary'>
        分类规则：文章 front-matter 的 <code>category</code> 字段优先指定栏目；否则以仓库一级目录名作为栏目（不存在时自动创建），更深层目录映射为文档分组。front-matter 还支持 <code>title</code>、<code>emoji</code>、<code>order</code> 字段。
      </Typography>

      {config.last_sync_result && (
        <Typography variant='body2' color='text.secondary'>
          最近同步：{config.last_sync_result}
          {config.last_sync_at ? `（${config.last_sync_at}）` : ''}
        </Typography>
      )}

      <Stack direction='row' spacing={2}>
        <Button variant='contained' loading={saving} onClick={handleSave}>
          保存配置
        </Button>
        <Button variant='outlined' loading={syncing} onClick={handleSync} disabled={!config.enabled}>
          立即同步
        </Button>
      </Stack>
    </Stack>
  );
};

export default CardGitHubSync;
