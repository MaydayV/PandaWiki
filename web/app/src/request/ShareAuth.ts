 
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
  DomainPWResponse,
  DomainResponse,
  GithubComChaitinPandaWikiApiShareV1AuthGetResp,
  V1AuthGitHubReq,
  V1AuthGitHubResp,
  V1AuthLoginSimpleReq,
} from "./types";

/**
 * @description AuthGet
 *
 * @tags share_auth
 * @name GetShareV1AuthGet
 * @summary AuthGet
 * @request GET:/share/v1/auth/get
 * @response `200` `(DomainPWResponse & {
    data?: GithubComChaitinPandaWikiApiShareV1AuthGetResp,

})` OK
 */

export const getShareV1AuthGet = (params: RequestParams = {}) =>
  httpRequest<
    DomainPWResponse & {
      data?: GithubComChaitinPandaWikiApiShareV1AuthGetResp;
    }
  >({
    path: `/share/v1/auth/get`,
    method: "GET",
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description GitHub登录
 *
 * @tags ShareAuth
 * @name PostShareV1AuthGithub
 * @summary GitHub登录
 * @request POST:/share/v1/auth/github
 * @response `200` `(DomainPWResponse & {
    data?: V1AuthGitHubResp,

})` OK
 */

export const postShareV1AuthGithub = (
  param: V1AuthGitHubReq,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: V1AuthGitHubResp;
    }
  >({
    path: `/share/v1/auth/github`,
    method: "POST",
    body: param,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description AuthLoginSimple
 *
 * @tags share_auth
 * @name PostShareV1AuthLoginSimple
 * @summary AuthLoginSimple
 * @request POST:/share/v1/auth/login/simple
 * @response `200` `DomainResponse` OK
 */

export const postShareV1AuthLoginSimple = (
  param: V1AuthLoginSimpleReq,
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse>({
    path: `/share/v1/auth/login/simple`,
    method: "POST",
    body: param,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
