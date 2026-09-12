package handlers

import "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"

type PaginationParams struct {
	Page  int `form:"page" validate:"omitempty,min=1"`
	Limit int `form:"limit" validate:"omitempty,min=1,pagination_limit"`
}

func (p PaginationParams) ToDomain() domain.Pagination {
	return domain.Pagination{
		Page:  p.Page,
		Limit: p.Limit,
	}
}
