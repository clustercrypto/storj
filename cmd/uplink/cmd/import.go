// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	libuplink "storj.io/storj/lib/uplink"
	"storj.io/storj/pkg/cfgstruct"
	"storj.io/storj/pkg/process"
)

var importCfg struct {
	Overwrite bool `default:"false" help:"if true, allows a access to be overwritten" source:"flag"`

	UplinkFlags
}

func init() {
	importCmd := &cobra.Command{
		Use:   "import NAME PATH",
		Short: "Imports an access under the given name from the supplied path",
		Args:  cobra.ExactArgs(2),
		RunE:  importMain,
	}
	RootCmd.AddCommand(importCmd)

	// We don't really want all of the uplink flags on this command but
	// otherwise, there is difficulty getting the config to load right since
	// configuration/flag code assumes it needs to load/persist everything from
	// flags.
	// TODO: revisit after the configuration/flag code is refactored.
	process.Bind(importCmd, &importCfg, defaults, cfgstruct.ConfDir(confDir))
}

// importMain is the function executed when importCmd is called
func importMain(cmd *cobra.Command, args []string) (err error) {
	name := args[0]
	path := args[1]

	// This is a little hacky but viper deserializes accesses into a map[string]interface{}
	// and complains if we try and override with map[string]string{}.
	accesses := map[string]interface{}{}
	for k, v := range importCfg.Accesses {
		accesses[k] = v
	}

	overwritten := false
	if _, ok := accesses[name]; ok {
		if !importCfg.Overwrite {
			return fmt.Errorf("access %q already exists", name)
		}
		overwritten = true
	}

	accessData, err := readFirstUncommentedLine(path)
	if err != nil {
		return Error.Wrap(err)
	}

	// Parse the scope data to ensure it is well formed
	if _, err := libuplink.ParseScope(accessData); err != nil {
		return Error.Wrap(err)
	}

	accesses[name] = accessData

	// There is no easy way currently to save off a "hidden" configurable into
	// the config file without a larger refactoring. For now, just do a manual
	// override of the accesses.
	// TODO: revisit when the configuration/flag code makes it easy
	err = process.SaveConfig(cmd, filepath.Join(confDir, process.DefaultCfgFilename),
		process.SaveConfigWithOverride("accesses", accesses),
		process.SaveConfigRemovingDeprecated())
	if err != nil {
		return Error.Wrap(err)
	}

	if overwritten {
		fmt.Printf("access %q overwritten.\n", name)
	} else {
		fmt.Printf("access %q imported.\n", name)
	}
	return nil
}

func readFirstUncommentedLine(path string) (_ string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", Error.Wrap(err)
	}
	defer func() { err = errs.Combine(err, f.Close()) }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line[0] == '#' {
			continue
		}
		return line, nil
	}

	if err := scanner.Err(); err != nil {
		return "", Error.Wrap(err)
	}

	return "", Error.New("no data found")
}
