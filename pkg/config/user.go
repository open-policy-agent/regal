package config

import (
	"errors"
	"io"
	"log"
)

// LoadUser reads the user config: the given file if there is one, or else the one found for the search path, or the
// global config file. The config is nil if none was found, and the default config should be used.
func LoadUser(configFile, searchPath string, debug bool) (cfg *Config, path string, err error) {
	if path = configFile; path == "" {
		if path, err = Find(searchPath, true); err != nil {
			return nil, "", err
		}
	}

	if path == "" {
		if debug {
			log.Println("no user-provided config file found, will use the default config")
		}

		return nil, "", nil // No user config provided, use default
	}

	if debug {
		log.Printf("found user config file: %s", path)
	}

	if cfg, err = FromPath(path); errors.Is(err, io.EOF) {
		log.Printf("user config file %q is empty, will use the default config", path)
	} else if err != nil {
		return nil, path, err
	}

	return cfg, path, nil
}
