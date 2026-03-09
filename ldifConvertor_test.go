package ldifdiff

import (
	"bytes"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteLdif(t *testing.T) {
	var buffer bytes.Buffer
	var wg sync.WaitGroup
	var err error
	queue := make(chan actionEntry)
	wg.Add(2)
	go func(queue chan actionEntry) {
		queue <- testActionEntryTestData.Add
		queue <- testActionEntryTestData.Modify
		close(queue)
		wg.Done()
	}(queue)

	go writeLdif(queue, &buffer, &bytes.Buffer{}, &wg, &err)
	wg.Wait()

	if err != nil {
		t.Error("Error not expected, got: ", err)
	}
	ldif := buffer.String()
	if diff := cmp.Diff(ldif, testModifyStr); diff != "" {
		t.Error("Diff:\n" + diff + "\nExpected:\n[" + testModifyStr + "]\nGot:\n[" + ldif + "]\n")
	}
}
