 
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

import httpRequest, { ContentType, RequestParams } from "./httpClient";
import {
  DomainChatSearchReq,
  DomainChatSearchResp,
  DomainResponse,
} from "./types";

/**
 * @description ChatSearch
 *
 * @tags share_chat_search
 * @name PostShareV1ChatSearch
 * @summary ChatSearch
 * @request POST:/share/v1/chat/search
 * @response `200` `(DomainResponse & {
    data?: DomainChatSearchResp,

})` OK
 */

export const postShareV1ChatSearch = (
  request: DomainChatSearchReq,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainResponse & {
      data?: DomainChatSearchResp;
    }
  >({
    path: `/share/v1/chat/search`,
    method: "POST",
    body: request,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
