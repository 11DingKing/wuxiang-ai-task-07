package hub

import "time"

type LicenseStatus string

const (
	LicensePending   LicenseStatus = "pending"
	LicenseActive    LicenseStatus = "active"
	LicenseSuspended LicenseStatus = "suspended"
	LicenseExpired   LicenseStatus = "expired"
)

type BrandLicense struct {
	ID                 string
	StoreID            string
	OwnerID            string
	RegionCodes        []string
	LogoVersion        int
	StandardVersion    int
	Status             LicenseStatus
	EffectiveFrom      time.Time
	EffectiveTo        time.Time
	LastInspectionID   string
	LastInspectionPass bool
	OutstandingActions int
	Version            int
}

type StoreProfile struct {
	ID                     string
	OwnerID                string
	CityCode               string
	DiningEnabled          bool
	SpecialtyRetailEnabled bool
	CultureDisplayEnabled  bool
	ApprovedLogoVersion    int
	DisplayedLogoVersion   int
	MenuBrandName          string
	Operators              []OperatorCertificate
	UpdatedAt              time.Time
}

type OperatorCertificate struct {
	OperatorID string
	CourseCode string
	IssuedAt   time.Time
	ExpiresAt  time.Time
}

type IngredientLot struct {
	ID             string
	SupplierID     string
	IngredientCode string
	OriginRegion   string
	CertificateID  string
	ProducedAt     time.Time
	ExpiresAt      time.Time
	MinTemperature float64
	MaxTemperature float64
	Temperatures   []float64
	ParentLotIDs   []string
}

type SupplierCertificate struct {
	ID           string
	SupplierID   string
	Scope        []string
	EffectiveAt  time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	DocumentHash string
}

type BrothBatch struct {
	ID              string
	StoreID         string
	BoneLotIDs      []string
	WaterLiters     float64
	BoneKilograms   float64
	StartedAt       time.Time
	FinishedAt      time.Time
	StandardVersion int
}

type ChiliOilBatch struct {
	ID              string
	StoreID         string
	ChiliLotID      string
	Cultivar        string
	OilLotID        string
	ProducedAt      time.Time
	StandardVersion int
}

type Inspection struct {
	ID              string
	StoreID         string
	InspectorID     string
	StartedAt       time.Time
	CompletedAt     time.Time
	StandardVersion int
	Sections        []InspectionSection
	Score           int
	Passed          bool
	Version         int
}

type InspectionSection struct {
	Code     string
	Required bool
	Score    int
	Evidence []string
}

type CorrectiveAction struct {
	ID           string
	InspectionID string
	StoreID      string
	Severity     string
	Description  string
	DueAt        time.Time
	CompletedAt  *time.Time
	VerifiedAt   *time.Time
	VerifierID   string
}

type ProductListing struct {
	ID               string
	StoreID          string
	SKU              string
	Name             string
	IngredientLotIDs []string
	OriginRegion     string
	OnlinePriceCents int
	StorePriceCents  int
	Stock            int
	Reserved         int
	Published        bool
}

type Recall struct {
	ID             string
	IngredientLot  string
	AffectedStores []string
	Acknowledged   []string
	CreatedAt      time.Time
	Deadline       time.Time
	ClosedAt       *time.Time
}

type CultureCampaign struct {
	ID                string
	StoreID           string
	Title             string
	ContentVersion    int
	ApprovedVersion   int
	StartsAt          time.Time
	EndsAt            time.Time
	FeaturedSKUs      []string
	DestinationCodes  []string
	RegionalBrandLogo bool
}

type SubsidyClaim struct {
	ID             string
	StoreID        string
	Period         string
	EligibleSales  int64
	EvidenceHashes []string
	SubmittedAt    time.Time
	Status         string
}

type Appeal struct {
	ID           string
	InspectionID string
	StoreID      string
	SubmittedAt  time.Time
	Reason       string
	Evidence     []string
}

type CatalogSnapshot struct {
	StoreID     string
	Listings    []ProductListing
	GeneratedAt time.Time
}
