package models

// ProblemDetails represents an RFC 7807 problem details object.
type ProblemDetails struct {
	Type   string      `json:"type"`
	Title  string      `json:"title"`
	Status int         `json:"status"`
	Detail string      `json:"detail"`
	Code   string      `json:"code,omitempty"`
	Errors interface{} `json:"errors,omitempty"`
}

// NewProblem creates a standard RFC 7807 ProblemDetails.
func NewProblem(status int, title, detail, code string, errors interface{}) ProblemDetails {
	t := "https://ontrack.app/problems/" + code
	if code == "" {
		t = "https://ontrack.app/problems/error"
	}
	return ProblemDetails{
		Type:   t,
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
		Errors: errors,
	}
}

