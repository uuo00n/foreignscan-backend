package handlers

// ErrorResponse 定义API错误响应的结构
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
