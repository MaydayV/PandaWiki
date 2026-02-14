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
  DomainGetNodeReleaseDetailResp,
  DomainNodeReleaseListItem,
  DomainPWResponse,
  GetApiProV1NodeReleaseDetailParams,
  GetApiProV1NodeReleaseListParams,
  PostApiProV1NodeReleaseRollbackParams,
} from "./types";

/**
 * @description Get Node Release Detail
 *
 * @tags node
 * @name GetApiProV1NodeReleaseDetail
 * @summary Get Node Release Detail
 * @request GET:/api/pro/v1/node/release/detail
 * @response `200` `(DomainPWResponse & {
    data?: DomainGetNodeReleaseDetailResp,

})` OK
 */

export const getApiProV1NodeReleaseDetail = (
  query: GetApiProV1NodeReleaseDetailParams,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: DomainGetNodeReleaseDetailResp;
    }
  >({
    path: `/api/pro/v1/node/release/detail`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description Get Node Release List
 *
 * @tags node
 * @name GetApiProV1NodeReleaseList
 * @summary Get Node Release List
 * @request GET:/api/pro/v1/node/release/list
 * @response `200` `(DomainPWResponse & {
    data?: (DomainNodeReleaseListItem)[],

})` OK
 */

export const getApiProV1NodeReleaseList = (
  query: GetApiProV1NodeReleaseListParams,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: DomainNodeReleaseListItem[];
    }
  >({
    path: `/api/pro/v1/node/release/list`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description Rollback Node Release
 *
 * @tags node
 * @name PostApiProV1NodeReleaseRollback
 * @summary Rollback Node Release
 * @request POST:/api/pro/v1/node/release/rollback
 * @response `200` `DomainPWResponse` OK
 */

export const postApiProV1NodeReleaseRollback = (
  request: PostApiProV1NodeReleaseRollbackParams,
  params: RequestParams = {},
) =>
  httpRequest<DomainPWResponse>({
    path: `/api/pro/v1/node/release/rollback`,
    method: "POST",
    body: request,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
