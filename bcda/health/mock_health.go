package health

type MockHealthChecker struct {
	DbOk         bool
	WorkerOk     bool
	BbOk         bool
	SsasOk       bool
	IntrospectOk bool
}

func (m MockHealthChecker) IsDatabaseOK() (string, bool) {
	return "", m.DbOk
}

func (m MockHealthChecker) IsWorkerDatabaseOK() (string, bool) {
	return "", m.WorkerOk
}

func (m MockHealthChecker) IsBlueButtonOK() bool {
	return m.BbOk
}

func (m MockHealthChecker) IsSsasOK() (string, bool) {
	return "", m.SsasOk
}

func (m MockHealthChecker) IsSsasIntrospectOK() (string, bool) {
	return "", m.IntrospectOk
}
