package gotots

import (
	"strings"
	"unicode"
)

type Config struct {
	FieldPackageNameToPrefix func(pkg string) string
	IndentWithTabs           bool
	PackageNameForAnonymous  string
}

func (cfg *Config) Init() {
	if cfg.FieldPackageNameToPrefix == nil {
		cfg.FieldPackageNameToPrefix = func(pkg string) string {
			if pkg == "" {
				return ""
			}
			prefix := strings.Replace(pkg, "/", "_", -1)

			return string(unicode.ToUpper(rune(prefix[0]))) + prefix[1:] + "_"
		}
	}
}
