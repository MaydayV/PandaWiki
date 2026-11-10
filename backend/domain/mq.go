package domain

type NodeReleaseVectorRequest struct {
	KBID          string `json:"kb_id"`
	NodeReleaseID string `json:"node_release_id"`
	NodeID        string `json:"node_id"`
	DocID         string `json:"doc_id"` // for delete
	Action        string `json:"action"` // upsert, delete, summary
	GroupIds      []int  `json:"group_ids"`
	RequestID     string `json:"request_id,omitempty"`
	// Force 只跳过内容级幂等检查，用于模型切换、手动重新学习等必须重跑的场景；
	// 同一个 RequestID 的重复投递仍会被请求级幂等拦截。
	Force bool `json:"force,omitempty"`
}
