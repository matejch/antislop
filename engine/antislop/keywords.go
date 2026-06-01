// antislop-ignore-file
package antislop

// TodoKeywords are markers that indicate incomplete/placeholder code.
var TodoKeywords = map[string]bool{
	"TODO":        true,
	"FIXME":       true,
	"HACK":        true,
	"XXX":         true,
	"TEMP":        true,
	"PLACEHOLDER": true,
	"STUB":        true,
}

// ExecKeyword is the Python exec() builtin function name.
const ExecKeyword = "exec"

// ExecCommandPrefix is the Go os/exec command invocation pattern.
const ExecCommandPrefix = "exec.Command"
