package message

type (
	msgCommon struct {
		InReply int    `json:"in_reply"`
		Type    string `json:"type"`
		MsgID   int    `json:"msg_id"`
	}

	BroadcastReq struct {
		Message int    `json:"message"`
		Type    string `json:"type"`
	}

	BroadcastResp struct {
		Type string `json:"type"`
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
