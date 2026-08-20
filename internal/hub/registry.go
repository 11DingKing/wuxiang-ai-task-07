package hub

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"
)

type Registry struct {
	mu          sync.RWMutex
	licenses    map[string]BrandLicense
	inspections map[string]Inspection
	catalogs    map[string]CatalogSnapshot
}

func NewRegistry() *Registry {
	return &Registry{
		licenses:    make(map[string]BrandLicense),
		inspections: make(map[string]Inspection),
		catalogs:    make(map[string]CatalogSnapshot),
	}
}

func (r *Registry) SaveLicense(ctx context.Context, license BrandLicense, expectedVersion int) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("save license: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, exists := r.licenses[license.ID]
	if exists && current.Version != expectedVersion {
		return fmt.Errorf("%w: expected license version %d, got %d", ErrInspectionConflict, expectedVersion, current.Version)
	}
	license.RegionCodes = slices.Clone(license.RegionCodes)
	license.Version = expectedVersion + 1
	r.licenses[license.ID] = license
	return nil
}

func (r *Registry) License(ctx context.Context, id string) (BrandLicense, bool, error) {
	if err := ctx.Err(); err != nil {
		return BrandLicense{}, false, fmt.Errorf("read license: %w", err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	license, ok := r.licenses[id]
	license.RegionCodes = slices.Clone(license.RegionCodes)
	return license, ok, nil
}

func (r *Registry) SaveInspection(ctx context.Context, inspection Inspection, expectedVersion int) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("save inspection: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, exists := r.inspections[inspection.ID]
	if exists && current.Version != expectedVersion {
		return fmt.Errorf("%w: expected inspection version %d, got %d", ErrInspectionConflict, expectedVersion, current.Version)
	}
	inspection.Sections = slices.Clone(inspection.Sections)
	inspection.Version = expectedVersion + 1
	r.inspections[inspection.ID] = inspection
	return nil
}

func (r *Registry) PublishCatalog(ctx context.Context, snapshot CatalogSnapshot) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("publish catalog: %w", err)
	}
	if snapshot.StoreID == "" || snapshot.GeneratedAt.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("%w: catalog identity or generation time is invalid", ErrStoreNotCompliant)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.catalogs[snapshot.StoreID] = CloneCatalog(snapshot)
	return nil
}

func (r *Registry) Catalog(ctx context.Context, storeID string) (CatalogSnapshot, bool, error) {
	if err := ctx.Err(); err != nil {
		return CatalogSnapshot{}, false, fmt.Errorf("read catalog: %w", err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot, ok := r.catalogs[storeID]
	return CloneCatalog(snapshot), ok, nil
}
