package ldifdiff

import (
	"sync"
	"testing"

	"github.com/go-ldap/ldif"
	"github.com/google/go-cmp/cmp"
)

func TestBuildLdif(t *testing.T) {
	var wg sync.WaitGroup
	var err error
	result := &ldif.LDIF{}
	queue := make(chan actionEntry)
	wg.Add(2)
	go func(queue chan actionEntry) {
		queue <- testActionEntryTestData.Add
		queue <- testActionEntryTestData.Modify
		close(queue)
		wg.Done()
	}(queue)

	go buildLdif(queue, result, &wg, &err)
	wg.Wait()

	if err != nil {
		t.Error("Error not expected, got: ", err)
	}

	ldifStr, err := ldif.Marshal(result)
	if err != nil {
		t.Fatal("Error not expected, got: ", err)
	}

	if diff := cmp.Diff(ldifStr, testModifyStr); diff != "" {
		t.Error("Diff:\n" + diff + "\nExpected:\n[" + testModifyStr + "]\nGot:\n[" + ldifStr + "]\n")
	}
}
