package ai

type QueryRequest struct {
	Question  string
	SessionID string
	AgentID   string
	AgentRole string
	AgentName string
	Timezone  string
}
type QueryResult struct {
	Answer       string
	SQL          string
	RowCount     int
	ExecutionMs  int64
	WasCached    bool
	WasCorrected bool
	TokensIn     int
	TokensOut    int
}

type StreamResult struct {
	Answer       string
	SQL          string
	RowCount     int
	ExecutionMs  int64
	WasCached    bool
	WasCorrected bool
	TokensIn     int
	TokensOut    int
	ErrorMessage string
	Cached       bool
}
