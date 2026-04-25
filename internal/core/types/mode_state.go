package types

type FreeChatState struct {
	RuntimeID         string `json:"runtime_id"`
	TurnCount         int64  `json:"turn_count"`
	LastUserText      string `json:"last_user_text,omitempty"`
	LastAssistantText string `json:"last_assistant_text,omitempty"`
}

type ProjectConsultState struct {
	RuntimeID         string `json:"runtime_id"`
	QueryCount        int64  `json:"query_count"`
	LastQuestion      string `json:"last_question,omitempty"`
	LastAssistantText string `json:"last_assistant_text,omitempty"`
}
