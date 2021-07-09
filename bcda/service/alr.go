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
	RunoutAlrRequest  AlrRequestType = 2
)

type AlrRequestWindow struct {
	LowerBound time.Time
	UpperBound time.Time
}

// Get the MBIs and put them into jobs
func (s *service) GetAlrJobs(ctx context.Context, cmsID string,
	reqType AlrRequestType, window AlrRequestWindow) ([]*models.JobAlrEnqueueArgs, error) {
	constraint, err := s.timeConstraints(ctx, cmsID)
	if err != nil {
		return nil, fmt.Errorf("failed to set time constraints: %w", err)
	}

	// Update the window based on any conditions set on the ACO
	window = s.getWindow(cmsID, window, constraint.claimsDate, reqType)

	fileType := models.FileTypeDefault
	if reqType == RunoutAlrRequest {
		fileType = models.FileTypeRunout
	}

	req := RequestConditions{
		CMSID:          cmsID,
		fileType:       fileType,
		timeConstraint: constraint,
	}

    // ALR at the current state still depends on CCLF file to get the attribution data
    // This may be removed in the future, where we relay on table 1-1
	benes, err := s.getBeneficiaries(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retreive beneficiaries: %w", err)
	}

	jobs := make([]*models.JobAlrEnqueueArgs, 0, len(benes)/int(s.alrMBIsPerJob))
	for {
		var part []*models.CCLFBeneficiary
		part, benes = partitionBenes(benes, s.alrMBIsPerJob) // default @ 1000 jobs
		if len(part) == 0 {
			break
		}

		job := &models.JobAlrEnqueueArgs{
			CMSID:      cmsID,
			LowerBound: window.LowerBound,
			UpperBound: window.UpperBound,
            BBBasePath: s.bbBasePath,
			MBIs:       make([]string, 0, s.alrMBIsPerJob),
		}

		for _, bene := range part {
			job.MBIs = append(job.MBIs, bene.MBI)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// getWindow returns an update request window based on any time constraints that are associated with the caller
func (s *service) getWindow(cmsID string, current AlrRequestWindow, claimsDate time.Time, req AlrRequestType) AlrRequestWindow {
	new := current
	if req == RunoutAlrRequest {
		new.UpperBound = s.rp.claimThruDate
	} else if !claimsDate.IsZero() {
		new.UpperBound = claimsDate
	}

	for pattern, cfg := range s.acoConfig {
		if pattern.MatchString(cmsID) && !cfg.LookbackTime().IsZero() {
			new.LowerBound = cfg.LookbackTime()
			break
		}
	}

	return new
}

func partitionBenes(input []*models.CCLFBeneficiary, size uint) (part, remaining []*models.CCLFBeneficiary) {
	if uint(len(input)) <= size {
		return input, nil
	}
	return input[:size], input[size:]
}
