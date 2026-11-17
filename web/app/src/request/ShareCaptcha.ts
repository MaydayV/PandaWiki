 
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
  ConstsRedeemCaptchaReq,
  GocapChallengeData,
  GocapVerificationResult,
} from "./types";

/**
 * @description CreateCaptcha
 *
 * @tags share_captcha
 * @name PostShareV1CaptchaChallenge
 * @summary CreateCaptcha
 * @request POST:/share/v1/captcha/challenge
 * @response `200` `GocapChallengeData` OK
 */

export const postShareV1CaptchaChallenge = (params: RequestParams = {}) =>
  httpRequest<GocapChallengeData>({
    path: `/share/v1/captcha/challenge`,
    method: "POST",
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description RedeemCaptcha
 *
 * @tags share_captcha
 * @name PostShareV1CaptchaRedeem
 * @summary RedeemCaptcha
 * @request POST:/share/v1/captcha/redeem
 * @response `200` `GocapVerificationResult` OK
 */

export const postShareV1CaptchaRedeem = (
  body: ConstsRedeemCaptchaReq,
  params: RequestParams = {},
) =>
  httpRequest<GocapVerificationResult>({
    path: `/share/v1/captcha/redeem`,
    method: "POST",
    body: body,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
