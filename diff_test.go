package ldifdiff

import (
	"os"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	ldif, err := Diff(testSourceStr, testTargetStr, nil, nil)
	if ldif != testResultStr {
		t.Error("Expected:\n[" + testResultStr + "]\nGot:\n[" + ldif + "]\n")
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
	if ldif != testResultMvStr {
		t.Error("Expected: " + testResultMvStr + " Got: " + ldif)
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
	if ldif != testResultStr {
		t.Error("Expected:\n[" + testResultStr + "]\nGot:\n[" + ldif + "]\n")
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
	joinedLines := strings.Join(dns, "\n") + "\n"
	if joinedLines != testResultDnStr {
		t.Error("Expected:\n[" + testResultDnStr + "]\nGot:\n[" + joinedLines + "]\n")
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}

	dns, err = ListDiffDn(testSourceStr, testTargetStr, testIgnoreAttrDn, nil)
	joinedLines = strings.Join(dns, "\n") + "\n"
	if joinedLines != testResultDnIgnoreAttrStr {
		t.Error("Expected:\n[" + testResultDnIgnoreAttrStr + "]\nGot:\n[" + joinedLines + "]\n")
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
}

func TestListDiffDnFromFiles(t *testing.T) {
	dns, err := ListDiffDnFromFiles(testSourceLdifFile, testTargetLdifFile, nil, nil)
	joinedLines := strings.Join(dns, "\n") + "\n"
	if joinedLines != testResultDnStr {
		t.Error("Expected:\n[" + testResultDnStr + "]\nGot:\n[" + joinedLines + "]\n")
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
	ldif, err := compare(&source, &target, nil, nil)
	if ldif != testResultStr {
		t.Error("Expected:\n[" + testResultStr + "]\nGot:\n[" + ldif + "]\n")
	}
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	if ldif == "" {
		t.Error("Expected changes, got an empty modifyStr")
	}
}
