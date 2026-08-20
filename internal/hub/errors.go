package hub

import "errors"

var (
	ErrInvalidLicense      = errors.New("invalid brand license")
	ErrLicenseNotRenewable = errors.New("brand license is not renewable")
	ErrStoreNotCompliant   = errors.New("store does not satisfy brand requirements")
	ErrInvalidIngredient   = errors.New("ingredient lot is not compliant")
	ErrInspectionConflict  = errors.New("inspection version conflict")
	ErrRecallIncomplete    = errors.New("recall acknowledgement is incomplete")
	ErrCampaignNotApproved = errors.New("campaign content is not approved")
	ErrInvalidClaim        = errors.New("subsidy claim is invalid")
	ErrAppealExpired       = errors.New("inspection appeal window expired")
)
