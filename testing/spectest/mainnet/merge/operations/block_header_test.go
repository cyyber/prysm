package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/merge/operations"
)

func TestMainnet_Merge_Operations_BlockHeader(t *testing.T) {
	operations.RunBlockHeaderTest(t, "mainnet")
}