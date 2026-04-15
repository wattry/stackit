package foreach

import "sync"

// JSONStatus represents the aggregate outcome of a foreach operation.
type JSONStatus string

const (
	JSONStatusSuccess JSONStatus = "success"
	JSONStatusFailure JSONStatus = "failure"
	JSONStatusError   JSONStatus = "error"
)

// JSONResult represents the JSON output for foreach operations.
type JSONResult struct {
	Status       JSONStatus         `json:"status"`
	Error        string             `json:"error,omitempty"`
	Message      string             `json:"message,omitempty"`
	Command      string             `json:"command,omitempty"`
	Results      []JSONBranchResult `json:"results"`
	TotalCount   int                `json:"total_count"`
	SuccessCount int                `json:"success_count"`
	FailureCount int                `json:"failure_count"`
}

// JSONBranchResult represents one branch result in foreach JSON output.
type JSONBranchResult struct {
	Branch   string       `json:"branch"`
	Status   BranchStatus `json:"status"`
	ExitCode int          `json:"exit_code"`
	Output   string       `json:"output,omitempty"`
	Error    string       `json:"error,omitempty"`
}

// JSONHandler collects foreach events for JSON output.
type JSONHandler struct {
	mu     sync.Mutex
	Result *JSONResult
}

// NewJSONHandler creates a handler that collects foreach results as JSON data.
func NewJSONHandler() *JSONHandler {
	return &JSONHandler{
		Result: &JSONResult{
			Results: []JSONBranchResult{},
		},
	}
}

// OnEvent implements Handler.
func (h *JSONHandler) OnEvent(e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	switch ev := e.(type) {
	case StackDisplayEvent:
		h.Result.Command = ev.Command
	case CompletionEvent:
		h.Result.Message = ev.Message
		h.Result.TotalCount = len(ev.Results)
		h.Result.Results = make([]JSONBranchResult, 0, len(ev.Results))
		h.Result.SuccessCount = 0
		h.Result.FailureCount = 0

		for _, result := range ev.Results {
			branchResult := JSONBranchResult{
				Branch:   result.BranchName,
				Status:   result.Status,
				ExitCode: result.ExitCode,
				Output:   result.Output,
			}
			if result.Error != nil {
				branchResult.Error = result.Error.Error()
			}
			h.Result.Results = append(h.Result.Results, branchResult)

			if result.Status == StatusDone {
				h.Result.SuccessCount++
			} else {
				h.Result.FailureCount++
			}
		}

		if h.Result.FailureCount > 0 {
			h.Result.Status = JSONStatusFailure
		} else {
			h.Result.Status = JSONStatusSuccess
		}
	}
}

// SetError records an action-level error in the JSON result.
func (h *JSONHandler) SetError(err error) {
	if err == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.Result.Error = err.Error()
	if h.Result.FailureCount > 0 {
		h.Result.Status = JSONStatusFailure
		return
	}
	h.Result.Status = JSONStatusError
}
