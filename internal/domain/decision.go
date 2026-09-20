package domain

// Action is the application-owned routing action.
type Action string

const (
	ActionForward Action = "forward"
	ActionReject  Action = "reject"
	ActionClarify Action = "clarify"
)

// Decision contains both the classifier result and the auditable routing reason.
type Decision struct {
	Action         Action         `json:"action"`
	Classification Classification `json:"classification"`
	Reason         string         `json:"reason,omitempty"`
}
