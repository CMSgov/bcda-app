package constants

type DataRequestType uint8

const (
	DefaultRequest          DataRequestType = iota
	RetrieveNewBeneHistData                 // Allows caller to retrieve all of the data for newly attributed beneficiaries
	Runout                                  // Allows caller to retrieve claims data for beneficiaries no longer attributed to the ACO
)
