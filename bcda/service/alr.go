package service

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/models"
)

// Get the MBIs and put them into jobs
func (s *service) GetAlrJobs(ctx context.Context, alrMBI *models.AlrMBIs) []*models.JobAlrEnqueueArgs {

	partition := int(s.alrMBIsPerJob)

	loop := len(alrMBI.MBIS) / partition

	bigJob := []*models.JobAlrEnqueueArgs{}

	for i := 0; i < loop; i++ {
		bigJob = append(bigJob, &models.JobAlrEnqueueArgs{
			CMSID:           alrMBI.CMSID,
			MBIs:            alrMBI.MBIS[:partition],
			BBBasePath:      s.bbBasePath,
			MetaKey:         alrMBI.Metakey,
			TransactionTime: alrMBI.TransactionTime,
		})
		// push the slice
		alrMBI.MBIS = alrMBI.MBIS[partition:]
	}
	if len(alrMBI.MBIS)%partition != 0 {
		// There are more MBIs, unless than the partition
		bigJob = append(bigJob, &models.JobAlrEnqueueArgs{
			CMSID:           alrMBI.CMSID,
			MBIs:            alrMBI.MBIS,
			MetaKey:         alrMBI.Metakey,
			BBBasePath:      s.bbBasePath,
			TransactionTime: alrMBI.TransactionTime,
		})
	}

	return bigJob
}

func partitionBenes(input []*models.CCLFBeneficiary, size uint) (part, remaining []*models.CCLFBeneficiary) {
	if uint(len(input)) <= size {
		return input, nil
	}
	return input[:size], input[size:]
}
