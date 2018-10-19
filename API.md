# Beneficiary Claims Data API
The Beneficiary Claims Data API (BCDA) enables Accountable Care Organizations (ACOs) to retrieve claims data for their Medicare beneficiaries. This includes claims data for instances in which beneficiaries receive care outside of the ACO, allowing a full picture of patient care.

This API follows the workflow outlined by the [FHIR Bulk Data Export Proposal](https://github.com/smart-on-fhir/fhir-bulk-data-docs/blob/master/export.md), using the [HL7 FHIR Standard](https://www.hl7.org/fhir/). Claims are provided as FHIR [Bundles](https://www.hl7.org/fhir/bundle.html) of [ExplanationOfBenefit](https://www.hl7.org/fhir/explanationofbenefit.html) resources, in [NDJSON](http://ndjson.org/) format.

## Getting Started

### APIs
Not familiar with APIs? Here are some great introductions:
* [Introduction to Web APIs](https://developer.mozilla.org/en-US/docs/Learn/JavaScript/Client-side_web_APIs/Introduction)
* [An Intro to APIs](https://www.codenewbie.org/blogs/an-intro-to-apis)

### Authentication and Authorization
An access token is required for most requests. The token is presented in API requests in the `Authorization` header as a `Bearer` token. The process of token distribution is to be determined.

### Environment
The examples below may be followed using any tool that can make HTTP GET requests with headers, such as [Postman](https://www.getpostman.com/) or [cURL](https://curl.haxx.se/).

## Examples

### BCDA Metadata
Metadata about the Beneficiary Claims Data API is available as a FHIR [CapabilityStatement](https://www.hl7.org/fhir/capabilitystatement.html) resource. A token is not required to access this information.

#### 1. Request the metadata

##### Request
`GET /api/v1/metadata`

##### Response
```json

{
"ResourceType": "CapabilityStatement"
}
```

### Beneficiary ExplanationOfBenefit Data

#### 1. Obtain an access token
See [Authentication and Authorization](#authentication-and-authorization) above.

#### 2. Initiate an export job

#### 3. Check the status of the export job

#### 4. Retrieve the NDJSON output file(s)
