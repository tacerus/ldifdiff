package ldifdiff

import (
	"slices"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldif"
)

func buildLdif(queue <-chan actionEntry, result *ldif.LDIF, wg *sync.WaitGroup, err *error) {
	defer wg.Done()

	var delEntries []*ldif.Entry
	var addModEntries []*ldif.Entry

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

		if len(actionEntry.Del) > 0 {
			lDel, ldifDelErr := ldif.ToLDIF(actionEntry.Del)
			if ldifDelErr != nil {
				*err = ldifDelErr
				continue
			}

			delEntries = append(delEntries, lDel.Entries...)
		}

		if len(actionEntry.Add) > 0 || len(actionEntry.Mod) > 0 {
			l, ldifErr := ldif.ToLDIF(actionEntry.Add, actionEntry.Mod)
			if ldifErr != nil {
				*err = ldifErr
				continue
			}

			addModEntries = append(addModEntries, l.Entries...)
		}
	}

	result.Entries = append(delEntries, addModEntries...)
}
