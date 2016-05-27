package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	_, err := flagParser.AddCommand("graph",
		"graph a Bash script",
		"Graph a Bash script, producing all defs, refs, and docs.",
		&graphCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Check that we have the '-i' flag.
	cmd := exec.Command("go", "help", "build")
	o, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	usage := strings.Split(string(o), "\n")[0] // The usage is on the first line.
	matched, err := regexp.MatchString("-i", usage)
	if err != nil {
		log.Fatal(err)
	}
	if !matched {
		log.Fatal("'go build' does not have the '-i' flag. Please upgrade to go1.3+.")
	}
}

type GraphCmd struct{}

var graphCmd GraphCmd

func (c *GraphCmd) Execute(args []string) error {
	inputBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("Failed to read STDIN: %s", err)
	}
	var units unit.SourceUnits
	if err := json.NewDecoder(bytes.NewReader(inputBytes)).Decode(&units); err != nil {
		// Legacy API: try parsing input as a single source unit
		var u *unit.SourceUnit
		if err := json.NewDecoder(bytes.NewReader(inputBytes)).Decode(&u); err != nil {
			return fmt.Errorf("Failed to parse source units from input: %s", err)
		}
		units = unit.SourceUnits{u}
	}
	if err := os.Stdin.Close(); err != nil {
		return fmt.Errorf("Failed to close STDIN: %s", err)
	}

	if len(units) == 0 {
		log.Fatal("Input contains no source unit data.")
	}

	out, err := graphUnits(units)
	if err != nil {
		return fmt.Errorf("Failed to graph source units: %s", err)
	}

	// Make paths relative to repo.
	for _, gs := range out.Defs {
		if gs.File == "" {
			log.Printf("no file %+v", gs)
		}
		if gs.File != "" {
			gs.File = relPath(cwd, gs.File)
		}
	}
	for _, gr := range out.Refs {
		if gr.File != "" {
			gr.File = relPath(cwd, gr.File)
		}
	}
	for _, gd := range out.Docs {
		if gd.File != "" {
			gd.File = relPath(cwd, gd.File)
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return fmt.Errorf("Failed to output graph data: %s", err)
	}
	return nil
}

func graphUnits(units unit.SourceUnits) (*graph.Output, error) {
	output := graph.Output{}

	for _, u := range units {
		for _, f := range u.Files {
			graphFile(f, &output)
		}
	}

	return &output, nil
}

func graphFile(name string, output *graph.Output) error {
	return nil
}
