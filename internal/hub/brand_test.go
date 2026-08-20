package hub

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func compliantStore(now time.Time) StoreProfile {
	return StoreProfile{
		ID:                     "store-1",
		OwnerID:                "owner-1",
		CityCode:               "TJ",
		DiningEnabled:          true,
		SpecialtyRetailEnabled: true,
		CultureDisplayEnabled:  true,
		ApprovedLogoVersion:    3,
		DisplayedLogoVersion:   3,
		MenuBrandName:          "青海拉面",
		Operators:              []OperatorCertificate{{OperatorID: "op-1", CourseCode: "quality-v2", IssuedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}},
	}
}

func activeLicense(now time.Time) BrandLicense {
	return BrandLicense{
		ID:                 "license-1",
		StoreID:            "store-1",
		OwnerID:            "owner-1",
		RegionCodes:        []string{"TJ", "JS"},
		LogoVersion:        3,
		StandardVersion:    2,
		Status:             LicenseActive,
		EffectiveFrom:      now.Add(-24 * time.Hour),
		EffectiveTo:        now.Add(30 * 24 * time.Hour),
		LastInspectionPass: true,
	}
}

func TestServiceActivatesCompliantLicense(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	service := NewService(NewRegistry(), func() time.Time { return now })
	license := activeLicense(now)
	license.Status = LicensePending

	activated, err := service.ActivateLicense(context.Background(), license, compliantStore(now))
	require.NoError(t, err)
	require.Equal(t, LicenseActive, activated.Status)
	require.Equal(t, 1, activated.Version)
}

func TestThreeStoreModelRequiresAllFunctions(t *testing.T) {
	store := compliantStore(time.Now())
	store.CultureDisplayEnabled = false
	require.ErrorIs(t, ValidateThreeStoreModel(store), ErrStoreNotCompliant)
}

func TestBrothRequiresApprovedTraceableBones(t *testing.T) {
	now := time.Now()
	batch := BrothBatch{ID: "broth-1", BoneLotIDs: []string{"bone-1"}, WaterLiters: 40, BoneKilograms: 6}
	lots := map[string]IngredientLot{"bone-1": {ID: "bone-1", IngredientCode: "yak-bone", ProducedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}}
	require.NoError(t, ValidateBrothBatch(batch, lots))
}

func TestRegistryHonorsCancellationAndIsolation(t *testing.T) {
	registry := NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, registry.SaveLicense(ctx, BrandLicense{ID: "x"}, 0), context.Canceled)

	license := BrandLicense{ID: "license-1", RegionCodes: []string{"TJ"}}
	require.NoError(t, registry.SaveLicense(context.Background(), license, 0))
	loaded, ok, err := registry.License(context.Background(), "license-1")
	require.NoError(t, err)
	require.True(t, ok)
	loaded.RegionCodes[0] = "MUTATED"
	again, _, err := registry.License(context.Background(), "license-1")
	require.NoError(t, err)
	require.Equal(t, []string{"TJ"}, again.RegionCodes)
}

func TestRecallRequiresEveryAffectedStore(t *testing.T) {
	recall := Recall{AffectedStores: []string{"s1", "s2"}, Acknowledged: []string{"s1"}}
	missing, complete := RecallCoverage(recall)
	require.False(t, complete)
	require.Equal(t, []string{"s2"}, missing)
	require.True(t, errors.Is(CanCloseRecall(recall, time.Now()), ErrRecallIncomplete))
}

func TestCatalogCloneDoesNotShareIngredientLots(t *testing.T) {
	original := CatalogSnapshot{Listings: []ProductListing{{ID: "p1", IngredientLotIDs: []string{"lot-1"}}}}
	clone := CloneCatalog(original)
	clone.Listings[0].IngredientLotIDs[0] = "changed"
	require.Equal(t, "lot-1", original.Listings[0].IngredientLotIDs[0])
}
