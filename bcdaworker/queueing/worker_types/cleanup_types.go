package worker_types

const CleanupJobKind = "CleanupJob"

type CleanupJobArgs struct {
}

func (args CleanupJobArgs) Kind() string {
	return CleanupJobKind
}
