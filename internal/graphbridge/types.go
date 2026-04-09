package graphbridge

// Request is the JSON payload sent to the Python subprocess via stdin.
type Request struct {
	Command string         `json:"command"`
	Args    map[string]any `json:"args"`
}

// Response is the JSON payload received from the Python subprocess via stdout.
type Response struct {
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data,omitempty"`
	Error string         `json:"error,omitempty"`
}
