package sensu

type Message struct {
	Labels      Labels
	Annotations Annotations
	StartsAt    string
}

type Labels struct {
	Client   string
	Check    string
	Severity string
}

type Annotations struct {
	Command  string
	Issued   int64
	Executed int64
	Duration float64
	Output   string
	Status   int
	Ves      string
	StartsAt string
}

type HealthCheckOutput []struct {
	Service   string
	Container int64
	Status    string
	Healthy   int64
}
