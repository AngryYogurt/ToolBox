package model

type Monitor struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Uuid      string `json:"uuid"`
	Type      string `json:"type"`
	JobUuid   string `json:"job_uuid"`
	JobType   string `json:"job_type"`
	JobResult string `json:"job_result"`
}
