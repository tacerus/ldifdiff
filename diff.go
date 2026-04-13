// Package ldifdiff is a fast library that outputs the difference
// between two LDIF files as a valid and importable LDIF (e.g.
// by your LDAP server).
package ldifdiff

import (
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldif"
)

// Used by the implementation program in the cmd directory.
const Version = "v0.2.0"

// Used by the implementation program in the cmd directory.
const Author = "Claudio Ramirez <pub.claudio@gmail.com>"

// Used by the implementation program in the cmd directory.
const Repo = "https://github.com/tacerus/ldifdiff"

type fn func(string, []string, []string) (entries, error)

var skipDnForDelete map[string]bool

/* Public functions */

// DiffLdif compares two *ldif.LDIF structs natively and outputs the differences as an *ldif.LDIF struct.
// An array of attributes of ignore during the comparison can be provided.
func DiffLdif(sourceLdif, targetLdif *ldif.LDIF, ignoreAttr []string, strictAttr []string) (*ldif.LDIF, error) {
	source := convertLdifToEntries(sourceLdif, ignoreAttr)
	target := convertLdifToEntries(targetLdif, ignoreAttr)
	return compareLdif(&source, &target, nil, strictAttr)
}

// Diff compares two LDIF strings (sourceStr and targetStr) and outputs the
// differences as a LDIF string. An array of attributes can be supplied. These
// attributes will be ignored when comparing the LDIF strings.
// The output is a string, a valid LDIF, and can be added to the "target"
// database (the one that created targetStr) in order to make it
// equal to the "source" database (the one that created sourceStr). In case of
// failure, an error is provided.
func Diff(sourceStr, targetStr string, ignoreAttr []string, strictAttr []string) (string, error) {
	return genericDiff(sourceStr, targetStr, ignoreAttr, strictAttr, convertLdifStr, nil)
}

// DiffFromFiles compares two LDIF files (sourceFile and targetFile) and
// outputs the differences as a LDIF string. An array of attributes can be
// supplied. These attributes will be ignored when comparing the LDIF strings.
// The output is a string, a valid LDIF, and can be added to the "target"
// database (the one that created targetFile) in order to make it equal to the
// "source" database (the one that created sourceFile). In case of failure, an
// error is provided.
func DiffFromFiles(sourceFile, targetFile string, ignoreAttr []string, strictAttr []string) (string, error) {
	return genericDiff(sourceFile, targetFile, ignoreAttr, strictAttr, importLdifFile, nil)
}

// ListDiffDn compares two LDIF strings (sourceStr and targetStr) and outputs
// the differences as a list of affected DNs (Dintinguished Names). An array of
// attributes can be supplied. These attributes will be ignored when comparing
// the LDIF strings.
// The output is a string slice. In case of failure, an error is provided.
func ListDiffDn(sourceStr, targetStr string, ignoreAttr []string, strictAttr []string) ([]string, error) {
	dnList := []string{}
	_, err := genericDiff(sourceStr, targetStr, ignoreAttr, strictAttr, convertLdifStr, &dnList)
	return dnList, err
}

// ListDiffDnFromFiles compares two LDIF files (sourceFile and targetFileStr)
// and outputs the differences as a list of affected DNs (Dintinguished Names).
// An array of attributes can be supplied. These attributes will be ignored
// when comparing the LDIF strings.
// The output is a string slice. In case of failure, an error is provided.
func ListDiffDnFromFiles(sourceFile, targetFile string, ignoreAttr []string, strictAttr []string) ([]string, error) {
	dnList := []string{}
	_, err := genericDiff(sourceFile, targetFile, ignoreAttr, strictAttr, importLdifFile, &dnList)
	return dnList, err
}

/* Package private functions */

func convertLdifToEntries(l *ldif.LDIF, ignoreAttr []string) (res entries) {
	res = make(entries)

	if l == nil {
		return
	}

	ignoreAttrMap := make(map[string]struct{})
	for _, attr := range ignoreAttr {
		ignoreAttrMap[attr] = struct{}{}
	}

	for _, e := range l.Entries {
		if e.Entry == nil {
			continue
		}

		dn := e.Entry.DN
		ent := make(entry)

		for _, attr := range e.Entry.Attributes {
			if _, ignore := ignoreAttrMap[attr.Name]; ignore {
				continue
			}

			ent[attr.Name] = attr.Values
		}

		res[dn] = ent
	}

	return
}

func entriesEqual(a, b entry) bool {
	for attr, vals := range a {
		bVals, bFound := b[attr]
		if !bFound || !slices.Equal(vals, bVals) {
			return false
		}
	}

	for attr, vals := range b {
		aVals, aFound := a[attr]
		if !aFound || !slices.Equal(vals, aVals) {
			return false
		}
	}

	return true
}

// Ordering Logic:
// Add: entries from source sorted S -> L. Otherwise is invalid.
// Remove: entries from target sorted L -> S. Otherwise is invalid.
// Modify:
//   - Keep S ->  L ordering
//   - If only 1 instance of attribute with different value on source and target:
//     update. This way we don't break the applicable LDAP schema.
func compareLdif(source, target *entries, dnList *[]string, strictAttr []string) (result *ldif.LDIF, err error) {
	queue := make(chan actionEntry, 10)
	var wg sync.WaitGroup

	result = new(ldif.LDIF)

	// Find the order in which operation must happen
	orderedSourceShortToLong := sortDnByDepth(source, false)
	orderedTargetLongToShort := sortDnByDepth(target, true)

	// Write the file concurrently
	wg.Add(1) // 1 writer
	go buildLdif(queue, result, &wg, &err)

	// Dn only on source + removal of identical entries
	skipDnForDelete = make(map[string]bool) // Keep track of dn to skip at Deletion
	sendForAddition(&orderedSourceShortToLong, source, target, queue, dnList)

	// Dn only on target
	sendForDeletion(&orderedTargetLongToShort, source, target, queue, dnList)

	// Dn on source and target
	sendForModification(&orderedSourceShortToLong, source, target, queue, dnList, strictAttr)

	// Done sending work
	close(queue)

	// Free some memory
	*source = entries{}
	*target = entries{}

	// Wait for the creation of the LDIF
	wg.Wait()

	// Return the results
	return result, err
}

func genericDiff(sourceParam, targetParam string, ignoreAttr, strictAttr []string, fn fn, dnList *[]string) (string, error) {
	// Read the files in memory as a Map with sorted attributes
	var source, target entries
	var sourceErr, targetErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func(entries *entries, wg *sync.WaitGroup, err *error) {
		result, e := fn(sourceParam, ignoreAttr, strictAttr)
		*entries = result
		*err = e
		wg.Done()
	}(&source, &wg, &sourceErr)
	go func(entries *entries, wg *sync.WaitGroup, err *error) {
		result, e := fn(targetParam, ignoreAttr, strictAttr)
		*entries = result
		*err = e
		wg.Done()
	}(&target, &wg, &targetErr)
	wg.Wait()

	if sourceErr != nil {
		return "", sourceErr
	}
	if targetErr != nil {
		return "", targetErr
	}

	result, err := compareLdif(&source, &target, dnList, strictAttr)
	if err != nil || dnList != nil {
		return "", err
	}

	return ldif.Marshal(result)
}

func sendForAddition(
	orderedSourceShortToLong *[]string,
	source, target *entries,
	queue chan<- actionEntry,
	dnList *[]string) {
	for _, dn := range *orderedSourceShortToLong {
		// Mark entries for addition if only on source
		if _, ok := (*target)[dn]; !ok {
			if dnList == nil {
				req := ldap.NewAddRequest(dn, nil)
				for name, vals := range (*source)[dn] {
					req.Attribute(name, vals)
				}
				actionEntry :=
					actionEntry{
						Dn:  dn,
						Add: []*ldap.AddRequest{req},
					}
				queue <- actionEntry
			} else {
				// Always an Add operation (attributes not relevant)
				*dnList = append(*dnList, dn)
			}
			delete(*source, dn)
		}
		// Implict else:
		// It exists on target and it's not equal, so it's a modifyStr
		skipDnForDelete[dn] = true

		if entriesEqual((*source)[dn], (*target)[dn]) {
			delete(*source, dn)
			delete(*target, dn)
			continue
		}

	}
}

func sendForDeletion(
	orderedTargetLongToShort *[]string,
	source, target *entries,
	queue chan<- actionEntry,
	dnList *[]string) {
	for _, dn := range *orderedTargetLongToShort {
		if skipDnForDelete[dn] { // We know it's not a delete operation
			continue
		}
		if _, ok := (*target)[dn]; ok { // It has not been deleted above
			if _, ok := (*source)[dn]; !ok { // does not exists on source
				if dnList == nil {
					actionEntry :=
						actionEntry{
							Dn:  dn,
							Del: []*ldap.DelRequest{ldap.NewDelRequest(dn, nil)},
						}
					queue <- actionEntry
				} else {
					// Always remove (attributes are not relevant)
					*dnList = append(*dnList, dn)
				}
				delete(*target, dn)
			}
			// Implict else:
			// It exists on source and it's not equal (tested on sendForAddition),
			// so it's a modifyStr
		}
	}
	// Free some memory
	skipDnForDelete = nil
}

func sendForModification(
	orderedSourceShortToLong *[]string, source,
	target *entries,
	queue chan<- actionEntry,
	dnList *[]string,
	strictAttr []string,
) {
	for _, dn := range *orderedSourceShortToLong {
		// DN is present on source and target:
		// sendForAdd/Remove clean up source and target
		_, okSource := (*source)[dn]
		_, okTarget := (*target)[dn]

		if okSource && okTarget { // it hasn't been deleted
			if dnList == nil {

				// Store the attributes to be added, deleted or replaced
				attrToModifyAdd := entry{}
				attrToModifyDelete := entry{}
				attrToModifyReplace := entry{}

				for attr, sourceVals := range (*source)[dn] {
					targetVals, ok := (*target)[dn][attr]
					if ok && !slices.Equal(sourceVals, targetVals) || !ok {
						strict := slices.Contains(strictAttr, attr)
						lS := len(sourceVals)
						lT := len(targetVals)
						if (strict && (lS > 0 && lT > 0)) || (!strict && (lS == 1 && lT == 1)) {
							attrToModifyReplace[attr] = sourceVals
						} else {
							targetMap := make(map[string]struct{}, len(targetVals))
							for _, val := range targetVals {
								targetMap[val] = struct{}{}
							}

							for _, val := range sourceVals {
								if _, exists := targetMap[val]; !exists {
									attrToModifyAdd[attr] = append(attrToModifyAdd[attr], val)
								}
							}
						}
					}
				}

				// Compare attribute values starting from the target.
				for attr, targetVals := range (*target)[dn] {
					sourceVals, ok := (*source)[dn][attr]
					// Looking for unique attributes
					if !ok && !(len(sourceVals) == 1 && len(targetVals) == 1) {
						attrToModifyDelete[attr] = targetVals
					} else if _, ok := attrToModifyReplace[attr]; !ok {
						sourceMap := make(map[string]struct{}, len(sourceVals))
						for _, val := range sourceVals {
							sourceMap[val] = struct{}{}
						}

						for _, val := range targetVals {
							if _, exists := sourceMap[val]; !exists {
								attrToModifyDelete[attr] = append(attrToModifyDelete[attr], val)
							}
						}
					}
				}

				// Send it
				req := ldap.NewModifyRequest(dn, nil)
				actionEntry := actionEntry{
					Dn: dn,
				}
				switch {
				case len(attrToModifyAdd) > 0:
					for name, vals := range attrToModifyAdd {
						req.Add(name, vals)
					}
					fallthrough
				case len(attrToModifyDelete) > 0:
					for name, vals := range attrToModifyDelete {
						req.Delete(name, vals)
					}
					fallthrough
				case len(attrToModifyReplace) > 0:
					for name, vals := range attrToModifyReplace {
						req.Replace(name, vals)
					}
				}
				actionEntry.Mod = []*ldap.ModifyRequest{req}
				queue <- actionEntry
			} else {
				// There must be something left to modify
				//if len((*source)[dn]) > 0 || len((*target)[dn]) > 0 {
				*dnList = append(*dnList, dn)
				//}
			}
			// Clean it up
			delete(*source, dn)
			delete(*target, dn)
		}
	}
}

func sortDnByDepth(entries *entries, longToShort bool) []string {
	var sorted []string

	dns := []string{}
	for dn := range *entries {
		dns = append(dns, dn)
	}

	splitByDc := make(map[string][]string)
	longestDn := 0
	// Split the components of the dn and remember the longest size
	for _, dn := range dns {
		parts := strings.Split(dn, ",")
		if len(parts) > longestDn {
			longestDn = len(parts)
		}
		splitByDc[dn] = parts
	}

	// Get the direction of the loop
	componentSizes := []int{}
	if longToShort {
		for i := longestDn; i > 0; i-- {
			componentSizes = append(componentSizes, i)
		}
	} else {
		for i := 1; i <= longestDn; i++ {
			componentSizes = append(componentSizes, i)
		}
	}

	// Sort by size and alpahbetically within size
	for _, size := range componentSizes {
		sameSize := []string{}
		for dn, components := range splitByDc {
			if len(components) == size {
				sameSize = append(sameSize, dn)
			}
		}
		sort.Strings(sameSize)
		sorted = append(sorted, sameSize...)
	}

	return sorted
}
