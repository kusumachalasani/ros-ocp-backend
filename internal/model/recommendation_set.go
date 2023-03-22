package model

import (
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	database "github.com/redhatinsights/ros-ocp-backend/internal/db"
)

type RecommendationSet struct {
	ID                     string `gorm:"primaryKey;not null;autoIncrement"`
	WorkloadID             uint
	Workload               Workload  `gorm:"foreignKey:WorkloadID"`
	MonitoringStartTime    time.Time `gorm:"type:timestamp"`
	MonitoringEndTime      time.Time `gorm:"type:timestamp"`
	Recommendations        datatypes.JSON
	CreatedAt              time.Time `gorm:"type:timestamp"`
	MonitoringStartTimeStr string    `gorm:"-"`
	MonitoringEndTimeStr   string    `gorm:"-"`
	CreatedAtStr           string    `gorm:"-"`
}

func (r *RecommendationSet) AfterFind(tx *gorm.DB) error {
	r.MonitoringStartTimeStr = r.MonitoringStartTime.Format(time.RFC3339)
	r.MonitoringEndTimeStr = r.MonitoringEndTime.Format(time.RFC3339)
	r.CreatedAtStr = r.CreatedAt.Format(time.RFC3339)
	return nil
}

func (r *RecommendationSet) GetRecommendationSets(orgID string, limit int, offset int, queryParams map[string]interface{}) ([]RecommendationSet, error) {

	var recommendationSets []RecommendationSet
	db := database.GetDB()

	query := db.Joins(`
		JOIN (
			SELECT workload_id, MAX(monitoring_end_time) AS latest_monitoring_end_time 
			FROM recommendation_sets GROUP BY workload_id
		) latest_rs ON recommendation_sets.workload_id = latest_rs.workload_id 
				AND recommendation_sets.monitoring_end_time = latest_rs.latest_monitoring_end_time
			JOIN workloads ON recommendation_sets.workload_id = workloads.id
			JOIN clusters ON workloads.cluster_id = clusters.id
			JOIN rh_accounts ON clusters.tenant_id = rh_accounts.id
		`).Preload("Workload.Cluster").
		Where("rh_accounts.org_id = ?", orgID)

	var clusterKey string
	for key, value := range queryParams {
		if strings.Contains(key, "clusters") {
			clusterKey = key
			query.Where(key, value).Or("clusters.cluster_uuid LIKE ?", value)
		}
	}

	delete(queryParams, clusterKey)

	for key, value := range queryParams {
		query.Where(key, value)
	}

	err := query.Offset(offset).Limit(limit).Find(&recommendationSets).Error

	if err != nil {
		return nil, err
	}

	return recommendationSets, nil
}

func (r *RecommendationSet) GetRecommendationSetByID(orgID string, recommendationID string) (RecommendationSet, error) {

	var recommendationSet RecommendationSet
	db := database.GetDB()

	db.Joins("JOIN workloads ON recommendation_sets.workload_id = workloads.id").
		Joins("JOIN clusters ON workloads.cluster_id = clusters.id").
		Joins("JOIN rh_accounts ON clusters.tenant_id = rh_accounts.id").
		Preload("Workload.Cluster").
		Where("rh_accounts.org_id = ?", orgID).
		Where("recommendation_sets.id = ?", recommendationID).
		First(&recommendationSet)

	return recommendationSet, nil
}

func (r *RecommendationSet) CreateRecommendationSet() error {
	db := database.GetDB()
	result := db.Create(r)

	if result.Error != nil {
		return result.Error
	}

	return nil
}