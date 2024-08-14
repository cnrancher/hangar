package scan

type OutputFormat string

const (
	FormatJSON     string = "json"
	FormatYAML     string = "yaml"
	FormatCSV      string = "csv"
	FormatSPDXCSV  string = "spdx-csv"
	FormatSPDXJSON string = "spdx-json"
)

var AvailableFormats = []string{
	FormatJSON,
	FormatYAML,
	FormatCSV,
	FormatSPDXCSV,
	FormatSPDXJSON,
}
