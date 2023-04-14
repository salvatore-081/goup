package models

type AddVolumeToBackupBody struct {
	Name string `json:"name" binding:"required"`
}
