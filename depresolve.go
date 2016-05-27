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
		"Performs no operation, provided for compatibility.",
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

	// NOTE(mate): intentionally empty (at least for now)
	var resolutions []*dep.Resolution

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
