package ldifdiff

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
)

func createModifyStr(actionEntry actionEntry) (string, error) {
	var buffer bytes.Buffer
	subActions := make(map[subAction]string)
	subActions[subActionModifyAdd] = "add"
	subActions[subActionModifyDelete] = "delete"
	subActions[subActionModifyReplace] = "replace"
	for idx, subActionList := range actionEntry.SubActionAttrs {
		for subAction, attrList := range subActionList {
			if subAction == subActionNone {
				return "", errors.New(("Invalid Subaction subActionNone for action actionModify"))
			}
			idxInner := 0
			for _, attr := range slices.Sorted(maps.Keys(attrList)) {
				vals := attrList[attr]
				if idxInner != 0 || idx != 0 {
					buffer.WriteString("-\n")
				}

				idxInner++

				for idxInnerV, val := range vals {
					if (subActions[subAction] != "add" && subActions[subAction] != "replace") && idxInnerV != 0 {
						buffer.WriteString("-\n")
					}
					if (subActions[subAction] != "add" && subActions[subAction] != "replace") || idxInnerV == 0 {
						buffer.WriteString(subActions[subAction] + ": " + attr + "\n")
					}
					buffer.WriteString(attr + ": " + val + "\n")
				}
			}
		}
	}
	return buffer.String(), nil
}

func writeLdif(queue <-chan actionEntry, writer *bytes.Buffer, delWriter *bytes.Buffer, wg *sync.WaitGroup, err *error) {
	defer wg.Done()
	for actionEntry := range queue {
		if *err != nil {
			continue
		}
		switch actionEntry.Action {
		case actionAdd:
			writer.WriteString(actionEntry.Dn + "\n") //dn
			writer.WriteString("changetype: add\n")
			attrList := actionEntry.SubActionAttrs[0][subActionNone]
			for _, attr := range slices.Sorted(maps.Keys(attrList)) {
				vals := attrList[attr]
				for _, val := range vals {
					writer.WriteString(attr + ": " + val + "\n")
				}
			}
			writer.WriteString("\n")
		case actionDelete:
			delWriter.WriteString(actionEntry.Dn + "\n") //dn
			delWriter.WriteString("changetype: delete\n\n")
		case actionModify:
			writer.WriteString(actionEntry.Dn + "\n") //dn
			writer.WriteString("changetype: modify\n")
			modifyStr, modifyErr := createModifyStr(actionEntry)
			if modifyErr != nil {
				*err = modifyErr
				continue
			}
			writer.WriteString(modifyStr + "\n")
		default:
			*err = errors.New(fmt.Sprintf("Unexpected LDIF action value: %d", actionEntry.Action))
			continue
		}
	}
}
