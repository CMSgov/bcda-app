package cclf

import (
	"bufio"
	"context"
	"math/rand"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
)

func TestNext(t *testing.T) {
	mbi1, mbi2, mbi3 := testUtils.RandomMBI(t), testUtils.RandomMBI(t), testUtils.RandomMBI(t)
	// Create set of duplicate MBIs to verify that we skip them when invoking the next call
	mbis := []string{mbi1, mbi1, mbi3, mbi2, mbi2}
	scanner := bufio.NewScanner(strings.NewReader(strings.Join(mbis, "\n")))

	importer := &cclf8Importer{processedMBIs: make(map[string]struct{}), scanner: scanner}
	for _, expected := range []string{mbi1, mbi3, mbi2} {
		assert.True(t, importer.Next())
		assert.Equal(t, expected, string(importer.scanner.Bytes()))
	}

	assert.False(t, importer.Next())
}

func TestValues(t *testing.T) {
	mbi := testUtils.RandomMBI(t)
	scanner := bufio.NewScanner(strings.NewReader(mbi))

	importer := &cclf8Importer{ctx: context.Background(), cclfFileID: uint(rand.Int31()),
		scanner: scanner, processedMBIs: make(map[string]struct{}),
		reportInterval: 1}
	assert.True(t, importer.Next())

	values, err := importer.Values()
	assert.NoError(t, err)
	assert.Len(t, values, 2)
}

func TestErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	importer := &cclf8Importer{ctx: ctx}
	assert.EqualError(t, importer.Err(), "context canceled")

	importer = &cclf8Importer{ctx: context.Background()}
	assert.NoError(t, importer.Err())
}
