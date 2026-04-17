package models

// K8sCluster represents a Kubernetes cluster.
type K8sCluster struct {
	BaseModel
	ClusterName string `gorm:"size:100;not null" json:"cluster_name"`
	Status      string `gorm:"size:20;not null;default:'pending'" json:"status"`
	Version     string `gorm:"size:20" json:"version"`
	MasterNode  string `gorm:"size:255" json:"master_node"`
	WorkerNodes JSONB  `gorm:"type:jsonb;not null;default:'[]'" json:"worker_nodes"`
	Config      JSONB  `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	Description string `gorm:"type:text" json:"description"`
}

func (K8sCluster) TableName() string {
	return "k8s_clusters"
}
