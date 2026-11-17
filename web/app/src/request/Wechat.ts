 
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
  DomainResponse,
  GetShareV1AppWechatServiceAnswerParams,
} from "./types";

/**
 * @description GetWechatAnswer
 *
 * @tags Wechat
 * @name GetShareV1AppWechatServiceAnswer
 * @summary GetWechatAnswer
 * @request GET:/share/v1/app/wechat/service/answer
 * @response `200` `DomainResponse` OK
 */

export const getShareV1AppWechatServiceAnswer = (
  query: GetShareV1AppWechatServiceAnswerParams,
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse>({
    path: `/share/v1/app/wechat/service/answer`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
