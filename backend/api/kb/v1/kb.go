package v1

import (
	"github.com/chaitin/panda-wiki/consts"
)

type KBUserListReq struct {
	KBId string `json:"kb_id" query:"kb_id"`
}

type KBUserListItemResp struct {
	ID      string                  `json:"id"`
	Account string                  `json:"account"`
	Role    consts.UserRole         `json:"role"`
	Perm    consts.UserKBPermission `json:"perms"`
}

type KBUserInviteReq struct {
	KBId   string                  `json:"kb_id" validate:"required"`
	UserId string                  `json:"user_id" validate:"required"`
	Perm   consts.UserKBPermission `json:"perm" validate:"required,oneof=full_control doc_manage data_operate"`
}

type KBUserInviteResp struct {
}

type KBUserUpdateReq struct {
	KBId   string                  `json:"kb_id" validate:"required"`
	UserId string                  `json:"user_id" validate:"required"`
	Perm   consts.UserKBPermission `json:"perm" validate:"required,oneof=full_control doc_manage data_operate"`
}

type KBUserUpdateResp struct {
}

type KBUserDeleteReq struct {
	KBId   string `json:"kb_id" query:"kb_id" validate:"required"`
	UserId string `json:"user_id" query:"user_id" validate:"required"`
}

type KBUserDeleteResp struct {
}

type ReindexRAGReq struct {
	KBID            string `json:"kb_id" validate:"required"`
	RecreateDataset bool   `json:"recreate_dataset"`
}

type ReindexRAGResp struct {
	Queued int `json:"queued"`
}
