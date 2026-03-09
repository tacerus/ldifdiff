package ldifdiff

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

// * Test data */
const testBigFilesEnv = "LDIFDIFF_BIGFILES"
const testBigFilesEnvValue = "1"
const testDn = "some_dn,ou=aAccounts,dc=domain,dc=ext"
const testSourceLdifFile = "t/source.ldif"
const testSourceLdifMvFile = "t/source_mv.ldif"
const testTargetLdifFile = "t/target.ldif"
const testTargetLdifMvFile = "t/target_mv.ldif"
const testResultLdifFile = "t/result.ldif"
const testResultLdifMvFile = "t/result_mv.ldif"
const testResultDnFile = "t/result_dn"
const testResultDnIgnoreAttrFile = "t/result_dn_ignore_attr"
const testInvalidLineContLdifFile = "t/invalid_line_continuation.ldif"
const testInvalidNoDnLdifFile = "t/invalid_no_dn.ldif"
const testSourceLdifFileBig = "t/_source_big.ldif"
const testTargetLdifFileBig = "t/_target_big.ldif"
const testResultLdifFileBig = "t/_result_big.ldif"
const testModifyLdifFile = "t/modify.ldif"

var testSourceStr = testGetLdifeStr(testSourceLdifFile, false)
var testSourceMvStr = testGetLdifeStr(testSourceLdifMvFile, false)
var testSourceNrEntries = testGetNrOfEntries(testSourceStr)
var testSourceMvNrEntries = testGetNrOfEntries(testSourceMvStr)
var testTargetStr = testGetLdifeStr(testTargetLdifFile, false)
var testTargetMvStr = testGetLdifeStr(testTargetLdifMvFile, false)
var testResultStr = testGetLdifeStr(testResultLdifFile, false)
var testResultMvStr = testGetLdifeStr(testResultLdifMvFile, false)
var testResultDnStr = testGetLdifeStr(testResultDnFile, false)
var testResultDnIgnoreAttrStr = testGetLdifeStr(testResultDnIgnoreAttrFile, false)
var testInvalidLineContStr = testGetLdifeStr(testInvalidLineContLdifFile, false)
var testInvalidNoDnStr = testGetLdifeStr(testInvalidNoDnLdifFile, false)
var testSourceStrBig = testGetLdifeStr(testSourceLdifFileBig, true)
var testTargetStrBig = testGetLdifeStr(testTargetLdifFileBig, true)
var testResultStrBig = testGetLdifeStr(testResultLdifFileBig, true)
var testModifyStr = testGetLdifeStr(testModifyLdifFile, false)
var testIgnoreAttr = []string{"sambaSID", "eduPersonEntitlement"}
var testIgnoreAttrDn = []string{"sambaSID", "eduPersonEntitlement", "mail"}
var testStrictAttr = []string{"mail"}
var testAttrList = entry{"mail": []string{"auth2@domain.ext"}, "phone": []string{"+32364564645"}}
var testAttrListModifyReplace = entry{"mail": []string{"auth2@domain.ext"}}
var testActionEntryTestData = testGetActionEntryMap()

/* Helper functions and types */
type TestActionEntryData struct {
	Add, Delete, Modify, ModifyOnlyAdd,
	ModifyOnlyDelete, ModifyOnlyReplace,
	ModifyNone, ModifyReplaceAttributes actionEntry
}

func addEntry(e entry, req *ldap.AddRequest) {
	for a, v := range e {
		req.Attribute(a, v)
	}
}

func modEntry(add entry, del entry, rep entry, req *ldap.ModifyRequest) {
	for a, v := range add {
		req.Add(a, v)
	}
	for a, v := range del {
		req.Delete(a, v)
	}
	for a, v := range rep {
		req.Replace(a, v)
	}
}

func testGetActionEntryMap() TestActionEntryData {
	reqAdd := ldap.NewAddRequest(testDn, nil)
	addEntry(testAttrList, reqAdd)
	reqDel := ldap.NewDelRequest(testDn, nil)
	reqMod := ldap.NewModifyRequest(testDn, nil)
	modEntry(testAttrList, testAttrList, testAttrListModifyReplace, reqMod)
	return TestActionEntryData{
		Add: actionEntry{
			Dn:  testDn,
			Add: []*ldap.AddRequest{reqAdd},
		},
		Delete: actionEntry{
			Dn:  testDn,
			Del: []*ldap.DelRequest{reqDel},
		},
		Modify: actionEntry{
			Dn:  testDn,
			Mod: []*ldap.ModifyRequest{reqMod},
		},
	}
}

func testGetLdifeStr(file string, big bool) string {
	if big && os.Getenv(testBigFilesEnv) != testBigFilesEnvValue {
		return ""
	}
	bytes, _ := ioutil.ReadFile(file)
	return string(bytes) + "\n"
}

func testGetNrOfEntries(ldifStr string) int {
	var counter int
	for _, line := range strings.Split(ldifStr, "\n") {
		if strings.HasPrefix(line, "dn:") {
			counter++
		}
	}
	return counter
}
