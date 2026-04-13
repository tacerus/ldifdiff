package ldifdiff

import (
	"os"
	"strings"
	"testing"

	"github.com/go-ldap/ldif"
	"github.com/google/go-cmp/cmp"
)

func TestDiff(t *testing.T) {
	ldif, err := Diff(testSourceStr, testTargetStr, nil, nil)
	if diff := cmp.Diff(testResultStr, ldif); diff != "" {
		t.Error(diff)
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}

	ldif, err = Diff(testSourceStr, testTargetStr, testIgnoreAttr, nil)
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}
	for _, ignoredAttr := range testIgnoreAttr {
		if strings.Contains(ldif, ignoredAttr+":") {
			t.Error("Attribute", ignoredAttr, "not ignored")
		}
	}
}

func TestDiffMv(t *testing.T) {
	ldif, err := Diff(testSourceMvStr, testTargetMvStr, nil, testStrictAttr)
	if diff := cmp.Diff(testResultMvStr, ldif); diff != "" {
		t.Error("Diff:\n" + diff + "\nExpected: " + testResultMvStr + " Got: " + ldif)
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}
}

func TestDiffFromFiles(t *testing.T) {
	ldif, err := DiffFromFiles(testSourceLdifFile, testTargetLdifFile, nil, nil)
	if diff := cmp.Diff(testResultStr, ldif); diff != "" {
		t.Error(diff)
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}

	ldif, err = DiffFromFiles(testSourceLdifFile, testTargetLdifFile, testIgnoreAttr, nil)
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}
	for _, ignoredAttr := range testIgnoreAttr {
		if strings.Contains(ldif, ignoredAttr+":") {
			t.Error("Attribute", ignoredAttr, "not ignored")
		}
	}
}

func TestListDiffDn(t *testing.T) {
	dns, err := ListDiffDn(testSourceStr, testTargetStr, nil, nil)
	joinedLines := strings.Join(dns, "\n") + "\n\n"
	if diff := cmp.Diff(testResultDnStr, joinedLines); diff != "" {
		t.Error("1 Diff:\n" + diff + "\nExpected:\n[" + testResultDnStr + "]\nGot:\n[" + joinedLines + "]\n")
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}

	dns, err = ListDiffDn(testSourceStr, testTargetStr, testIgnoreAttrDn, nil)
	// TODO: why does this have two trailing newlines?
	joinedLines = strings.Join(dns, "\n") + "\n\n"
	if diff := cmp.Diff(testResultDnIgnoreAttrStr, joinedLines); diff != "" {
		t.Error("2 Diff:\n" + diff + "\nExpected:\n[" + testResultDnIgnoreAttrStr + "]\nGot:\n[" + joinedLines + "]\n")
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
}

func TestListDiffDnFromFiles(t *testing.T) {
	dns, err := ListDiffDnFromFiles(testSourceLdifFile, testTargetLdifFile, nil, nil)
	joinedLines := strings.Join(dns, "\n") + "\n\n"
	if diff := cmp.Diff(testResultDnStr, joinedLines); diff != "" {
		t.Error("Diff:\n" + diff + "\nExpected:\n[" + testResultDnStr + "]\nGot:\n[" + joinedLines + "]\n")
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
}

func TestDiffFromFilesBig(t *testing.T) {
	if os.Getenv(testBigFilesEnv) != testBigFilesEnvValue {
		t.Skip("Skipping big files test")
	}
	ldif, err := DiffFromFiles(testSourceLdifFileBig, testTargetLdifFileBig, nil, nil)
	if ldif != testResultStr {
		t.Error("Expected:\n[" + testResultStrBig + "]\nGot:\n[" + ldif + "]\n")
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}
}

func TestDiffBig(t *testing.T) {
	if os.Getenv(testBigFilesEnv) != testBigFilesEnvValue {
		t.Skip("Skipping big files test")
	}
	ldif, err := Diff(testSourceStrBig, testTargetStrBig, nil, nil)
	if ldif != testResultStr {
		t.Error("Expected:\n" + testResultStrBig + "Got:\n" + ldif)
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}

	ldif, err = Diff(testSourceStrBig, testTargetStrBig, testIgnoreAttr, nil)
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}
	for _, ignoredAttr := range testIgnoreAttr {
		if strings.Contains(ldif, ignoredAttr+":") {
			t.Error("Attribute", ignoredAttr, "not ignored")
		}
	}
}

func TestCompare(t *testing.T) {
	source, _ := importLdifFile(testSourceLdifFile, nil, nil)
	target, _ := importLdifFile(testTargetLdifFile, nil, nil)

	ldifObj, err := compareLdif(&source, &target, nil, nil)
	if err != nil {
		t.Fatal("Expected ldifObj, got error: ", err)
	}

	ldifStr, err := ldif.Marshal(ldifObj)
	if err != nil {
		t.Fatal("Expected ldifStr, got error: ", err)
	}

	if diff := cmp.Diff(ldifStr, testResultStr); diff != "" {
		t.Error("Diff:\n" + diff + "\nExpected:\n[" + testResultStr + "]\nGot:\n[" + ldifStr + "]\n")
	}
}
