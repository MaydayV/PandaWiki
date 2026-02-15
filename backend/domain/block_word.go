package domain

type BlockWords struct {
	Words []string `json:"words"`
}

type CreateBlockWordsReq struct {
	KBID       string   `json:"kb_id" validate:"required"`
	BlockWords []string `json:"block_words"`
}
