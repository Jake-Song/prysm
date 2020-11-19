package slashingprotection

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// IsSlashableBlock determines if an incoming block is slashable
// according to local protection and remote protection (if enabled). Then, if the block
// successfully passes checks, we update our local proposals history accordingly.
func (s *Service) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, signingRoot [32]byte,
) (bool, error) {
	if block == nil || block.Block == nil {
		return false, errors.New("received nil block")
	}
	// Check if the block is slashable by local and remote slashing protection
	existingSigningRoot, err := s.validatorDB.ProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot)
	if err != nil {
		return false, errors.Wrap(err, "failed to get proposal history")
	}
	// Check if we are performing a double block proposal.
	if !bytes.Equal(existingSigningRoot, params.BeaconConfig().ZeroHash[:]) {
		localSlashableProposalsTotal.Inc()
		return true, nil
	}
	// If the block is not slashable, we perform an immediate write to our DB.
	if err := s.validatorDB.SaveProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot, signingRoot[:]); err != nil {
		return false, errors.Wrap(err, "failed to save updated proposal history")
	}
	if s.remoteProtector != nil {
		slashable, err := s.remoteProtector.IsSlashableBlock(ctx, block, pubKey, signingRoot)
		if err != nil {
			return false, errors.Wrap(err, "failed to get proposal history")
		}
		remoteSlashableProposalsTotal.Inc()
		return slashable, nil
	}
	return false, nil
}