package conf

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

func LoadJSONConfig(path string, cfg any) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func PickConfigPathFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c":
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		case strings.HasPrefix(a, "-c="):
			return strings.TrimPrefix(a, "-c=")
		}
	}
	return ""
}
