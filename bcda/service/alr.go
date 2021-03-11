package service

import (
	"context"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
)

type AlrRequestType uint8

const (
	DefaultAlrRequest AlrRequestType = iota
	RunoutAlrRequest
)

type AlrRequestWindow struct {
	LowerBound time.Time
	UpperBound time.Time
}

func (s *service) CreateAlrJobs(ctx context.Context, cmsID string, reqType AlrRequestType, window AlrRequestWindow) ([]*models.JobAlrEnqueueArgs, error) {
	constraint, err := s.timeConstraints(ctx, cmsID)
	if err != nil {
		return nil, fmt.Errorf("failed to set time constraints: %w", err)
	}

	fileType := models.FileTypeDefault
	if reqType == RunoutAlrRequest {
		fileType = models.FileTypeRunout
	}

	req := RequestConditions{
		CMSID:          cmsID,
		fileType:       fileType,
		timeConstraint: constraint,
	}
	benes, err := s.getBeneficiaries(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retreive beneficiaries: %w", err.Error())
	}


	return nil, nil
}

