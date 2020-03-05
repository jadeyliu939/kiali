package models

type Iter8Info struct {
	Enabled bool `json:"enabled"`
	Permissions ResourcePermissions `json:"permissions"`
}
/*
type ExperimentList struct {
	Namespace string `json:"namespace"`

	Experiments []ExperimentListItem `json:"experiments"`
}
*/
type ExperimentListItem struct {
	Name string `json:"name"`
	ResourceVersion string `json:"resourceVersion"`
	Phase string `json:"phase"`
	CreatedAt string `json:"createdAt"`
	StartedAT string `json:"startedAt"`
	EndedAt string `json:"endedAt"`
	Status string `json:"status"`
	Baseline string `json:"baseline"`
	BaselinePercentage int `json:"baselinePercentage"`
	Candidate string `json:"candidate"`
	CandidatePercentage int `json:"candidatePercentage"`
	Namespace string `json:"namespace"`
	TargetService string `json:"targetService"`
	TargetServiceNamespace string `json:"targetServiceNamespace"`
	AssessmentConclusion string `json:"assessmentConclusion"`
}
type ExperimentDetail struct {
	ExperimentItem ExperimentListItem `json:"experimentItem"`
	CriteriaDetails []CriteriaDetail `json:"criterias"`
	TrafficControl TrafficControl `json:"trafficControl"`
}

type CriteriaDetail struct {
	Name string `json:"name"`
	Criteria Criteria `json:"criteria"`
	Metric Metric `json:"metric"`
}

type Metric struct {
	AbsentValue string `json:"absent_value"`
	IsCounter bool `json:"is_counter"`
	QueryTemplate string `json:"query_template"`
	SampleSizeTemplate string `json:"sample_size_template"`
}

type ExperimentSpec struct {
	Name string `json:"name"`
	Namespace string `json:"namespace"`
	Service string `json:"service"`
	APIVersion string `json:"apiversion"`
	Baseline string `json:"baseline"`
	Candidate string `json:"candidate"`
	TrafficControl TrafficControl `json:"trafficControl"`

	Criterias Criteria `json:"criteria"`
}
type TrafficControl struct {
	Algorithm string `json:"algorithm"`
	Interval string `json:"interval"`
	MaxIteration int  `json:"maxIteration"`
	MaxTrafficPercentage float64 `json:"maxTrafficPercentage"`
	TrafficStepSize float64 `json:"trafficStepSize"`
}
type Criteria struct {
	Metric string `json:"metric"`
	ToleranceType string `json:"toleranceType"`
	Tolerance float64 `json:"tolerance"`
	SampleSize int `json:"sampleSize"`
	StopOnFailure bool  `json:"stoponFailure"`
}