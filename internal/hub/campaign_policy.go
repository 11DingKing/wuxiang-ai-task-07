package hub

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

func ValidateCampaign(campaign CultureCampaign, store StoreProfile, listings map[string]ProductListing, at time.Time) error {
	if campaign.StoreID != store.ID || campaign.Title == "" || !campaign.StartsAt.Before(campaign.EndsAt) {
		return fmt.Errorf("%w: campaign identity or window is invalid", ErrCampaignNotApproved)
	}
	if campaign.ContentVersion != campaign.ApprovedVersion || !campaign.RegionalBrandLogo {
		return fmt.Errorf("%w: content or regional brand logo is not approved", ErrCampaignNotApproved)
	}
	if at.Before(campaign.StartsAt) || !at.Before(campaign.EndsAt) {
		return fmt.Errorf("%w: campaign is outside its active window", ErrCampaignNotApproved)
	}
	for _, sku := range campaign.FeaturedSKUs {
		listing, ok := listings[sku]
		if !ok || listing.StoreID != store.ID || !listing.Published {
			return fmt.Errorf("%w: featured SKU %s is unavailable", ErrCampaignNotApproved, sku)
		}
	}
	return nil
}

func ValidateDestinationCoverage(campaign CultureCampaign, licensedRegions []string) error {
	if len(campaign.DestinationCodes) == 0 {
		return fmt.Errorf("%w: campaign destinations are required", ErrCampaignNotApproved)
	}
	for _, destination := range campaign.DestinationCodes {
		if !slices.Contains(licensedRegions, destination) {
			return fmt.Errorf("%w: destination %s is outside license scope", ErrCampaignNotApproved, destination)
		}
	}
	return nil
}

func ValidateSubsidyClaim(claim SubsidyClaim, store StoreProfile, license BrandLicense, now time.Time) error {
	if claim.StoreID != store.ID || license.StoreID != store.ID || license.Status != LicenseActive {
		return fmt.Errorf("%w: claim store has no active license", ErrInvalidClaim)
	}
	if strings.TrimSpace(claim.Period) == "" || claim.EligibleSales <= 0 || len(claim.EvidenceHashes) == 0 {
		return fmt.Errorf("%w: period, eligible sales and evidence are required", ErrInvalidClaim)
	}
	if claim.SubmittedAt.After(now) || now.Sub(claim.SubmittedAt) > 30*24*time.Hour {
		return fmt.Errorf("%w: claim was submitted outside the filing window", ErrInvalidClaim)
	}
	return nil
}

func MergeFeaturedSKUs(existing, incoming []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(incoming))
	merged := make([]string, 0, len(existing)+len(incoming))
	for _, values := range [][]string{existing, incoming} {
		for _, sku := range values {
			sku = strings.TrimSpace(sku)
			if sku == "" {
				continue
			}
			if _, exists := seen[sku]; exists {
				continue
			}
			seen[sku] = struct{}{}
			merged = append(merged, sku)
		}
	}
	return merged
}
