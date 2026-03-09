package ldifdiff

import (
	"bytes"
	"slices"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldif"
)

func writeLdif(queue <-chan actionEntry, writer *bytes.Buffer, delWriter *bytes.Buffer, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	for actionEntry := range queue {
		if *err != nil {
			continue
		}

		for i := 0; i < len(actionEntry.Add); i++ {
			slices.SortFunc(actionEntry.Add[i].Attributes, func(a, b ldap.Attribute) int {
				return strings.Compare(a.Type, b.Type)
			})
		}

		for i := 0; i < len(actionEntry.Mod); i++ {
			addChanges := []ldap.Change{}
			delChanges := []ldap.Change{}
			repChanges := []ldap.Change{}

			for _, chg := range actionEntry.Mod[i].Changes {
				switch chg.Operation {
				case ldap.AddAttribute:
					addChanges = append(addChanges, chg)
				case ldap.DeleteAttribute:
					delChanges = append(delChanges, chg)
				case ldap.ReplaceAttribute:
					repChanges = append(repChanges, chg)
				}
			}

			slices.SortFunc(addChanges, func(a, b ldap.Change) int {
				return strings.Compare(a.Modification.Type, b.Modification.Type)
			})
			slices.SortFunc(delChanges, func(a, b ldap.Change) int {
				return strings.Compare(a.Modification.Type, b.Modification.Type)
			})
			slices.SortFunc(repChanges, func(a, b ldap.Change) int {
				return strings.Compare(a.Modification.Type, b.Modification.Type)
			})

			actionEntry.Mod[i].Changes = slices.Concat(addChanges, delChanges, repChanges)
		}

		lDel, ldifDelErr := ldif.ToLDIF(
			actionEntry.Del,
		)
		if ldifDelErr != nil {
			*err = ldifDelErr
			continue
		}

		l, ldifErr := ldif.ToLDIF(
			actionEntry.Add,
			actionEntry.Mod,
		)
		if ldifErr != nil {
			*err = ldifErr
			continue
		}

		lDelStr, ldifDelMarshErr := ldif.Marshal(lDel)
		if ldifDelMarshErr != nil {
			*err = ldifDelMarshErr
			continue
		}

		lStr, ldifMarshErr := ldif.Marshal(l)
		if ldifMarshErr != nil {
			*err = ldifMarshErr
			continue
		}

		delWriter.WriteString(lDelStr)
		writer.WriteString(lStr)
	}
}
