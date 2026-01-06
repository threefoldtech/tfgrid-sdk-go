package models

// DeploymentAttempt is the GORM model for deployment_attempts table
type DeploymentAttempt struct {
	Time            int64   `gorm:"primaryKey;autoIncrement:false;column:time"`
	NodeID          int64   `gorm:"index:idx_node_time;index:idx_node_time_desc;column:node_id"`
	FarmID          int64   `gorm:"index:idx_farm_time;index:idx_farm_time_desc;column:farm_id"`
	Status          string  `gorm:"type:varchar(20);not null;column:status"`
	TotalDurationMs *int    `gorm:"type:integer;column:total_duration_ms"`
	ErrorCode       *string `gorm:"type:varchar(100);column:error_code"`
}

// TableName specifies the table name for GORM
func (DeploymentAttempt) TableName() string {
	return "deployment_attempts"
}
