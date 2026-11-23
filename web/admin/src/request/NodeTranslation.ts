import httpRequest, { ContentType, RequestParams } from './httpClient';
import { DomainPWResponse } from './types';

export interface NodeAutoTranslateReq {
  kb_id: string;
  node_id: string;
  source_language?: string;
  target_languages: string[];
  publish_to_latest_release?: boolean;
  overwrite?: boolean;
}

export interface NodeAutoTranslateResp {
  node_id: string;
  translated_languages: string[];
  skipped_languages: string[];
  available_languages: string[];
  published: boolean;
}

export const postApiProV1NodeTranslationAuto = (
  body: NodeAutoTranslateReq,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: NodeAutoTranslateResp;
    }
  >({
    path: '/api/pro/v1/node/translation/auto',
    method: 'POST',
    body,
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
