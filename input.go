package ldifdiff

import (
	"bufio"
	"errors"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
)

/* Package only functions */

func convertLdifStr(ldifStr string, ignoreAttr []string, strictAttr []string) (entries, error) {
	return importRecords(ldifStr, "", ignoreAttr, strictAttr)
}

func importLdifFile(file string, ignoreAttr []string, strictAttr []string) (entries, error) {
	entries, err := importRecords("", file, ignoreAttr, strictAttr)
	if err != nil {
		err = errors.New(err.Error() + " [" + file + "]")
	}
	return entries, err
}

/* Internal functions */

func addLineToRecord(line *string, record *[]string, ignoreAttr []string, prevAttrSkipped *bool) error {
	var err error

	// Append continuation lines to previous line
	if strings.HasPrefix(*line, " ") {
		if *prevAttrSkipped { // but not lines from a skipped attribute
			return nil
		}
		switch len(*record) > 0 {
		case true:
			prevIdx := len(*record) - 1
			prevLine := strings.TrimSuffix((*record)[prevIdx], "\n") +
				strings.TrimPrefix(*line, " ")
			(*record)[prevIdx] = prevLine
			return nil
		case false:
			err = errors.New("Invalid modifyStr line continuation: \"" + *line + "\"")
			return err
		}
	}

	// Regular line
	if len(*line) != 0 {
		for _, attrName := range ignoreAttr {
			if strings.HasPrefix(*line, attrName+":") {
				*prevAttrSkipped = true
				return nil
			}
		}
		*record = append(*record, *line)
		*prevAttrSkipped = false
	}

	return nil
}

func importRecords(ldifStr, file string, ignoreAttr []string, strictAttr []string) (entries, error) {
	var readErr, parseErr error
	queue := make(chan []string, 10)
	entries := make(entries)

	// Read and Parse the file concurrently
	var wg sync.WaitGroup
	wg.Add(2) // 1 reader + 1 parser
	switch file {
	case "": // it's a ldifStr
		go readStr(ldifStr, ignoreAttr, queue, &wg, &readErr)
	default: // it's a file
		go readFile(file, ignoreAttr, queue, &wg, &readErr)
	}
	go parse(entries, strictAttr, queue, &wg, &parseErr)
	wg.Wait()

	// Return values
	switch {
	case readErr != nil:
		return entries, readErr
	case parseErr != nil:
		return entries, parseErr
	default:
		return entries, nil
	}
}

func readFile(file string, ignoreAttr []string, queue chan<- []string, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	defer close(queue)
	fh, osErr := os.Open(file)
	if osErr != nil {
		*err = osErr
		return
	}
	defer fh.Close()

	record := []string{}
	scanner := bufio.NewScanner(fh)
	var prevAttrSkipped bool // use to skip continuation lines of skipped attr
	firstLine := true
	for scanner.Scan() {

		line := scanner.Text()

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Check if first line is a "version: *" line and skip it
		if firstLine {
			if strings.HasPrefix(line, "version: ") {
				firstLine = false
				continue
			} else if line == "" {
				continue
			}
			firstLine = false
		}

		// Import lines as records
		*err = addLineToRecord(&line, &record, ignoreAttr, &prevAttrSkipped)
		if *err != nil {
			return
		}

		//  Dispatch the record to buffer & reset record
		if len(line) == 0 && len(record) != 0 {
			queue <- record
			record = []string{}
		}
	}

	// Last record may be a leftover (no empty line)
	if len(record) != 0 {
		queue <- record
	}

	if *err == nil {
		*err = scanner.Err()
	}
}

func readStr(ldifStr string, ignoreAttr []string, queue chan<- []string, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	defer close(queue)

	for idx, recordStr := range strings.Split(ldifStr, "\n\n") {
		var prevAttrSkipped bool
		record := []string{}
		for _, line := range strings.Split(recordStr, "\n") {

			// Skip comments
			if strings.HasPrefix(line, "#") {
				continue
			}

			if idx == 0 { // First record only
				if strings.HasPrefix(line, "version: ") {
					continue
				}
			}

			*err = addLineToRecord(&line, &record, ignoreAttr, &prevAttrSkipped)
			if *err != nil {
				return
			}

			//  Dispatch the record to buffer & reset record
			if len(line) == 0 && len(record) != 0 {
				queue <- record
				record = []string{}
			}
		}
		// Last record may be a leftover (no empty line)
		if len(record) != 0 {
			queue <- record
		}
	}
}

func parse(entries entries, strictAttr []string, queue <-chan []string, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	for record := range queue {
		dn := record[0] // Find dn, should be the first line
		if !strings.HasPrefix(dn, "dn:") {
			*err = errors.New("No dn could be retrieved")
			continue
		}

		// Sort the entries
		rest := record[1:]

		newEntry := make(entry, len(rest))
		entries[dn] = make(entry)

		for _, line := range rest {
			parts := strings.SplitN(line, ": ", 2)
			attr := parts[0]
			val := parts[1]
			newEntry[attr] = append(newEntry[attr], val)
		}

		for attr := range maps.Keys(newEntry) {
			if !slices.Contains(strictAttr, attr) {
				sort.Strings(newEntry[attr])
			}
		}

		for _, attr := range slices.Sorted(maps.Keys(newEntry)) {
			if len(newEntry[attr]) > 0 {
				entries[dn][attr] = newEntry[attr]
			}
		}
	}
}
