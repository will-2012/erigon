package eth2

import (
	"encoding/binary"

	libcommon "github.com/ledgerwatch/erigon-lib/common"

	"github.com/ledgerwatch/erigon/cl/abstract"
	"github.com/ledgerwatch/erigon/cl/cltypes"
	"github.com/ledgerwatch/erigon/cl/utils"
)

const VERSIONED_HASH_VERSION_KZG byte = byte(1)

func kzgCommitmentToVersionedHash(kzgCommitment *cltypes.KZGCommitment) (libcommon.Hash, error) {
	versionedHash := [32]byte{}
	kzgCommitmentHash := utils.Sha256(kzgCommitment[:])

	buf := append([]byte{}, VERSIONED_HASH_VERSION_KZG)
	buf = append(buf, kzgCommitmentHash[1:]...)
	copy(versionedHash[:], buf)

	return versionedHash, nil
}

func computeSigningRootEpoch(epoch uint64, domain []byte) (libcommon.Hash, error) {
	b := make([]byte, 32)
	binary.LittleEndian.PutUint64(b, epoch)
	return utils.Sha256(b, domain), nil
}

// transitionSlot is called each time there is a new slot to process
func transitionSlot(s abstract.BeaconState) error {
	slot := s.Slot()
	previousStateRoot := s.PreviousStateRoot()
	var err error
	if previousStateRoot == (libcommon.Hash{}) {
		previousStateRoot, err = s.HashSSZ()
		if err != nil {
			return err
		}
	}

	beaconConfig := s.BeaconConfig()

	s.SetStateRootAt(int(slot%beaconConfig.SlotsPerHistoricalRoot), previousStateRoot)

	latestBlockHeader := s.LatestBlockHeader()
	if latestBlockHeader.Root == [32]byte{} {
		latestBlockHeader.Root = previousStateRoot
		s.SetLatestBlockHeader(&latestBlockHeader)
	}
	blockHeader := s.LatestBlockHeader()

	previousBlockRoot, err := (&blockHeader).HashSSZ()
	if err != nil {
		return err
	}
	s.SetBlockRootAt(int(slot%beaconConfig.SlotsPerHistoricalRoot), previousBlockRoot)
	return nil
}
