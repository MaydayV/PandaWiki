// GitHub MD 同步配置接口（手写：cx-swagger-api 未重新生成前的过渡）
import request from './httpClient';

export interface DomainGitHubSyncSetting {
  kb_id: string;
  enabled: boolean;
  repo: string;
  branch: string;
  token?: string;
  has_token?: boolean;
  interval_minutes: number;
  delete_orphans: boolean;
  auto_publish: boolean;
  last_sync_at?: string;
  last_sync_result?: string;
}

export const getApiV1GithubSyncConfig = (query: { kb_id: string }) =>
  request<DomainGitHubSyncSetting>({
    path: '/api/v1/github_sync/config',
    method: 'GET',
    query,
  });

export const putApiV1GithubSyncConfig = (
  query: { kb_id: string },
  body: DomainGitHubSyncSetting,
) =>
  request<string>({
    path: '/api/v1/github_sync/config',
    method: 'PUT',
    query,
    body: { ...body, kb_id: query.kb_id || body.kb_id },
  });

export const postApiV1GithubSyncSync = (body: { kb_id: string }) =>
  request<{
    created: number;
    updated: number;
    skipped: number;
    deleted: number;
    failed: number;
    errors?: string[];
    duration: string;
  }>({
    path: '/api/v1/github_sync/sync',
    method: 'POST',
    body,
  });
