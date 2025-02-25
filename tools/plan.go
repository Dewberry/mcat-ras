package tools

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// PlanFileContents keywords and data container for ras plan file search
type PlanFileContents struct {
	Path            string
	Hash            string
	FileExt         string //`json:"File Extension"`
	PlanTitle       string //`json:"Plan Title"`
	ShortIdentifier string //`json:"Short Identifier"`
	ProgramVersion  string //`json:"Program Version"`
	GeomFile        string //`json:"Geom File"`
	FlowFile        string //`json:"Flow File"` // unsteady or steady both flow files are stored as FlowFile in HEC RAS plan file, replicating the same here
	FlowRegime      string //`json:"FlowRegime"`
	Description     string //`json:"Description"`
	Notes           string
}

// getPlanData Reads a plan file. returns none to allow concurrency
func getPlanData(rm *RasModel, fn string, wg *sync.WaitGroup) {
	defer wg.Done()

	meta := PlanFileContents{Path: fn, FileExt: filepath.Ext(fn)}

	var err error
	msg := fmt.Sprintf("%s failed to process.", filepath.Base(fn))
	defer func() {
		meta.Notes += msg
		rm.Metadata.PlanFiles = append(rm.Metadata.PlanFiles, meta)
		if err != nil {
			log.Println(err)
		}
	}()

	f, err := rm.FileStore.GetObject(fn)
	if err != nil {
		return
	}
	defer f.Close()

	hasher := sha256.New()

	fs := io.TeeReader(f, hasher) // fs is still a stream
	sc := bufio.NewScanner(fs)

	var line string
	for sc.Scan() {

		line = sc.Text()

		match, err := regexp.MatchString("=", line)
		if err != nil {
			return
		}

		beginDescription, err := regexp.MatchString("BEGIN DESCRIPTION", line)
		if err != nil {
			return
		}

		flowRegime, err := regexp.MatchString("Subcritical|Supercritical|Mixed", line)
		if err != nil {
			return
		}

		if match {
			data := strings.Split(line, "=")

			switch data[0] {

			case "Plan Title":
				meta.PlanTitle = data[1]

			case "Short Identifier":
				meta.ShortIdentifier = data[1]

			case "Program Version":
				meta.ProgramVersion = data[1]

			case "Geom File":
				meta.GeomFile = data[1]

			case "Flow File":
				meta.FlowFile = data[1]

			}

		} else if beginDescription {

			for sc.Scan() {
				line = sc.Text()
				endDescription, _ := regexp.MatchString("END DESCRIPTION", line)

				if endDescription {
					break

				} else {
					if line != "" {
						meta.Description += line + "\n"
					}
				}

			}

		} else if flowRegime {
			meta.FlowRegime = line
		}
	}
	msg = ""
	meta.Hash = fmt.Sprintf("%x", hasher.Sum(nil))

	return

}
