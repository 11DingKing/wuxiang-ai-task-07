package hub

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

func ValidateLicenseAt(license BrandLicense, store StoreProfile, at time.Time) error {
	if license.ID == "" || license.StoreID != store.ID || license.OwnerID != store.OwnerID {
		return fmt.Errorf("%w: license identity does not match store", ErrInvalidLicense)
	}
	if license.Status != LicenseActive || at.Before(license.EffectiveFrom) || !at.Before(license.EffectiveTo) {
		return fmt.Errorf("%w: license is not active at requested time", ErrInvalidLicense)
	}
	if !slices.Contains(license.RegionCodes, store.CityCode) {
		return fmt.Errorf("%w: city %s is outside licensed regions", ErrInvalidLicense, store.CityCode)
	}
	return nil
}

func ValidateThreeStoreModel(store StoreProfile) error {
	missing := make([]string, 0, 3)
	if !store.DiningEnabled {
		missing = append(missing, "noodle dining")
	}
	if !store.SpecialtyRetailEnabled {
		missing = append(missing, "specialty retail")
	}
	if !store.CultureDisplayEnabled {
		missing = append(missing, "Qinghai culture display")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing %s", ErrStoreNotCompliant, strings.Join(missing, ", "))
	}
	return nil
}

func ValidateLogoVersion(license BrandLicense, store StoreProfile) error {
	if license.LogoVersion <= 0 || store.ApprovedLogoVersion != license.LogoVersion {
		return fmt.Errorf("%w: approved logo version is inconsistent", ErrStoreNotCompliant)
	}
	if store.DisplayedLogoVersion != store.ApprovedLogoVersion {
		return fmt.Errorf("%w: storefront uses an obsolete logo", ErrStoreNotCompliant)
	}
	return nil
}

func ValidateMenuBrandName(store StoreProfile) error {
	name := strings.Join(strings.Fields(store.MenuBrandName), " ")
	if name != "青海拉面" {
		return fmt.Errorf("%w: menu brand name must be 青海拉面", ErrStoreNotCompliant)
	}
	return nil
}

func ValidateOperatorTraining(store StoreProfile, course string, at time.Time) error {
	for _, certificate := range store.Operators {
		if certificate.CourseCode == course && !at.Before(certificate.IssuedAt) && at.Before(certificate.ExpiresAt) {
			return nil
		}
	}
	return fmt.Errorf("%w: no operator holds active %s training", ErrStoreNotCompliant, course)
}

func CanRenewLicense(license BrandLicense, store StoreProfile, now time.Time) error {
	if license.Status != LicenseActive || license.StoreID != store.ID {
		return fmt.Errorf("%w: license is not active for this store", ErrLicenseNotRenewable)
	}
	if !license.LastInspectionPass || license.OutstandingActions != 0 {
		return fmt.Errorf("%w: inspection or corrective actions are incomplete", ErrLicenseNotRenewable)
	}
	if now.Before(license.EffectiveTo.Add(-90 * 24 * time.Hour)) {
		return fmt.Errorf("%w: renewal window has not opened", ErrLicenseNotRenewable)
	}
	return ValidateThreeStoreModel(store)
}

func ValidateLicenseTransfer(license BrandLicense, currentOwner, proposedOwner string) error {
	if license.OwnerID != currentOwner || strings.TrimSpace(proposedOwner) == "" {
		return fmt.Errorf("%w: owner identity is invalid", ErrInvalidLicense)
	}
	if proposedOwner != currentOwner {
		return fmt.Errorf("%w: regional brand licenses cannot be transferred", ErrInvalidLicense)
	}
	return nil
}
