 
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
  DomainCommentReq,
  DomainPWResponse,
  GetShareV1CommentListParams,
  ShareShareCommentLists,
} from "./types";

/**
 * @description CreateComment
 *
 * @tags share_comment
 * @name PostShareV1Comment
 * @summary CreateComment
 * @request POST:/share/v1/comment
 * @response `200` `(DomainPWResponse & {
    data?: string,

})` CommentID
 */

export const postShareV1Comment = (
  comment: DomainCommentReq,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: string;
    }
  >({
    path: `/share/v1/comment`,
    method: "POST",
    body: comment,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description GetCommentList
 *
 * @tags share_comment
 * @name GetShareV1CommentList
 * @summary GetCommentList
 * @request GET:/share/v1/comment/list
 * @response `200` `(DomainPWResponse & {
    data?: ShareShareCommentLists,

})` CommentList
 */

export const getShareV1CommentList = (
  query: GetShareV1CommentListParams,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: ShareShareCommentLists;
    }
  >({
    path: `/share/v1/comment/list`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
