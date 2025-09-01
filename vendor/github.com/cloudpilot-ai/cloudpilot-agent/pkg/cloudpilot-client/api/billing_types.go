package api

type ClusterBilling struct {
	AWSBilling AWSBilling `json:"awsBilling"`
}

type AWSBilling struct {
	SpotCost              string              `json:"spotCost"`
	UncoveredOnDemandCost string              `json:"uncoveredOnDemandCost"`
	SavingsPlans          []ActiveSavingsPlan `json:"savingsPlans"`
}

type ActiveSavingsPlan struct {
	ID             string `json:"id,omitempty"`
	Type           string `json:"type,omitempty"`
	InstanceFamily string `json:"instanceFamily,omitempty"`
	Region         string `json:"region,omitempty"`
	PaymentOption  string `json:"paymentOption,omitempty"`
	Commitment     string `json:"commitment,omitempty"`
	Usage          string `json:"usage,omitempty"`
	StartDate      string `json:"startDate,omitempty"`
	EndDate        string `json:"endDate,omitempty"`
}
