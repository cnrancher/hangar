package cmdconfig

type Provider interface {
	Get(key string) any
	GetString(key string) string
	GetStringSlice(key string) []string
	GetInt(key string) int
	GetBool(key string) bool
	Set(key string, value any)
	IsSet(key string) bool
}
