package hub

import "time"

type AIApplicationStatus string

const (
	ApplicationPending   AIApplicationStatus = "pending"
	ApplicationApproved  AIApplicationStatus = "approved"
	ApplicationActive    AIApplicationStatus = "active"
	ApplicationReturned  AIApplicationStatus = "returned"
	ApplicationClosed    AIApplicationStatus = "closed"
	ApplicationCancelled AIApplicationStatus = "cancelled"
)

type EnterpriseApplication struct {
	ID               string
	TenantID         string
	ApplicantID      string
	CompanyName      string
	Industry         string
	ASEANCountry     string
	Status           AIApplicationStatus
	Materials        []string
	RequestedPetaops int
	RouteID          string
	ServiceOfferID   string
	RuleVersion      int
	Attempt          int
	Deadline         time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Version          int
}

type ComputeReservation struct {
	ApplicationID string
	ClusterID     string
	Petaops       int
	ReservedAt    time.Time
	ReleasedAt    *time.Time
}

type ComputeCluster struct {
	ID           string
	Region       string
	TotalPetaops int
	Allocated    int
	Version      int
}

type CrossBorderRoute struct {
	ID            string
	TenantID      string
	Country       string
	CapacityMbps  int
	AllocatedMbps int
	LatencyTarget time.Duration
	Active        bool
	Version       int
}

type ServiceOffer struct {
	ID            string
	ProviderID    string
	Kind          string
	Countries     []string
	Capacity      int
	Used          int
	EffectiveFrom time.Time
	EffectiveTo   time.Time
	Version       int
}

type SceneSubmission struct {
	ID            string
	TenantID      string
	OwnerID       string
	SceneType     string
	Status        string
	Evidence      []string
	ApplicationID string
	SubmittedAt   time.Time
	ReviewedAt    *time.Time
	Version       int
}

type OPCCommunity struct {
	ID         string
	Name       string
	Capacity   int
	MemberIDs  []string
	MentorIDs  []string
	Milestones map[string]string
	Version    int
}

type AIServiceEvent struct {
	ID       string
	TenantID string
	EntityID string
	Action   string
	ActorID  string
	At       time.Time
	Metadata map[string]string
}

type AIQuery struct {
	TenantID string
	Status   AIApplicationStatus
	Country  string
	Offset   int
	Limit    int
}

type AIQueryResult struct {
	Applications []EnterpriseApplication
	Total        int
}
