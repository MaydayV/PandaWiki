package v1

import "github.com/chaitin/panda-wiki/domain"

type ShareFileUploadReq struct {
	KbId         string         `json:"-"`
	File         string         `form:"file"`
	CaptchaToken string         `form:"captcha_token" json:"captcha_token" validate:"required"`
	AppType      domain.AppType `form:"app_type" json:"app_type" validate:"required,oneof=1 2"`
}

type FileUploadResp struct {
	Key string `json:"key"`
}

type ShareFileUploadUrlReq struct {
	KbId         string `json:"-"`
	Url          string `json:"url" validate:"required,url"`
	CaptchaToken string `json:"captcha_token" validate:"required"`
}

type ShareFileUploadUrlResp struct {
	Key string `json:"key"`
}
