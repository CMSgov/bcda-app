package constants

const DevACOUUID = "0c527d2e-2e8a-4808-b11d-0fa06baf8254"
const SmallACOUUID = "3461C774-B48F-11E8-96F8-529269fb1459"
const MediumACOUUID = "C74C008D-42F8-4ED9-BF88-CEE659C7F692"
const LargeACOUUID = "8D80925A-027E-43DD-8AED-9A501CC4CD91"

const ImportInprog = "In-Progress"
const ImportComplete = "Completed"
const ImportFail = "Failed"

// This is set during compilation.  See build_and_package.sh in the /ops dir
var Version = "latest"

const Adjudicated = "adjudicated"
const PartiallyAdjudicated = "partially-adjudicated"

// Cli.go constants
const CliCMSIDArg = "cms-id"
const CliCMSIDDesc = "CMS ID of ACO"
const CliArchDesc = "How long files should wait in archive before deletion"
const CliRemoveArchDesc = "Remove job directory and files from archive and update job status to Expired"
const CliAuthToolsCategory = "Authentication tools"
const CliDataImpCategory = "Data import"

const ContentType = "Content-Type"
const JsonContentType = "application/json"
const FHIRJsonContentType = "application/fhir+json"

const BBHeaderTS = "BlueButton-OriginalQueryTimestamp"
const BBHeaderOriginURL = "BlueButton-OriginalUrl"
const BBHeaderOriginQID = "BlueButton-OriginalQueryId"
const BBHeaderOriginQ = "BlueButton-OriginalQuery"
const BBHeaderOriginQC = "BlueButton-OriginalQueryCounter"

const CCLFFileRetID = "%s RETURNING id"
const JobKeyCreateErr = "failed to create job key: %w"

const JOBIDPath = "/jobs/{jobID}"

const IssuerSSAS = "ssas"

const EmptyString = ""

const FiveSeconds = "5"
const FiveHundredSeconds = "500"

const CCLF8FileNum = int(8)

const BFDV1Path = "/v1/fhir"
const BFDV2Path = "/v2/fhir"
const BFDV3Path = "/v3/fhir" // TODO: V3
const V3Version = "demo"

const GetExistingBenes = "GetExistingBenes"
const GetNewAndExistingBenes = "GetNewAndExistingBenes"
