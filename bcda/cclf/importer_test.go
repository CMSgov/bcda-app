package cclf

// import (
// 	"context"
// 	"database/sql"
// 	"fmt"
// 	"math/rand"
// 	"regexp"
// 	"testing"

// 	"github.com/CMSgov/bcda-app/bcda/models"
// 	"github.com/pkg/errors"

// 	"github.com/sirupsen/logrus"

// 	"github.com/stretchr/testify/assert"

// 	"github.com/DATA-DOG/go-sqlmock"
// 	"github.com/stretchr/testify/suite"
// )

// type ImporterTestSuite struct {
// 	suite.Suite

// 	tx   *sql.Tx
// 	mock sqlmock.Sqlmock

// 	db *sql.DB
// }

// func (s *ImporterTestSuite) SetupTest() {
// 	var err error
// 	s.db, s.mock, err = sqlmock.New()
// 	assert.NoError(s.T(), err)

// 	s.mock.ExpectBegin()
// 	tx, err := s.db.Begin()
// 	assert.NoError(s.T(), err)

// 	s.tx = tx
// }

// func (s *ImporterTestSuite) AfterTest(_, _ string) {
// 	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
// 	s.db.Close()
// }

// func TestMetricTestSuite(t *testing.T) {
// 	suite.Run(t, new(ImporterTestSuite))
// }

// // TestCCLF8ImporterHappyPath verifies that we're able to flush statements
// // that have exceeded the query threshold
// func (s *ImporterTestSuite) TestCCLF8ImporterHappyPath() {
// 	var benes []*models.CCLFBeneficiary
// 	fileID := uint(rand.Uint32())
// 	for i := 0; i < 100; i++ {
// 		benes = append(benes, getBeneficiary(fileID))
// 	}

// 	tests := []struct {
// 		name              string
// 		maxPendingQueries int
// 	}{
// 		{"NoFlush", 100000},
// 		{"InProgressFlush", 7},
// 		{"SingleExecPerFlush", 1},
// 	}

// 	for _, tt := range tests {
// 		s.T().Run(tt.name, func(t *testing.T) {
// 			s.SetupTest()
// 			defer s.AfterTest("", "")

// 			importer := &cclf8Importer{
// 				logger:            logrus.New(),
// 				maxPendingQueries: tt.maxPendingQueries,
// 			}

// 			execCount := 0
// 			prepare := s.mock.ExpectPrepare(regexp.QuoteMeta(`COPY "cclf_beneficiaries" ("file_id", "mbi")`))
// 			for _, bene := range benes {
// 				mbi := string([]byte(bene.MBI)[0:11])
// 				cclfBeneficiary := models.CCLFBeneficiary{
// 					FileID: fileID,
// 					MBI:    mbi,
// 				}
// 				prepare.ExpectExec().WithArgs(bene.FileID, mbi).WillReturnResult(sqlmock.NewResult(1, 1))

// 				err := importer.do(context.Background(), s.tx, cclfBeneficiary)
// 				assert.NoError(t, err)

// 				execCount++
// 				// The last bene import call will not close out the in progress statement.
// 				if execCount%tt.maxPendingQueries == 0 && execCount < len(benes) {
// 					prepare.ExpectExec().WithArgs().WillReturnResult(sqlmock.NewResult(1, 1))
// 					prepare.WillBeClosed()
// 					prepare = s.mock.ExpectPrepare(regexp.QuoteMeta(`COPY "cclf_beneficiaries" ("file_id", "mbi")`))
// 				}
// 			}

// 			// Calling flush should close out the pending prepared statement
// 			prepare.ExpectExec().WithArgs().WillReturnResult(sqlmock.NewResult(1, 1))
// 			prepare.WillBeClosed()
// 			importer.flush(context.Background())
// 		})
// 	}
// }

// func (s *ImporterTestSuite) TestCCLF8ImporterErrorPaths() {
// 	type errorType int
// 	const (
// 		execWithArgsError errorType = 1
// 		execEmptyError    errorType = 2
// 		closeError        errorType = 3
// 		doErrorOnFlush    errorType = 4
// 	)

// 	type doError struct {
// 		et  errorType
// 		err error
// 	}

// 	fileID := uint(rand.Uint32())
// 	bene := getBeneficiary(fileID)

// 	tests := []struct {
// 		name string
// 		err  doError
// 	}{
// 		{"ErrorOnExecWithArgs", doError{execWithArgsError, errors.New("Some error when exec call with bene args")}},
// 		{"ErrorOnExecEmpty", doError{execEmptyError, errors.New("Some exec error when attempting to flush statement")}},
// 		{"ErrorOnClose", doError{closeError, errors.New("Some exec error when attempting to close statement")}},
// 		{"ErrorOnFlushOnDo", doError{doErrorOnFlush, errors.New("Some exec error when attempting to flush statement")}},
// 	}

// 	for _, tt := range tests {
// 		s.T().Run(tt.name, func(t *testing.T) {
// 			s.SetupTest()
// 			defer s.AfterTest("", "")

// 			importer := &cclf8Importer{
// 				logger:            logrus.New(),
// 				maxPendingQueries: 1,
// 			}

// 			prepare := s.mock.ExpectPrepare(regexp.QuoteMeta(`COPY "cclf_beneficiaries" ("file_id", "mbi")`))
// 			mbi := string([]byte(bene.MBI)[0:11])
// 			cclfBeneficiary := models.CCLFBeneficiary{
// 				FileID: fileID,
// 				MBI:    mbi,
// 			}

// 			execWithArgs := prepare.ExpectExec().WithArgs(bene.FileID, mbi)
// 			execNoArgs := prepare.ExpectExec().WithArgs()

// 			switch tt.err.et {
// 			case execWithArgsError:
// 				execWithArgs.WillReturnError(tt.err.err)
// 			case execEmptyError:
// 				// Need to ensure that the exec call with args succeeds to ensure that we hit the failure on the exec with no args
// 				execWithArgs.WillReturnResult(sqlmock.NewResult(1, 1))
// 				execNoArgs.WillReturnError(tt.err.err)
// 			case closeError:
// 				execWithArgs.WillReturnResult(sqlmock.NewResult(1, 1))
// 				execNoArgs.WillReturnResult(sqlmock.NewResult(1, 1))
// 				prepare.WillReturnCloseError(tt.err.err)
// 			case doErrorOnFlush:
// 				execWithArgs.WillReturnResult(sqlmock.NewResult(1, 1))
// 				execNoArgs.WillReturnResult(sqlmock.NewResult(1, 1))
// 				prepare.WillReturnCloseError(tt.err.err)

// 				// Run an import call, this'll force the next importer.do call to
// 				// fail on the attempt to flush (which will fail)
// 				err := importer.do(context.Background(), s.tx, cclfBeneficiary)
// 				assert.NoError(t, err)
// 			}

// 			errOnDo := importer.do(context.Background(), s.tx, cclfBeneficiary)
// 			errOnFlush := importer.flush(context.Background())

// 			switch tt.err.et {
// 			case execWithArgsError:
// 				assert.Contains(t, errOnDo.Error(), "could not create CCLF8 beneficiary record")
// 				assert.Contains(t, errors.Cause(errOnDo).Error(), tt.err.err.Error())
// 			case execEmptyError, closeError:
// 				assert.NoError(t, errOnDo)
// 				assert.Contains(t, errOnFlush.Error(), tt.err.err.Error())
// 			case doErrorOnFlush:
// 				assert.Contains(t, errOnDo.Error(), "failed to flush statement")
// 				assert.Contains(t, errors.Cause(errOnDo).Error(), tt.err.err.Error())
// 			}

// 		})
// 	}
// }

// func (s *ImporterTestSuite) TestFlushOnNoExistingStatement() {
// 	importer := &cclf8Importer{
// 		logger:            logrus.New(),
// 		maxPendingQueries: 10,
// 	}
// 	assert.NoError(s.T(), importer.flush(context.Background()))
// }

// func getBeneficiary(fileID uint) *models.CCLFBeneficiary {
// 	return &models.CCLFBeneficiary{
// 		FileID: fileID,
// 		// We expect 11 bytes for MBI - we'll ensure that each of the values are AT LEAST 11 bytes
// 		MBI: fmt.Sprintf("MBI%08d", rand.Uint64()),
// 	}
// }
