package signv2

type OutputFormat string

const (
	FormatJSON string = "json"
	FormatYAML string = "yaml"
	FormatCSV  string = "csv"
)

var AvailableFormats = []string{
	FormatJSON,
	FormatYAML,
	FormatCSV,
}
