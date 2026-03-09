package ldifdiff

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestImportLdifFile(t *testing.T) {
	entries, err := importLdifFile(testSourceLdifFile, testIgnoreAttr, nil)
	okLdifTests(t, entries, testIgnoreAttr, nil, err)

	entries, err = importLdifFile(testSourceLdifMvFile, testIgnoreAttr, testStrictAttr)
	okLdifTests(t, entries, testIgnoreAttr, testStrictAttr, err)

	_, err = importLdifFile(testInvalidLineContLdifFile, testIgnoreAttr, nil)
	invalidLineCont(t, err)

	_, err = importLdifFile(testInvalidNoDnLdifFile, testIgnoreAttr, nil)
	invalidNoDn(t, err)

}

func TestConvertLdifStr(t *testing.T) {
	entries, err := convertLdifStr(testSourceStr, testIgnoreAttr, nil)
	okLdifTests(t, entries, testIgnoreAttr, nil, err)

	entries, err = convertLdifStr(testSourceMvStr, nil, testStrictAttr)
	okLdifTests(t, entries, testIgnoreAttr, testStrictAttr, err)

	_, err = convertLdifStr(testInvalidLineContStr, testIgnoreAttr, nil)
	invalidLineCont(t, err)

	entries, err = convertLdifStr(testInvalidNoDnStr, testIgnoreAttr, nil)
	invalidNoDn(t, err)
}

func TestImportLdifFileBig(t *testing.T) {
	if os.Getenv(testBigFilesEnv) != testBigFilesEnvValue {
		t.Skip("Skipping big files test")
	}
	entries, err := importLdifFile(testSourceLdifFileBig, testIgnoreAttr, nil)
	okLdifTests(t, entries, testIgnoreAttr, nil, err)
}

func TestConvertLdifStrBig(t *testing.T) {
	if os.Getenv(testBigFilesEnv) != testBigFilesEnvValue {
		t.Skip("Skipping big files test")
	}
	entries, err := convertLdifStr(testSourceStrBig, testIgnoreAttr, nil)
	okLdifTests(t, entries, testIgnoreAttr, nil, err)
}

/* Helper test evaluation */
func okLdifTests(t *testing.T, entries entries, ignoreAttr []string, strictAttr []string, err error) {
	if err != nil {
		t.Error("Expected values, got error: ", err)
	}
	n := testSourceNrEntries
	if len(strictAttr) > 0 {
		n = testSourceMvNrEntries
	}
	if len(entries) != n {
		t.Error("Expected", n, "entries, got", strconv.Itoa(len(entries)))
	}
	for dn, attributes := range entries {
		if !strings.HasPrefix(dn, "dn:") {
			t.Error("Invalid dn:", dn)
		}
		for attr, vals := range attributes {
			if strings.HasPrefix(attr, "#") {
				t.Error("Invalid comment:", attr)
			}
			for _, val := range vals {
				if strings.HasPrefix(val, " ") {
					t.Error("Line continuation not correctly appended:", attr, val)
				}
			}
		}
	}
	if len(ignoreAttr) > 0 {
		for _, attributes := range entries {
			for attr, _ := range attributes {
				for _, ignoredAttr := range ignoreAttr {
					if strings.HasPrefix(attr, ignoredAttr+":") {
						t.Error("Attributed not ignored as requested:",
							ignoredAttr, attr)
					}
				}
			}
		}
	}
	for _, attributes := range entries {
		for attr, vals := range attributes {
			// TODO: consider checking ordered equality for strict attrs here as well?
			if !slices.Contains(strictAttr, attr) {
				sortedVals := slices.Sorted(slices.Values(vals))
				if !slices.Equal(vals, sortedVals) {
					t.Errorf("Attribute values not sorted: %s.\nHave: %v\nWant: %v\n",
						attr, vals, sortedVals)
				}
			}
		}
	}
}

func invalidLineCont(t *testing.T, err error) {
	if err == nil {
		t.Error("Error expected (line continuation), but none received")
	}
}

func invalidNoDn(t *testing.T, err error) {
	if err == nil {
		t.Error("Error expected (no dn), but none received")
	}
}
