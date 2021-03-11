package service

import (
	"fmt"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
)

func TestPartitionBenes(t *testing.T) {
	var benes []*models.CCLFBeneficiary
	for i := 0; i < 15; i++ {
		benes = append(benes, &models.CCLFBeneficiary{MBI: testUtils.RandomMBI(t)})
	}

	tests := []struct {
		name        string
		size        uint
		expNumParts int
	}{
		{"InputEqualParts", 3, 5},
		{"InputNotEqual", 4, 4},
		{"SizeLargerThanInput", 30, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				total    []*models.CCLFBeneficiary
				part     []*models.CCLFBeneficiary
				start    = benes
				numParts int
			)
			for {
				part, start = partitionBenes(start, tt.size)
				if len(part) == 0 {
					break
				}
				numParts++
				total = append(total, part...)
			}
			assert.Equal(t, benes, total)
		})
	}
}
