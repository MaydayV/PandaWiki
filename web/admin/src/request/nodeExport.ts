import httpRequest from './httpClient';

export type NodeExportReq = {
  id: string;
  kb_id: string;
  format: 'md' | 'html';
};

export type NodeExportResp = {
  content: string;
  filename: string;
};

export const getApiV1NodeExport = (query: NodeExportReq) =>
  httpRequest<NodeExportResp>({
    path: '/api/v1/node/export',
    method: 'GET',
    query,
  });
