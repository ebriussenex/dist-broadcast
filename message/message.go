package message

type (
	BroadcastReq struct {
		Message int    `json:"message"`
		Type    string `json:"type"`
		MsgID   int    `json:"msg_id"`
	}

	BroadcastResp struct {
		Type      string `json:"type"`
		InReplyTo int    `json:"in_reply_to"`
	}

	SyncReq struct {
		Messages []int  `json:"messages"`
		Type     string `json:"type"`
	}

	SyncResp struct {
		Type      string `json:"type"`
		InReplyTo int    `json:"in_reply_to"`
	}

	ReadReq struct {
		Type string `json:"type"`
	}

	ReadResp struct {
		Type     string `json:"type"`
		Messages []int  `json:"messages"`
	}

	TopologyReq struct {
		Type     string              `json:"type"`
		Topology map[string][]string `json:"topology"`
	}

	TopologyResp struct {
		Type string `json:"type"`
	}
)
