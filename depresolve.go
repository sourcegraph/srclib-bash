package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib/dep"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	_, err := flagParser.AddCommand("depresolve",
		"resolve a Bash script's imports",
		"Lists the man page repository as a dependency.",
		&depResolveCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type DepResolveCmd struct{}

var depResolveCmd DepResolveCmd

func (c *DepResolveCmd) Execute(args []string) error {
	var unit *unit.SourceUnit
	if err := json.NewDecoder(os.Stdin).Decode(&unit); err != nil {
		return fmt.Errorf("parsing the source unit from STDIN failed with: %s", err)
	}
	if err := os.Stdin.Close(); err != nil {
		return fmt.Errorf("closing STDIN failed with: %s", err)
	}

	var resolutions []*dep.Resolution
	if len(unit.Files) > 0 {
		res := &dep.Resolution{
			Target: &dep.ResolvedTarget{
				ToRepoCloneURL: "github.com/sourcegraph/man-pages-posix",
				ToUnitType:     "ManPages",
				ToUnit:         "man",
			},
		}
		resolutions = append(resolutions, res)
	}

	bytes, err := json.MarshalIndent(resolutions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling resolved units failed with: %s, resolutions: %s", err, resolutions)
	}
	if _, err := os.Stdout.Write(bytes); err != nil {
		return fmt.Errorf("writing output failed with: %s", err)
	}
	fmt.Println()
	return nil
}
