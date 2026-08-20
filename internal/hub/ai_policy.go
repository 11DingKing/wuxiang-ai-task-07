package hub

import (
	"fmt"
	"strings"
	"time"
)

var allowedASEANCountries = map[string]struct{}{"VN": {}, "TH": {}, "ID": {}, "MY": {}, "SG": {}, "KH": {}, "PH": {}, "LA": {}, "MM": {}, "BN": {}}

func validateApplicationInput(application EnterpriseApplication) error {
	if application.ID == "" || application.TenantID == "" || application.ApplicantID == "" || strings.TrimSpace(application.CompanyName) == "" {
		return fmt.Errorf("%w: identity", ErrAIValidation)
	}
	if _, ok := allowedASEANCountries[strings.ToUpper(strings.TrimSpace(application.ASEANCountry))]; !ok {
		return fmt.Errorf("%w: ASEAN country", ErrAIValidation)
	}
	if application.RequestedPetaops <= 0 || application.RequestedPetaops > 5000 {
		return fmt.Errorf("%w: requested compute", ErrAIValidation)
	}
	if len(application.Materials) == 0 {
		return fmt.Errorf("%w: materials", ErrAIValidation)
	}
	return nil
}

func validApplicationTransition(from, to AIApplicationStatus) bool {
	switch from {
	case ApplicationPending:
		return to == ApplicationApproved || to == ApplicationReturned || to == ApplicationCancelled
	case ApplicationReturned:
		return to == ApplicationPending || to == ApplicationCancelled
	case ApplicationApproved:
		return to == ApplicationActive || to == ApplicationCancelled
	case ApplicationActive:
		return to == ApplicationClosed
	case ApplicationClosed, ApplicationCancelled:
		return false
	default:
		return false
	}
}

func normalizeCountry(country string) string { return strings.ToUpper(strings.TrimSpace(country)) }

func offerAvailable(offer ServiceOffer, country string, at time.Time) bool {
	if at.Before(offer.EffectiveFrom) || !at.Before(offer.EffectiveTo) {
		return false
	}
	if offer.Used >= offer.Capacity {
		return false
	}
	for _, candidate := range offer.Countries {
		if normalizeCountry(candidate) == normalizeCountry(country) {
			return true
		}
	}
	return false
}

func sceneCanApprove(scene SceneSubmission) error {
	if scene.Status != "submitted" {
		return fmt.Errorf("%w: scene status %s", ErrAIInvalidState, scene.Status)
	}
	if strings.TrimSpace(scene.OwnerID) == "" || len(scene.Evidence) < 2 {
		return fmt.Errorf("%w: scene evidence", ErrAIValidation)
	}
	return nil
}

func computeAvailable(cluster ComputeCluster, requested int) bool {
	return requested > 0 && cluster.TotalPetaops-cluster.Allocated >= requested
}
