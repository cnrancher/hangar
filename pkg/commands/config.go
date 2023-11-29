package commands

import (
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/cmdconfig"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

func initializeFlagsConfig(cmd *cobra.Command, cfg cmdconfig.Provider) {
	if cmd.Parent() != nil {
		initializeFlagsConfig(cmd.Parent(), cfg)
	}

	flags := cmd.Flags()
	cmd.Flags().VisitAll(func(f *flag.Flag) {
		setValueFromFlag(flags, f.Name, cfg, "")
	})
}

func setValueFromFlag(flags *flag.FlagSet, key string, cfg cmdconfig.Provider, targetKey string) {
	key = strings.TrimSpace(key)
	if flags.Lookup(key) != nil || flags.Changed(key) {
		f := flags.Lookup(key)
		configKey := key
		if targetKey != "" {
			configKey = targetKey
		}
		// Gotta love this API too.
		switch f.Value.Type() {
		case "bool":
			bv, _ := flags.GetBool(key)
			cfg.Set(configKey, bv)
		case "string":
			cfg.Set(configKey, f.Value.String())
		case "stringSlice":
			sv, _ := flags.GetStringSlice(key)
			cfg.Set(configKey, sv)
		case "int":
			iv, _ := flags.GetInt(key)
			cfg.Set(configKey, iv)
		case "stringArray":
			sav, _ := flags.GetStringArray(key)
			cfg.Set(configKey, sav)
		case "duration":
			d, _ := flags.GetDuration(key)
			cfg.Set(configKey, d)
		default:
			panic(fmt.Sprintf("update switch with %s", f.Value.Type()))
		}
	}
}
