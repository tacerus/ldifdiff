package ldifdiff

import (
	"errors"
	"maps"
	"os"
	"slices"
	"sync"

	"github.com/go-ldap/ldif"
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

func importRecords(ldifStr, file string, ignoreAttr []string, strictAttr []string) (entries, error) {
	var readErr, parseErr error
	queue := make(chan ldif.LDIF, 10)
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
	go parse(entries, ignoreAttr, strictAttr, queue, &wg, &parseErr)
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

func readFile(file string, ignoreAttr []string, queue chan<- ldif.LDIF, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	defer close(queue)

	fh, osErr := os.Open(file)
	if osErr != nil {
		*err = osErr
		return
	}
	defer fh.Close()

	l := new(ldif.LDIF)
	ldifErr := ldif.Unmarshal(fh, l)
	if ldifErr != nil {
		*err = ldifErr
		return
	}

	queue <- *l
}

func readStr(ldifStr string, ignoreAttr []string, queue chan<- ldif.LDIF, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	defer close(queue)

	l, ldifErr := ldif.Parse(ldifStr)
	if ldifErr != nil {
		*err = ldifErr
		return
	}

	queue <- *l
}

func parse(entries entries, ignoreAttr []string, strictAttr []string, queue <-chan ldif.LDIF, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	for record := range queue {
		for _, e := range record.Entries {
			dn := e.Entry.DN
			newEntry := make(entry, len(e.Entry.Attributes))
			entries[dn] = make(entry)

			for _, a := range e.Entry.Attributes {
				if slices.Contains(ignoreAttr, a.Name) {
					continue
				}

				v := a.Values

				if !slices.Contains(strictAttr, a.Name) {
					slices.Sort(v)
				}

				newEntry[a.Name] = v
			}

			for _, attr := range slices.Sorted(maps.Keys(newEntry)) {
				if len(newEntry[attr]) > 0 {
					entries[dn][attr] = newEntry[attr]
				}
			}
		}
	}
}
