package connection

// GroupInfo 群组信息
type GroupInfo struct {
	ID     int64  `json:"group_id"`
	Name   string `json:"group_name"`
	Active bool   `json:"active"`
}

// Message 消息结构
type Message struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   string      `json:"echo,omitempty"`
}

// SendMessageParams 发送消息参数
type SendMessageParams struct {
	MessageType string `json:"message_type,omitempty"`
	GroupID     int64  `json:"group_id,omitempty"`
	UserID      int64  `json:"user_id,omitempty"`
	Message     string `json:"message"`
}
