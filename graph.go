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
			page, hasPage := manPages[ident]
			if hasPage {
				// ref to a standard command
				ref, err := makeCommandRef(name, ident, page, offset)
				if err != nil {
					return fmt.Errorf("failed to create command ref: %s", err)
				}
				output.Refs = append(output.Refs, ref)
			} else {
				// possible def and ref to user-defined function
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
			}
			prevTok = tok
			prevIdent = ident
		}
	}

	return nil
}

func makeRef(filename string, ident string, offset int, isDef bool) (*graph.Ref, error) {
	return &graph.Ref{
		DefUnitType: "BashDirectory",
		DefUnit:     "bash",
		DefPath:     filename + "/" + ident,
		UnitType:    "BashDirectory",
		Unit:        "bash",
		Def:         isDef,
		File:        filename,
		Start:       uint32(offset - len(ident)),
		End:         uint32(offset),
	}, nil
}

func makeDef(filename string, ident string, offset int) (*graph.Def, error) {
	data, err := json.Marshal(DefData{
		Name:    ident,
		Keyword: "function",
		Kind:    "function",
	})
	if err != nil {
		return nil, err
	}
	return &graph.Def{
		DefKey: graph.DefKey{
			UnitType: "BashDirectory",
			Unit:     "bash",
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

func makeCommandRef(filename string, command string, page string, offset int) (*graph.Ref, error) {
	return &graph.Ref{
		DefRepo:     "github.com/sourcegraph/man-pages-posix",
		DefUnitType: "ManPages",
		DefUnit:     "man",
		DefPath:     page + "/" + command,
		UnitType:    "BashDirectory",
		Unit:        "bash",
		Def:         false,
		File:        filename,
		Start:       uint32(offset - len(command)),
		End:         uint32(offset),
	}, nil
}

type DefData struct {
	Name      string
	Keyword   string
	Type      string
	Kind      string
	Separator string
}

var manPages = map[string]string{
	"admin":      "man1p/admin.1p.txt",
	"alias":      "man1p/alias.1p.txt",
	"ar":         "man1p/ar.1p.txt",
	"asa":        "man1p/asa.1p.txt",
	"at":         "man1p/at.1p.txt",
	"awk":        "man1p/awk.1p.txt",
	"basename":   "man1p/basename.1p.txt",
	"batch":      "man1p/batch.1p.txt",
	"bc":         "man1p/bc.1p.txt",
	"bg":         "man1p/bg.1p.txt",
	"break":      "man1p/break.1p.txt",
	"c99":        "man1p/c99.1p.txt",
	"cal":        "man1p/cal.1p.txt",
	"cat":        "man1p/cat.1p.txt",
	"cd":         "man1p/cd.1p.txt",
	"cflow":      "man1p/cflow.1p.txt",
	"chgrp":      "man1p/chgrp.1p.txt",
	"chmod":      "man1p/chmod.1p.txt",
	"chown":      "man1p/chown.1p.txt",
	"cksum":      "man1p/cksum.1p.txt",
	"cmp":        "man1p/cmp.1p.txt",
	"colon":      "man1p/colon.1p.txt",
	"comm":       "man1p/comm.1p.txt",
	"command":    "man1p/command.1p.txt",
	"compress":   "man1p/compress.1p.txt",
	"continue":   "man1p/continue.1p.txt",
	"cp":         "man1p/cp.1p.txt",
	"crontab":    "man1p/crontab.1p.txt",
	"csplit":     "man1p/csplit.1p.txt",
	"ctags":      "man1p/ctags.1p.txt",
	"cut":        "man1p/cut.1p.txt",
	"cxref":      "man1p/cxref.1p.txt",
	"date":       "man1p/date.1p.txt",
	"dd":         "man1p/dd.1p.txt",
	"delta":      "man1p/delta.1p.txt",
	"df":         "man1p/df.1p.txt",
	"diff":       "man1p/diff.1p.txt",
	"dirname":    "man1p/dirname.1p.txt",
	"dot":        "man1p/dot.1p.txt",
	"du":         "man1p/du.1p.txt",
	"echo":       "man1p/echo.1p.txt",
	"ed":         "man1p/ed.1p.txt",
	"env":        "man1p/env.1p.txt",
	"eval":       "man1p/eval.1p.txt",
	"ex":         "man1p/ex.1p.txt",
	"exec":       "man1p/exec.1p.txt",
	"exit":       "man1p/exit.1p.txt",
	"expand":     "man1p/expand.1p.txt",
	"export":     "man1p/export.1p.txt",
	"expr":       "man1p/expr.1p.txt",
	"false":      "man1p/false.1p.txt",
	"fc":         "man1p/fc.1p.txt",
	"fg":         "man1p/fg.1p.txt",
	"file":       "man1p/file.1p.txt",
	"find":       "man1p/find.1p.txt",
	"fold":       "man1p/fold.1p.txt",
	"fort77":     "man1p/fort77.1p.txt",
	"fuser":      "man1p/fuser.1p.txt",
	"gencat":     "man1p/gencat.1p.txt",
	"get":        "man1p/get.1p.txt",
	"getconf":    "man1p/getconf.1p.txt",
	"getopts":    "man1p/getopts.1p.txt",
	"grep":       "man1p/grep.1p.txt",
	"hash":       "man1p/hash.1p.txt",
	"head":       "man1p/head.1p.txt",
	"iconv":      "man1p/iconv.1p.txt",
	"id":         "man1p/id.1p.txt",
	"ipcrm":      "man1p/ipcrm.1p.txt",
	"ipcs":       "man1p/ipcs.1p.txt",
	"jobs":       "man1p/jobs.1p.txt",
	"join":       "man1p/join.1p.txt",
	"kill":       "man1p/kill.1p.txt",
	"lex":        "man1p/lex.1p.txt",
	"link":       "man1p/link.1p.txt",
	"ln":         "man1p/ln.1p.txt",
	"locale":     "man1p/locale.1p.txt",
	"localedef":  "man1p/localedef.1p.txt",
	"logger":     "man1p/logger.1p.txt",
	"logname":    "man1p/logname.1p.txt",
	"lp":         "man1p/lp.1p.txt",
	"ls":         "man1p/ls.1p.txt",
	"m4":         "man1p/m4.1p.txt",
	"mailx":      "man1p/mailx.1p.txt",
	"make":       "man1p/make.1p.txt",
	"man":        "man1p/man.1p.txt",
	"mesg":       "man1p/mesg.1p.txt",
	"mkdir":      "man1p/mkdir.1p.txt",
	"mkfifo":     "man1p/mkfifo.1p.txt",
	"more":       "man1p/more.1p.txt",
	"mv":         "man1p/mv.1p.txt",
	"newgrp":     "man1p/newgrp.1p.txt",
	"nice":       "man1p/nice.1p.txt",
	"nl":         "man1p/nl.1p.txt",
	"nm":         "man1p/nm.1p.txt",
	"nohup":      "man1p/nohup.1p.txt",
	"od":         "man1p/od.1p.txt",
	"paste":      "man1p/paste.1p.txt",
	"patch":      "man1p/patch.1p.txt",
	"pathchk":    "man1p/pathchk.1p.txt",
	"pax":        "man1p/pax.1p.txt",
	"pr":         "man1p/pr.1p.txt",
	"printf":     "man1p/printf.1p.txt",
	"prs":        "man1p/prs.1p.txt",
	"ps":         "man1p/ps.1p.txt",
	"pwd":        "man1p/pwd.1p.txt",
	"qalter":     "man1p/qalter.1p.txt",
	"qdel":       "man1p/qdel.1p.txt",
	"qhold":      "man1p/qhold.1p.txt",
	"qmove":      "man1p/qmove.1p.txt",
	"qmsg":       "man1p/qmsg.1p.txt",
	"qrerun":     "man1p/qrerun.1p.txt",
	"qrls":       "man1p/qrls.1p.txt",
	"qselect":    "man1p/qselect.1p.txt",
	"qsig":       "man1p/qsig.1p.txt",
	"qstat":      "man1p/qstat.1p.txt",
	"qsub":       "man1p/qsub.1p.txt",
	"read":       "man1p/read.1p.txt",
	"readonly":   "man1p/readonly.1p.txt",
	"renice":     "man1p/renice.1p.txt",
	"return":     "man1p/return.1p.txt",
	"rm":         "man1p/rm.1p.txt",
	"rmdel":      "man1p/rmdel.1p.txt",
	"rmdir":      "man1p/rmdir.1p.txt",
	"sact":       "man1p/sact.1p.txt",
	"sccs":       "man1p/sccs.1p.txt",
	"sed":        "man1p/sed.1p.txt",
	"set":        "man1p/set.1p.txt",
	"sh":         "man1p/sh.1p.txt",
	"shift":      "man1p/shift.1p.txt",
	"sleep":      "man1p/sleep.1p.txt",
	"sort":       "man1p/sort.1p.txt",
	"split":      "man1p/split.1p.txt",
	"strings":    "man1p/strings.1p.txt",
	"strip":      "man1p/strip.1p.txt",
	"stty":       "man1p/stty.1p.txt",
	"tabs":       "man1p/tabs.1p.txt",
	"tail":       "man1p/tail.1p.txt",
	"talk":       "man1p/talk.1p.txt",
	"tee":        "man1p/tee.1p.txt",
	"test":       "man1p/test.1p.txt",
	"time":       "man1p/time.1p.txt",
	"times":      "man1p/times.1p.txt",
	"touch":      "man1p/touch.1p.txt",
	"tput":       "man1p/tput.1p.txt",
	"tr":         "man1p/tr.1p.txt",
	"trap":       "man1p/trap.1p.txt",
	"true":       "man1p/true.1p.txt",
	"tsort":      "man1p/tsort.1p.txt",
	"tty":        "man1p/tty.1p.txt",
	"type":       "man1p/type.1p.txt",
	"ulimit":     "man1p/ulimit.1p.txt",
	"umask":      "man1p/umask.1p.txt",
	"unalias":    "man1p/unalias.1p.txt",
	"uname":      "man1p/uname.1p.txt",
	"uncompress": "man1p/uncompress.1p.txt",
	"unexpand":   "man1p/unexpand.1p.txt",
	"unget":      "man1p/unget.1p.txt",
	"uniq":       "man1p/uniq.1p.txt",
	"unlink":     "man1p/unlink.1p.txt",
	"unset":      "man1p/unset.1p.txt",
	"uucp":       "man1p/uucp.1p.txt",
	"uudecode":   "man1p/uudecode.1p.txt",
	"uuencode":   "man1p/uuencode.1p.txt",
	"uustat":     "man1p/uustat.1p.txt",
	"uux":        "man1p/uux.1p.txt",
	"val":        "man1p/val.1p.txt",
	"vi":         "man1p/vi.1p.txt",
	"wait":       "man1p/wait.1p.txt",
	"wc":         "man1p/wc.1p.txt",
	"what":       "man1p/what.1p.txt",
	"who":        "man1p/who.1p.txt",
	"write":      "man1p/write.1p.txt",
	"xargs":      "man1p/xargs.1p.txt",
	"yacc":       "man1p/yacc.1p.txt",
	"zcat":       "man1p/zcat.1p.txt",
}
