package main

import (
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/tests/seed/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/tests/seed/repository"
)

type demoCampaignSpec = repository.CampaignSpec

func DemoCampaignSpecs() []demoCampaignSpec {
	return []demoCampaignSpec{
		{
			Name:           "running-mid",
			DeviceModel:    config.DemoModelRunning,
			BaseVersion:    config.DemoRunningBaseVersion,
			TargetVersion:  config.DemoRunningTargetVersion,
			Status:         "running",
			SeedRunningSet: true,
			StageStatuses:  []string{"passed", "active", "pending"},
			TargetPercents: []int{10, 50, 100},
			NumDevices:     config.DemoDevicesPerCampaign,
			NumAttempts:    config.DemoAttemptsPerCampaign,
		},
		{
			Name:           "paused",
			DeviceModel:    config.DemoModelPaused,
			BaseVersion:    config.DemoPausedBaseVersion,
			TargetVersion:  config.DemoPausedTargetVersion,
			Status:         "paused",
			SeedRunningSet: false,
			StageStatuses:  []string{"passed", "active", "pending"},
			TargetPercents: []int{10, 50, 100},
			NumDevices:     config.DemoDevicesPerCampaign,
			NumAttempts:    config.DemoAttemptsPerCampaign,
		},
		{
			Name:           "draft",
			DeviceModel:    config.DemoModelDraft,
			BaseVersion:    config.DemoDraftBaseVersion,
			TargetVersion:  config.DemoDraftTargetVersion,
			Status:         "draft",
			SeedRunningSet: false,
			StageStatuses:  []string{"pending", "pending", "pending"},
			TargetPercents: []int{10, 50, 100},
			NumDevices:     config.DemoDevicesPerCampaign,
			NumAttempts:    0,
		},
		{
			Name:           "completed",
			DeviceModel:    config.DemoModelCompleted,
			BaseVersion:    config.DemoCompletedBaseVersion,
			TargetVersion:  config.DemoCompletedTargetVersion,
			Status:         "completed",
			SeedRunningSet: false,
			StageStatuses:  []string{"passed", "passed", "passed"},
			TargetPercents: []int{10, 50, 100},
			NumDevices:     config.DemoDevicesPerCampaign,
			NumAttempts:    config.DemoAttemptsPerCampaign,
		},
		{
			Name:           "rolled-back",
			DeviceModel:    config.DemoModelRolledBack,
			BaseVersion:    config.DemoRolledBackBaseVersion,
			TargetVersion:  config.DemoRolledBackTargetVersion,
			Status:         "rolled_back",
			SeedRunningSet: false,
			StageStatuses:  []string{"passed", "failed", "pending"},
			TargetPercents: []int{10, 50, 100},
			NumDevices:     config.DemoDevicesPerCampaign,
			NumAttempts:    config.DemoAttemptsPerCampaign,
		},
	}
}

func LoadCampaignModels() []string {
	models := make([]string, config.NumLoadCampaigns)

	for i := range models {
		models[i] = fmt.Sprintf("%s-%03d", config.DeviceModel, i+1)
	}

	return models
}

func AllSeedModels() []string {
	models := append([]string{config.DemoModelOrphan}, LoadCampaignModels()...)

	for _, spec := range DemoCampaignSpecs() {
		models = append(models, spec.DeviceModel)
	}

	return models
}
