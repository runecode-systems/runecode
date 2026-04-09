package artifacts

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const promotionActionJustification = "promotion approval request"

func BuildPromotionActionRequest(req PromotionRequest) (policyengine.ActionRequest, error) {
	if !isValidDigest(req.UnapprovedDigest) {
		return policyengine.ActionRequest{}, fmt.Errorf("invalid source digest %q", req.UnapprovedDigest)
	}
	sourceDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimPrefix(req.UnapprovedDigest, "sha256:")}
	if _, err := sourceDigest.Identity(); err != nil {
		return policyengine.ActionRequest{}, err
	}
	return policyengine.NewPromotionAction(policyengine.PromotionActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID:           policyengine.ActionKindPromotion,
			RelevantArtifactHashes: []trustpolicy.Digest{sourceDigest},
			Actor: policyengine.ActionActor{
				ActorKind:  "daemon",
				RoleFamily: "workspace",
				RoleKind:   "workspace-edit",
			},
		},
		PromotionKind:        "excerpt",
		SourceArtifactHash:   sourceDigest,
		TargetDataClass:      string(DataClassApprovedFileExcerpts),
		Justification:        promotionActionJustification,
		RepoPath:             req.RepoPath,
		Commit:               req.Commit,
		ExtractorToolVersion: req.ExtractorToolVersion,
		Approver:             req.Approver,
	}), nil
}

func CanonicalPromotionActionRequestHash(req PromotionRequest) (string, error) {
	action, err := BuildPromotionActionRequest(req)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalActionRequestHash(action)
}

func promotionActionRequestHash(req PromotionRequest) (string, error) {
	return CanonicalPromotionActionRequestHash(req)
}
