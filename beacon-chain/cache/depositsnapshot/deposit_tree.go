package depositsnapshot

import (
	"crypto/sha256"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
)

var (
	ErrEmptyExecutionBlock = errors.New("empty execution block")
	ErrInvalidSnapshotRoot = errors.New("snapshot root is invalid")
)

// DepositTree is the Merkle tree representation of deposits.
type DepositTree struct {
	tree                    MerkleTreeNode
	mixInLength             uint64 // number of deposits in the tree, reference implementation calls this mix_in_length.
	finalizedExecutionBlock ExecutionBlock
}

type ExecutionBlock struct {
	Hash  [32]byte
	Depth uint64
}

// New creates an empty deposit tree.
func New() *DepositTree {
	var leaves [][32]byte
	merkle := create(leaves, DepositContractDepth)
	return &DepositTree{
		tree:                    merkle,
		mixInLength:             0,
		finalizedExecutionBlock: ExecutionBlock{},
	}
}

// getSnapshot returns a deposit tree snapshot.
func (d *DepositTree) getSnapshot() (DepositTreeSnapshot, error) {
	if d.finalizedExecutionBlock == (ExecutionBlock{}) {
		return DepositTreeSnapshot{}, ErrEmptyExecutionBlock
	}
	var finalized [][32]byte
	depositCount, _ := d.tree.GetFinalized(finalized)
	return fromTreeParts(finalized, depositCount, d.finalizedExecutionBlock), nil
}

// fromSnapshot returns a deposit tree from a deposit tree snapshot.
func fromSnapshot(snapshot DepositTreeSnapshot) (DepositTree, error) {
	if snapshot.depositRoot != snapshot.CalculateRoot() {
		return DepositTree{}, ErrInvalidSnapshotRoot
	}
	finalizedExecutionBlock := ExecutionBlock{
		Hash:  snapshot.executionBlockHash,
		Depth: snapshot.executionBlockHeight,
	}
	tree := fromSnapshotParts(snapshot.finalized, snapshot.depositCount, DepositContractDepth)
	return DepositTree{
		tree:                    tree,
		mixInLength:             snapshot.depositCount,
		finalizedExecutionBlock: finalizedExecutionBlock,
	}, nil
}

func (d *DepositTree) finalize(eth1data eth.Eth1Data, executionBlockHeight uint64) {
	d.finalizedExecutionBlock = ExecutionBlock{
		Hash:  eth1data.BlockHash,
		Depth: executionBlockHeight,
	}
	d.tree.Finalize(eth1data.DepositCount, DepositContractDepth)
}

func (d *DepositTree) getProof(index uint64) ([32]byte, [][32]byte, error) {
	if d.mixInLength <= 0 {
		return [32]byte{}, nil, nil
	}
	finalizedDeposits, _ := d.tree.GetFinalized([][32]byte{})
	if index <= (finalizedDeposits - 1) {
		return [32]byte{}, nil, nil
	}
	leaf, proof := generateProof(d.tree, index, DepositContractDepth)
	proof = append(proof, bytesutil.Uint64ToBytesLittleEndian32(d.mixInLength))
	return leaf, proof, nil
}

// getRoot returns the root of the deposit tree.
func (d *DepositTree) getRoot() [32]byte {
	root := d.tree.GetRoot()
	return sha256.Sum256(append(root[:], bytesutil.Uint64ToBytesLittleEndian32(d.mixInLength)...))
}

// pushLeaf adds a new deposit to the tree.
func (d *DepositTree) pushLeaf(leaf [32]byte) error {
	var err error
	d.mixInLength += 1
	d.tree, err = d.tree.PushLeaf(leaf, DepositContractDepth)
	if err != nil {
		return err
	}
	return nil
}