package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mkovacs/bash/scanner"

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
	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("Failed to open file %s: %s", name, err)
	}
	defer f.Close()

	sc := scanner.Scanner{}
	sc.Init(bufio.NewReader(f))
	prevTok := scanner.Nothing
	prevIdent := ""
loop:
	for {
		tok, err := sc.Scan()
		if err != nil {
			return fmt.Errorf("failed to scan for identifiers: %s", err)
		}
		switch tok {
		case scanner.EOF:
			break loop
		case scanner.Ident:
			ident := sc.TokenText()
			offset := sc.Pos().Offset
			// fmt.Fprintf(os.Stderr, "ident: \"%s\" at %d\n", ident, offset-len(ident))
			isDef := false
			if prevTok == scanner.Ident && prevIdent == "function" {
				def, err := makeDef(name, ident, offset)
				if err != nil {
					return fmt.Errorf("failed to create def: %s", err)
				}
				output.Defs = append(output.Defs, def)
				isDef = true
			}
			ref, err := makeRef(name, ident, offset, isDef)
			if err != nil {
				return fmt.Errorf("failed to create ref: %s", err)
			}
			output.Refs = append(output.Refs, ref)
			prevTok = tok
			prevIdent = ident
		}
	}

	return nil
}

func makeRef(filename string, ident string, offset int, isDef bool) (*graph.Ref, error) {
	return &graph.Ref{
		DefUnitType: "BashDirectory",
		DefPath:     filename + "/" + ident,
		UnitType:    "BashDirectory",
		Def:         isDef,
		File:        filename,
		Start:       uint32(offset - len(ident)),
		End:         uint32(offset),
	}, nil
}

func makeDef(filename string, ident string, offset int) (*graph.Def, error) {
	data, err := json.Marshal(DefData{
		Name:      ident,
		Keyword:   "function",
		Kind:      "function",
		Type:      " ",
		Separator: " ",
	})
	if err != nil {
		return nil, err
	}
	return &graph.Def{
		DefKey: graph.DefKey{
			UnitType: "BashDirectory",
			Path:     filename + "/" + ident,
		},
		Exported: true,
		Data:     data,
		Name:     ident,
		Kind:     "function",
		File:     filename,
		DefStart: uint32(offset - len(ident)),
		DefEnd:   uint32(offset),
	}, nil
}

type DefData struct {
	Name      string
	Keyword   string
	Type      string
	Kind      string
	Separator string
}
