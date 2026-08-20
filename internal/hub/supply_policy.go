package hub

import (
	"fmt"
	"slices"
	"time"
)

func ValidateLotTrace(lot IngredientLot, known map[string]IngredientLot) error {
	if lot.ID == "" || lot.SupplierID == "" || lot.CertificateID == "" || lot.OriginRegion == "" {
		return fmt.Errorf("%w: lot identity, supplier, certificate and origin are required", ErrInvalidIngredient)
	}
	if !lot.ProducedAt.Before(lot.ExpiresAt) {
		return fmt.Errorf("%w: production and expiry times are inconsistent", ErrInvalidIngredient)
	}
	for _, parent := range lot.ParentLotIDs {
		parentLot, ok := known[parent]
		if !ok || parentLot.ID == lot.ID || parentLot.ProducedAt.After(lot.ProducedAt) {
			return fmt.Errorf("%w: parent lot %s is invalid", ErrInvalidIngredient, parent)
		}
	}
	return nil
}

func ValidateHighlandProduct(listing ProductListing, lots map[string]IngredientLot) error {
	if len(listing.IngredientLotIDs) == 0 || listing.OriginRegion == "" {
		return fmt.Errorf("%w: listing origin and ingredient lots are required", ErrInvalidIngredient)
	}
	for _, id := range listing.IngredientLotIDs {
		lot, ok := lots[id]
		if !ok || lot.OriginRegion != listing.OriginRegion {
			return fmt.Errorf("%w: listing origin does not match lot %s", ErrInvalidIngredient, id)
		}
	}
	return nil
}

func ValidateListingPrices(listing ProductListing) error {
	if listing.OnlinePriceCents <= 0 || listing.StorePriceCents <= 0 {
		return fmt.Errorf("%w: online and store prices must be positive", ErrStoreNotCompliant)
	}
	difference := listing.OnlinePriceCents - listing.StorePriceCents
	if difference < 0 {
		difference = -difference
	}
	if difference*100 > listing.StorePriceCents*5 {
		return fmt.Errorf("%w: channel price difference exceeds five percent", ErrStoreNotCompliant)
	}
	return nil
}

func AvailableStock(listing ProductListing) (int, error) {
	if listing.Stock < 0 || listing.Reserved < 0 || listing.Reserved > listing.Stock {
		return 0, fmt.Errorf("%w: stock or reservation is inconsistent", ErrStoreNotCompliant)
	}
	return listing.Stock - listing.Reserved, nil
}

func RecallCoverage(recall Recall) (missing []string, complete bool) {
	acknowledged := make(map[string]struct{}, len(recall.Acknowledged))
	for _, storeID := range recall.Acknowledged {
		acknowledged[storeID] = struct{}{}
	}
	for _, storeID := range recall.AffectedStores {
		if _, ok := acknowledged[storeID]; !ok {
			missing = append(missing, storeID)
		}
	}
	slices.Sort(missing)
	return missing, len(missing) == 0
}

func CanCloseRecall(recall Recall, now time.Time) error {
	if recall.ClosedAt != nil || now.Before(recall.CreatedAt) {
		return fmt.Errorf("%w: recall lifecycle is invalid", ErrRecallIncomplete)
	}
	missing, complete := RecallCoverage(recall)
	if !complete {
		return fmt.Errorf("%w: %d stores have not acknowledged", ErrRecallIncomplete, len(missing))
	}
	return nil
}

func CloneCatalog(snapshot CatalogSnapshot) CatalogSnapshot {
	clone := snapshot
	clone.Listings = make([]ProductListing, len(snapshot.Listings))
	for index, listing := range snapshot.Listings {
		clone.Listings[index] = listing
		clone.Listings[index].IngredientLotIDs = slices.Clone(listing.IngredientLotIDs)
	}
	return clone
}
