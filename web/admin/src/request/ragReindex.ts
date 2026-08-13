import httpRequest, { ContentType } from './httpClient';

export type ReindexRAGReq = {
  kb_id: string;
  recreate_dataset?: boolean;
};

export type ReindexRAGResp = {
  queued: number;
};

export const postApiV1KnowledgeBaseRagReindex = (
  body: ReindexRAGReq,
) =>
  httpRequest<ReindexRAGResp>({
    path: '/api/v1/knowledge_base/rag/reindex',
    method: 'POST',
    body,
    type: ContentType.Json,
  });
