package hub

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var aiTestNow = time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)

func newAITestService() (*AIServices, *AIRegistry) {
	registry := NewAIRegistry()
	return NewAIServices(registry, func() time.Time { return aiTestNow }), registry
}

func testApplication(id, tenant string) EnterpriseApplication {
	return EnterpriseApplication{ID: id, TenantID: tenant, ApplicantID: "operator-" + id, CompanyName: "南A智算企业 " + id, Industry: "logistics", ASEANCountry: " vn ", Materials: []string{"plan.pdf", "security.pdf"}, RequestedPetaops: 100}
}

func reserveReady(t *testing.T, service *AIServices, registry *AIRegistry, appID string) {
	t.Helper()
	if err := registry.AddCluster(context.Background(), ComputeCluster{ID: "cluster-1", Region: "五象", TotalPetaops: 500}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitApplication(context.Background(), testApplication(appID, "tenant-a")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReserveCompute(context.Background(), appID, "cluster-1", 100); err != nil {
		t.Fatal(err)
	}
}

func TestAIApplicationSubmissionClonesMaterials(t *testing.T) {
	service, registry := newAITestService()
	materials := []string{"proposal.pdf", "model-card.pdf"}
	app := testApplication("app-clone", "tenant-a")
	app.Materials = materials
	if _, err := service.SubmitApplication(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	materials[0] = "tampered.pdf"
	stored, ok := registry.Application(app.ID)
	if !ok || stored.Materials[0] != "proposal.pdf" {
		t.Fatalf("materials leaked into stored application: %#v", stored.Materials)
	}
}

func TestAIComputeReservationConcurrentCapacity(t *testing.T) {
	service, registry := newAITestService()
	if err := registry.AddCluster(context.Background(), ComputeCluster{ID: "cluster-concurrent", Region: "五象", TotalPetaops: 100}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"app-c1", "app-c2"} {
		if _, err := service.SubmitApplication(context.Background(), testApplication(id, "tenant-a")); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []string{"app-c1", "app-c2"} {
		wg.Add(1)
		go func(applicationID string) {
			defer wg.Done()
			_, err := service.ReserveCompute(context.Background(), applicationID, "cluster-concurrent", 100)
			results <- err
		}(id)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrAICapacity) {
			t.Fatalf("unexpected reservation result: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected one reservation, got %d", successes)
	}
	cluster, _ := registry.Cluster("cluster-concurrent")
	if cluster.Allocated != 100 {
		t.Fatalf("allocated compute oversold: %d", cluster.Allocated)
	}
}

func TestAIReleaseComputeIsIdempotent(t *testing.T) {
	service, registry := newAITestService()
	reserveReady(t, service, registry, "app-release")
	if err := service.ReleaseCompute(context.Background(), "app-release"); err != nil {
		t.Fatal(err)
	}
	if err := service.ReleaseCompute(context.Background(), "app-release"); err != nil {
		t.Fatal(err)
	}
	cluster, _ := registry.Cluster("cluster-1")
	if cluster.Allocated != 0 {
		t.Fatalf("release was not idempotent: %d", cluster.Allocated)
	}
}

func TestAIApprovalRequiresRoute(t *testing.T) {
	service, registry := newAITestService()
	reserveReady(t, service, registry, "app-route-required")
	app, _ := registry.Application("app-route-required")
	app.RouteID = "route-required"
	registry.mu.Lock()
	registry.applications[app.ID] = app
	registry.mu.Unlock()
	if _, err := service.ApproveApplication(context.Background(), app.ID, "reviewer-1"); !errors.Is(err, ErrAIRouteNotFound) {
		t.Fatalf("expected unavailable route, got %v", err)
	}
}

func TestAIAssignRouteTenantIsolation(t *testing.T) {
	service, registry := newAITestService()
	reserveReady(t, service, registry, "app-route-tenant")
	if err := registry.AddRoute(context.Background(), CrossBorderRoute{ID: "route-tenant", TenantID: "tenant-b", Country: "VN", CapacityMbps: 100, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := service.AssignRoute(context.Background(), "app-route-tenant", "route-tenant", 10); !errors.Is(err, ErrAIAuthorization) {
		t.Fatalf("expected tenant isolation, got %v", err)
	}
}

func TestAIServiceOfferNormalizesCountry(t *testing.T) {
	service, registry := newAITestService()
	if err := registry.AddOffer(context.Background(), ServiceOffer{ID: "offer-country", ProviderID: "provider-1", Countries: []string{" vn "}, Capacity: 1, EffectiveFrom: aiTestNow.Add(-time.Hour), EffectiveTo: aiTestNow.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-offer-country", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	if err := service.ApplyOffer(context.Background(), "app-offer-country", "offer-country", aiTestNow); err != nil {
		t.Fatalf("country normalization rejected valid offer: %v", err)
	}
}

func TestAIServiceOfferExpiryBoundary(t *testing.T) {
	service, registry := newAITestService()
	if err := registry.AddOffer(context.Background(), ServiceOffer{ID: "offer-expiry", ProviderID: "provider-1", Countries: []string{"VN"}, Capacity: 2, EffectiveFrom: aiTestNow.Add(-time.Hour), EffectiveTo: aiTestNow}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-offer-expiry", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	if err := service.ApplyOffer(context.Background(), "app-offer-expiry", "offer-expiry", aiTestNow); !errors.Is(err, ErrAICapacity) {
		t.Fatalf("expired offer was accepted: %v", err)
	}
}

func TestAIAppliedOfferConsumesCapacity(t *testing.T) {
	service, registry := newAITestService()
	if err := registry.AddOffer(context.Background(), ServiceOffer{ID: "offer-capacity", ProviderID: "provider-1", Countries: []string{"VN"}, Capacity: 1, EffectiveFrom: aiTestNow.Add(-time.Hour), EffectiveTo: aiTestNow.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"app-offer-1", "app-offer-2"} {
		if _, err := service.SubmitApplication(context.Background(), testApplication(id, "tenant-a")); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.ApplyOffer(context.Background(), "app-offer-1", "offer-capacity", aiTestNow); err != nil {
		t.Fatal(err)
	}
	if err := service.ApplyOffer(context.Background(), "app-offer-2", "offer-capacity", aiTestNow); !errors.Is(err, ErrAICapacity) {
		t.Fatalf("offer capacity was oversold: %v", err)
	}
}

func TestAISceneRequiresEvidence(t *testing.T) {
	service, registry := newAITestService()
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-scene-evidence", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	_, err := service.PublishScene(context.Background(), SceneSubmission{ID: "scene-evidence", TenantID: "tenant-a", OwnerID: "school-1", ApplicationID: "app-scene-evidence", Evidence: []string{"photo.jpg"}})
	if !errors.Is(err, ErrAIValidation) {
		t.Fatalf("expected evidence validation, got %v", err)
	}
	_ = registry
}

func TestAISceneEvidenceIsIsolated(t *testing.T) {
	service, registry := newAITestService()
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-scene-clone", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	evidence := []string{"photo.jpg", "report.pdf"}
	if _, err := service.PublishScene(context.Background(), SceneSubmission{ID: "scene-clone", TenantID: "tenant-a", OwnerID: "school-1", ApplicationID: "app-scene-clone", Evidence: evidence}); err != nil {
		t.Fatal(err)
	}
	evidence[0] = "altered.jpg"
	scene, _ := registry.Scene("scene-clone")
	if scene.Evidence[0] != "photo.jpg" {
		t.Fatalf("scene evidence was aliased: %#v", scene.Evidence)
	}
}

func TestAISceneApprovalState(t *testing.T) {
	service, registry := newAITestService()
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-scene-state", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PublishScene(context.Background(), SceneSubmission{ID: "scene-state", TenantID: "tenant-a", OwnerID: "school-1", ApplicationID: "app-scene-state", Evidence: []string{"a", "b"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApproveScene(context.Background(), "scene-state", "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApproveScene(context.Background(), "scene-state", "reviewer"); !errors.Is(err, ErrAIInvalidState) {
		t.Fatalf("approved scene reopened: %v", err)
	}
	_ = registry
}

func TestAIOPCEnrollmentCapacity(t *testing.T) {
	service, registry := newAITestService()
	if err := registry.AddCommunity(context.Background(), OPCCommunity{ID: "opc-cap", Name: "南A东盟谷", Capacity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnrollOPC(context.Background(), "opc-cap", "founder-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnrollOPC(context.Background(), "opc-cap", "founder-2"); !errors.Is(err, ErrAICapacity) {
		t.Fatalf("community exceeded capacity: %v", err)
	}
}

func TestAIOPCEnrollmentIdempotent(t *testing.T) {
	service, registry := newAITestService()
	if err := registry.AddCommunity(context.Background(), OPCCommunity{ID: "opc-idem", Name: "南A东盟谷", Capacity: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnrollOPC(context.Background(), "opc-idem", "founder-1"); err != nil {
		t.Fatal(err)
	}
	community, err := service.EnrollOPC(context.Background(), "opc-idem", "founder-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(community.MemberIDs) != 1 {
		t.Fatalf("duplicate enrollment created a second seat: %#v", community.MemberIDs)
	}
}

func TestAIResubmissionPreservesAttempt(t *testing.T) {
	service, registry := newAITestService()
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-resubmit", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	registry.mu.Lock()
	app := registry.applications["app-resubmit"]
	app.Status = ApplicationReturned
	registry.applications[app.ID] = app
	registry.mu.Unlock()
	updated, err := service.ResubmitApplication(context.Background(), "app-resubmit", []string{"new-plan.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Attempt != 2 || updated.Status != ApplicationPending {
		t.Fatalf("unexpected resubmission: %#v", updated)
	}
}

func TestAICannotClosePending(t *testing.T) {
	service, registry := newAITestService()
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-close", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	if err := service.CloseApplication(context.Background(), "app-close"); !errors.Is(err, ErrAIInvalidState) {
		t.Fatalf("pending application closed: %v", err)
	}
	_ = registry
}

func TestAIListTenantIsolation(t *testing.T) {
	service, _ := newAITestService()
	for _, tenant := range []string{"tenant-a", "tenant-b"} {
		if _, err := service.SubmitApplication(context.Background(), testApplication("app-list-"+tenant, tenant)); err != nil {
			t.Fatal(err)
		}
	}
	result, err := service.ListApplications(context.Background(), AIQuery{TenantID: "tenant-a"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Applications[0].TenantID != "tenant-a" {
		t.Fatalf("cross-tenant result: %#v", result)
	}
}

func TestAIListPaginationTotal(t *testing.T) {
	service, _ := newAITestService()
	for i := 0; i < 3; i++ {
		if _, err := service.SubmitApplication(context.Background(), testApplication("app-page-"+string(rune('a'+i)), "tenant-a")); err != nil {
			t.Fatal(err)
		}
	}
	result, err := service.ListApplications(context.Background(), AIQuery{TenantID: "tenant-a", Offset: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applications) != 1 || result.Total != 3 {
		t.Fatalf("pagination total mismatch: %#v", result)
	}
}

func TestAIContextCancellation(t *testing.T) {
	service, _ := newAITestService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.SubmitApplication(ctx, testApplication("app-cancelled", "tenant-a")); !errors.Is(err, context.Canceled) {
		t.Fatalf("context cancellation lost: %v", err)
	}
}

func TestAIRouteCapacityIsAtomic(t *testing.T) {
	service, registry := newAITestService()
	if err := registry.AddRoute(context.Background(), CrossBorderRoute{ID: "route-cap", TenantID: "tenant-a", Country: "VN", CapacityMbps: 100, Active: true}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"app-route-cap-1", "app-route-cap-2"} {
		if _, err := service.SubmitApplication(context.Background(), testApplication(id, "tenant-a")); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []string{"app-route-cap-1", "app-route-cap-2"} {
		wg.Add(1)
		go func(applicationID string) {
			defer wg.Done()
			results <- service.AssignRoute(context.Background(), applicationID, "route-cap", 100)
		}(id)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ErrAICapacity) {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("route oversold, successes=%d", success)
	}
}

func TestAIApplicationInputNormalization(t *testing.T) {
	service, registry := newAITestService()
	app := testApplication("app-normalize", "tenant-a")
	app.CompanyName = "  东盟物流  "
	if _, err := service.SubmitApplication(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	stored, _ := registry.Application(app.ID)
	if stored.CompanyName != "东盟物流" || stored.ASEANCountry != "VN" {
		t.Fatalf("input was not normalized: %#v", stored)
	}
}

func TestAIApplicationDuplicateRejected(t *testing.T) {
	service, _ := newAITestService()
	app := testApplication("app-duplicate", "tenant-a")
	if _, err := service.SubmitApplication(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitApplication(context.Background(), app); !errors.Is(err, ErrAIConflict) {
		t.Fatalf("duplicate application accepted: %v", err)
	}
}

func TestAIApplicationEventAudit(t *testing.T) {
	service, registry := newAITestService()
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-audit", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	events := registry.Events()
	if len(events) != 1 || events[0].Action != "application.submitted" || events[0].EntityID != "app-audit" {
		t.Fatalf("missing audit event: %#v", events)
	}
}

func TestAIClusterVersionAdvances(t *testing.T) {
	service, registry := newAITestService()
	reserveReady(t, service, registry, "app-version")
	cluster, _ := registry.Cluster("cluster-1")
	if cluster.Version != 1 {
		t.Fatalf("cluster version did not advance: %d", cluster.Version)
	}
}

func TestAIReleaseMissingCluster(t *testing.T) {
	service, registry := newAITestService()
	reserveReady(t, service, registry, "app-missing-cluster")
	registry.mu.Lock()
	delete(registry.clusters, "cluster-1")
	registry.mu.Unlock()
	if err := service.ReleaseCompute(context.Background(), "app-missing-cluster"); !errors.Is(err, ErrAIClusterNotFound) {
		t.Fatalf("missing cluster was ignored: %v", err)
	}
}

func TestAIOfferCountryListIsolation(t *testing.T) {
	service, registry := newAITestService()
	countries := []string{"VN", "TH"}
	if err := registry.AddOffer(context.Background(), ServiceOffer{ID: "offer-list", ProviderID: "provider", Countries: countries, Capacity: 2, EffectiveFrom: aiTestNow.Add(-time.Hour), EffectiveTo: aiTestNow.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	countries[0] = "XX"
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-offer-list", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	if err := service.ApplyOffer(context.Background(), "app-offer-list", "offer-list", aiTestNow); err != nil {
		t.Fatalf("offer countries leaked: %v", err)
	}
}

func TestAICommunityMapIsolation(t *testing.T) {
	service, registry := newAITestService()
	milestones := map[string]string{"phase-1": "planned"}
	if err := registry.AddCommunity(context.Background(), OPCCommunity{ID: "opc-map", Name: "南A东盟谷", Capacity: 2, Milestones: milestones}); err != nil {
		t.Fatal(err)
	}
	milestones["phase-1"] = "tampered"
	community, _ := registry.Community("opc-map")
	if community.Milestones["phase-1"] != "planned" {
		t.Fatalf("community map leaked: %#v", community.Milestones)
	}
	_ = service
}

func TestAIPendingToApprovedFlow(t *testing.T) {
	service, registry := newAITestService()
	reserveReady(t, service, registry, "app-approve")
	if err := registry.AddRoute(context.Background(), CrossBorderRoute{ID: "route-approve", TenantID: "tenant-a", Country: "VN", CapacityMbps: 20, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := service.AssignRoute(context.Background(), "app-approve", "route-approve", 10); err != nil {
		t.Fatal(err)
	}
	updated, err := service.ApproveApplication(context.Background(), "app-approve", "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != ApplicationApproved {
		t.Fatalf("application did not approve: %#v", updated)
	}
}

func TestAIClosedStateTerminal(t *testing.T) {
	service, registry := newAITestService()
	reserveReady(t, service, registry, "app-terminal")
	if _, err := service.ApproveApplication(context.Background(), "app-terminal", "reviewer"); err != nil {
		t.Fatal(err)
	}
	registry.mu.Lock()
	app := registry.applications["app-terminal"]
	app.Status = ApplicationActive
	registry.applications[app.ID] = app
	registry.mu.Unlock()
	if err := service.CloseApplication(context.Background(), "app-terminal"); err != nil {
		t.Fatal(err)
	}
	if err := service.CloseApplication(context.Background(), "app-terminal"); !errors.Is(err, ErrAIInvalidState) {
		t.Fatalf("closed application reopened: %v", err)
	}
}

func TestAIQueryStatusFilter(t *testing.T) {
	service, registry := newAITestService()
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-status-pending", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitApplication(context.Background(), testApplication("app-status-approved", "tenant-a")); err != nil {
		t.Fatal(err)
	}
	registry.mu.Lock()
	app := registry.applications["app-status-approved"]
	app.Status = ApplicationApproved
	registry.applications[app.ID] = app
	registry.mu.Unlock()
	result, err := service.ListApplications(context.Background(), AIQuery{TenantID: "tenant-a", Status: ApplicationApproved})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Applications[0].ID != "app-status-approved" {
		t.Fatalf("status filter leaked records: %#v", result)
	}
}

func TestAIServiceOfferRejectsReversedWindow(t *testing.T) {
	service, registry := newAITestService()
	err := registry.AddOffer(context.Background(), ServiceOffer{ID: "offer-window", ProviderID: "provider", Countries: []string{"VN"}, Capacity: 1, EffectiveFrom: aiTestNow.Add(time.Hour), EffectiveTo: aiTestNow})
	if !errors.Is(err, ErrAIValidation) {
		t.Fatalf("reversed service window accepted: %v", err)
	}
	_ = service
}
