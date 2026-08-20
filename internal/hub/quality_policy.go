package hub

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

func ValidateBrothBatch(batch BrothBatch, lots map[string]IngredientLot) error {
	if len(batch.BoneLotIDs) == 0 || batch.WaterLiters <= 0 || batch.BoneKilograms <= 0 {
		return fmt.Errorf("%w: broth quantities and bone lots are required", ErrInvalidIngredient)
	}
	if batch.BoneKilograms*8 < batch.WaterLiters {
		return fmt.Errorf("%w: bone-to-water ratio is below the quality guide", ErrInvalidIngredient)
	}
	for _, id := range batch.BoneLotIDs {
		lot, ok := lots[id]
		if !ok || !slices.Contains([]string{"yak-bone", "tibetan-sheep-bone"}, lot.IngredientCode) {
			return fmt.Errorf("%w: broth bone lot %s is missing or not approved", ErrInvalidIngredient, id)
		}
	}
	return nil
}

func ValidateChiliOil(batch ChiliOilBatch, chili IngredientLot, oil IngredientLot) error {
	if batch.ChiliLotID != chili.ID || batch.OilLotID != oil.ID {
		return fmt.Errorf("%w: chili oil traceability does not match supplied lots", ErrInvalidIngredient)
	}
	if !strings.EqualFold(strings.TrimSpace(batch.Cultivar), "循化线辣椒") || chili.OriginRegion != "循化" {
		return fmt.Errorf("%w: chili cultivar or origin is not approved", ErrInvalidIngredient)
	}
	if batch.ProducedAt.Before(chili.ProducedAt) || !batch.ProducedAt.Before(chili.ExpiresAt) {
		return fmt.Errorf("%w: chili lot is not valid at production time", ErrInvalidIngredient)
	}
	return nil
}

func ValidateSupplierCertificate(lot IngredientLot, certificate SupplierCertificate, at time.Time) error {
	if lot.SupplierID != certificate.SupplierID || lot.CertificateID != certificate.ID {
		return fmt.Errorf("%w: supplier certificate does not match lot", ErrInvalidIngredient)
	}
	if certificate.RevokedAt != nil && !at.Before(*certificate.RevokedAt) {
		return fmt.Errorf("%w: supplier certificate was revoked", ErrInvalidIngredient)
	}
	if at.Before(certificate.EffectiveAt) || !at.Before(certificate.ExpiresAt) {
		return fmt.Errorf("%w: supplier certificate is outside its validity window", ErrInvalidIngredient)
	}
	if !slices.Contains(certificate.Scope, lot.IngredientCode) {
		return fmt.Errorf("%w: certificate does not cover ingredient", ErrInvalidIngredient)
	}
	return nil
}

func ValidateColdChain(lot IngredientLot) error {
	if len(lot.Temperatures) == 0 || lot.MinTemperature > lot.MaxTemperature {
		return fmt.Errorf("%w: cold-chain readings or limits are invalid", ErrInvalidIngredient)
	}
	for _, reading := range lot.Temperatures {
		if reading < lot.MinTemperature || reading > lot.MaxTemperature {
			return fmt.Errorf("%w: temperature %.1f is outside %.1f..%.1f", ErrInvalidIngredient, reading, lot.MinTemperature, lot.MaxTemperature)
		}
	}
	return nil
}

func CalculateInspection(sections []InspectionSection, passScore int) (int, bool, error) {
	if len(sections) == 0 || passScore <= 0 || passScore > 100 {
		return 0, false, fmt.Errorf("%w: invalid inspection input", ErrStoreNotCompliant)
	}
	total := 0
	for _, section := range sections {
		if section.Score < 0 || section.Score > 100 {
			return 0, false, fmt.Errorf("%w: section %s score is invalid", ErrStoreNotCompliant, section.Code)
		}
		if section.Required && len(section.Evidence) == 0 {
			return 0, false, fmt.Errorf("%w: required section %s lacks evidence", ErrStoreNotCompliant, section.Code)
		}
		total += section.Score
	}
	score := total / len(sections)
	return score, score >= passScore, nil
}

func CorrectiveDeadline(severity string, observedAt time.Time) (time.Time, error) {
	var duration time.Duration
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		duration = 24 * time.Hour
	case "major":
		duration = 7 * 24 * time.Hour
	case "minor":
		duration = 30 * 24 * time.Hour
	default:
		return time.Time{}, fmt.Errorf("%w: unknown corrective severity", ErrStoreNotCompliant)
	}
	return observedAt.UTC().Add(duration), nil
}

func ValidateAppeal(appeal Appeal, inspection Inspection, now time.Time) error {
	if appeal.InspectionID != inspection.ID || appeal.StoreID != inspection.StoreID {
		return fmt.Errorf("%w: appeal does not match inspection", ErrAppealExpired)
	}
	if strings.TrimSpace(appeal.Reason) == "" || len(appeal.Evidence) == 0 {
		return fmt.Errorf("%w: reason and evidence are required", ErrAppealExpired)
	}
	if now.After(inspection.CompletedAt.Add(10 * 24 * time.Hour)) {
		return ErrAppealExpired
	}
	return nil
}
