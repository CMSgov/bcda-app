package constants

//Error Constants
const DefaultError = "Some other error"
const SQLErr = "Some SQL error"
const NoACORecord = "no ACO record found for "
const SsasClientErr = "no client for SSAS; %s"
const TestChangeTimeErr = "Failed to change modified time for file"

//Messages Constants
const CompleteMedSupDataImp = "Completed 1-800-MEDICARE suppression data import."
const ReadingFileN = "Reading file %s from archive %s.\n"
const ReadingFile = "Reading file %s from archive %s"
const FileNotFoundN = "File %s not found in archive %s.\n"
const FileNotFound = "file %s not found in archive %s"
const TestACOName = "ACO name (--name) must be provided"
const TestReqErr = "Request error"
const InProgress = "In Progress"
const TestSomeTime = "some time"
const TestRouter = "Test router"

//Path Constants
const V1Path = "/api/v1/"
const V2Path = "/api/v2/"
const EOBExportPath = "ExplanationOfBenefit/$export"
const PatientEOBPath = "Patient/$export?_type=ExplanationOfBenefit"
const GroupExportPath = "Group/all/$export"
const PatientExportPath = "Patient/$export"
const ALRExportPath = "alr/$export"
const ExportPath = "$export"
const TestArchivepath = "../bcdaworker/data/test/archive"
const ServerPath = "%s/v1/"
const TestFHIRPath = "/v1/fhir"
const TestSynthMedFilesPath = "synthetic1800MedicareFiles/test/"
const TestFakePath = "/src/github.com/CMSgov/bcda-app/conf/FAKE"
const TestConfPath = "/src/github.com/CMSgov/bcda-app/conf/test"
const TestSuppressBadPath = "T#EFT.ON.ACO.NGD1800.FRPD.D191220.T1000009"
const JobsFilePath = "jobs/1"
const TokenPath = "/auth/token" // #nosec - G101 credentials for unit testing
const TestFilePathVariable = "%s/%d/%s"

//ID Constants
const TestACOID = "DBBD1CE1-AE24-435C-807D-ED45953077D3"
const TestTokenID = "665341c9-7d0c-4844-b66f-5910d9d0822f" // #nosec - G101 credentials for unit testing
//Url Constants
const TestAPIUrl = "https://www.api.com"
const ExpectedTestUrl = "http://example.com/data"

//Arguments Constants
const MockClient = "mock-client"
const CMSIDArg = "--cms-id"
const ThresholdArg = "--threshold"
const CleanupArchArg = "cleanup-archive"
const CreateGroupArg = "create-group"
const NameArg = "--name"
const ACOIDArg = "--aco-id"
const CreateACOID = "create-aco"
const GenClientCred = "generate-client-credentials"
const ResetClientCred = "reset-client-credentials" // #nosec - G101 credentials for unit testing
const ArchJobFiles = "archive-job-files"
const DelDirContents = "delete-dir-contents"
const ImportSupDir = "import-suppression-directory"
const DirectoryArg = "--directory"
const TestExcludeSAMHSA = "excludeSAMHSA=true"
const TestSvcDate = "service-date"
const FakeClientID = "fake-client-id"
const FakeClientIDBt = `"fake-client-id"`
const FakeSecret = "fake-secret"
const FakeSecretBt = `"fake-secret"`
const CacheControl = "Cache-Control"
const TestRespondAsync = "respond-async"

//SQL Constants
const TestSelectNowSQL = "SELECT NOW()"
const TestCountSQL = "COUNT(1)"

//Named Constants
const CCLF8CompPath = "cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000"
const CCLF8Name = "T.BCD.A0001.ZC8Y18.D181120.T1000009"
const TestFileTime = "2018-11-20T10:00:00Z"
const TestKeyName = "foo.pem"
const EmptyKeyName = "../static/emptyFile.pem"
const BadKeyName = "../static/badPublic.pem"
const TestBlobFileName = "blob.ndjson"
const TestSuppressMetaFileName = "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009"
const TestListData = "One,Two,Three,Four"
const TestSvcDateResult = "2006-01-02"
const TestScore = "1.2345"
const RegexACOID = "TEST\\d{4}"

const ExpiresInDefault = "1200"
