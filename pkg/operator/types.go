// Package operator defines types and reconciliation logic for the
// web crawler Kubernetes Operator. It manages CrawlJob and CrawlCluster
// custom resources.
package operator

import "time"

// CrawlJobSpec defines the desired state of a CrawlJob.
type CrawlJobSpec struct {
	SeedURLs             []string     `json:"seedURLs"`
	MaxDepth             int          `json:"maxDepth,omitempty"`
	Workers              int          `json:"workers,omitempty"`
	ConcurrencyPerWorker int          `json:"concurrencyPerWorker,omitempty"`
	MaxRetries           int          `json:"maxRetries,omitempty"`
	Image                string       `json:"image,omitempty"`
	Resources            ResourceSpec `json:"resources,omitempty"`
	Kafka                KafkaSpec    `json:"kafka,omitempty"`
	Schedule             string       `json:"schedule,omitempty"`
	Suspend              bool         `json:"suspend,omitempty"`
}

// CrawlJobStatus defines the observed state of a CrawlJob.
type CrawlJobStatus struct {
	Phase          JobPhase    `json:"phase,omitempty"`
	ActiveWorkers  int         `json:"activeWorkers,omitempty"`
	URLsProcessed  int64       `json:"urlsProcessed,omitempty"`
	URLsFailed     int64       `json:"urlsFailed,omitempty"`
	StartTime      *time.Time  `json:"startTime,omitempty"`
	CompletionTime *time.Time  `json:"completionTime,omitempty"`
	Conditions     []Condition `json:"conditions,omitempty"`
}

// JobPhase represents the lifecycle phase of a CrawlJob.
type JobPhase string

const (
	JobPhasePending   JobPhase = "Pending"
	JobPhaseRunning   JobPhase = "Running"
	JobPhasePaused    JobPhase = "Paused"
	JobPhaseCompleted JobPhase = "Completed"
	JobPhaseFailed    JobPhase = "Failed"
)

// Condition represents a status condition on a CrawlJob.
type Condition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// ResourceSpec defines CPU and memory limits/requests.
type ResourceSpec struct {
	Limits   ResourceValues `json:"limits,omitempty"`
	Requests ResourceValues `json:"requests,omitempty"`
}

// ResourceValues holds CPU and memory values.
type ResourceValues struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// KafkaSpec defines Kafka connection settings.
type KafkaSpec struct {
	Brokers   []string `json:"brokers,omitempty"`
	TaskTopic string   `json:"taskTopic,omitempty"`
}

// CrawlClusterSpec defines the desired state of a CrawlCluster.
type CrawlClusterSpec struct {
	Kafka              CrawlClusterKafka `json:"kafka"`
	DefaultWorkerImage string            `json:"defaultWorkerImage,omitempty"`
	Autoscaling        AutoscalingSpec   `json:"autoscaling,omitempty"`
	Monitoring         MonitoringSpec    `json:"monitoring,omitempty"`
}

// CrawlClusterKafka defines Kafka topics for the cluster.
type CrawlClusterKafka struct {
	Brokers      []string `json:"brokers"`
	TaskTopic    string   `json:"taskTopic,omitempty"`
	DLQTopic     string   `json:"dlqTopic,omitempty"`
	ControlTopic string   `json:"controlTopic,omitempty"`
}

// AutoscalingSpec configures worker autoscaling.
type AutoscalingSpec struct {
	Enabled          bool `json:"enabled,omitempty"`
	MinWorkers       int  `json:"minWorkers,omitempty"`
	MaxWorkers       int  `json:"maxWorkers,omitempty"`
	TargetQueueDepth int  `json:"targetQueueDepth,omitempty"`
}

// MonitoringSpec configures metrics and monitoring.
type MonitoringSpec struct {
	Enabled        bool `json:"enabled,omitempty"`
	MetricsPort    int  `json:"metricsPort,omitempty"`
	ServiceMonitor bool `json:"serviceMonitor,omitempty"`
}

// CrawlClusterStatus defines the observed state of a CrawlCluster.
type CrawlClusterStatus struct {
	Phase          string `json:"phase,omitempty"`
	ActiveJobs     int    `json:"activeJobs,omitempty"`
	TotalWorkers   int    `json:"totalWorkers,omitempty"`
	KafkaConnected bool   `json:"kafkaConnected,omitempty"`
}
