/* eslint-disable */
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
  DomainCreatePromptReq,
  DomainPWResponse,
  DomainPrompt,
  GetApiProV1PromptParams,
  GetApiProV1PromptVersionDetailParams,
  GetApiProV1PromptVersionListParams,
  PostApiProV1PromptVersionRollbackParams,
} from "./types";

/**
 * @description Get all prompts
 *
 * @tags prompt
 * @name GetApiProV1Prompt
 * @summary Get all prompts
 * @request GET:/api/pro/v1/prompt
 * @response `200` `(DomainPWResponse & {
    data?: DomainPrompt,

})` OK
 */

export const getApiProV1Prompt = (
  query: GetApiProV1PromptParams,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: DomainPrompt;
    }
  >({
    path: `/api/pro/v1/prompt`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description Create a new prompt
 *
 * @tags prompt
 * @name PostApiProV1Prompt
 * @summary Create a new prompt
 * @request POST:/api/pro/v1/prompt
 * @response `200` `(DomainPWResponse & {
    data?: DomainPrompt,

})` OK
 */

export const postApiProV1Prompt = (
  req: DomainCreatePromptReq,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: DomainPrompt;
    }
  >({
    path: `/api/pro/v1/prompt`,
    method: "POST",
    body: req,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description Get prompt version list
 *
 * @tags prompt
 * @name GetApiProV1PromptVersionList
 * @summary Get prompt version list
 * @request GET:/api/pro/v1/prompt/version/list
 * @response `200` `DomainPWResponse` OK
 */
export const getApiProV1PromptVersionList = (
  query: GetApiProV1PromptVersionListParams,
  params: RequestParams = {},
) =>
  httpRequest<DomainPWResponse>({
    path: `/api/pro/v1/prompt/version/list`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description Get prompt version detail
 *
 * @tags prompt
 * @name GetApiProV1PromptVersionDetail
 * @summary Get prompt version detail
 * @request GET:/api/pro/v1/prompt/version/detail
 * @response `200` `DomainPWResponse` OK
 */
export const getApiProV1PromptVersionDetail = (
  query: GetApiProV1PromptVersionDetailParams,
  params: RequestParams = {},
) =>
  httpRequest<DomainPWResponse>({
    path: `/api/pro/v1/prompt/version/detail`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description Rollback prompt version
 *
 * @tags prompt
 * @name PostApiProV1PromptVersionRollback
 * @summary Rollback prompt version
 * @request POST:/api/pro/v1/prompt/version/rollback
 * @response `200` `DomainPWResponse` OK
 */
export const postApiProV1PromptVersionRollback = (
  req: PostApiProV1PromptVersionRollbackParams,
  params: RequestParams = {},
) =>
  httpRequest<DomainPWResponse>({
    path: `/api/pro/v1/prompt/version/rollback`,
    method: "POST",
    body: req,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
