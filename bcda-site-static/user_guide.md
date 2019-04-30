---
layout: home
title:  "Beneficiary Claims Data API User Guide"
date:   2017-10-30 09:21:12 -0500
description: Information on accessing and working with the API
landing-page: live
gradient: "blueberry-lime-background"
subnav-link-gradient: "blueberry-lime-link"
sections:
  - User Guide
  - About APIs
  - Authentication and Authorization
  - Encryption
  - Environment
  - Examples
  - BCDA Metadata
  - Beneficiary Explanation of Benefit Data
  - Beneficiary Patient Data
  - Beneficiary Coverage Data

ctas:

  - title: Learn about encryption
    link: ./encryption.html
  - title: Join the BCDA Google Group
    link: https://groups.google.com/forum/#!forum/bc-api
    target: _blank
    
---

# User Guide
The [Beneficiary Claims Data API (BCDA)](https://sandbox.bcda.cms.gov) enables Accountable Care Organizations (ACOs) to retrieve claims data for their Medicare beneficiaries. This includes claims data for instances in which beneficiaries receive care outside of the ACO, allowing a full picture of patient care.

While the BCDA sandbox is currently focused on the needs of Medicare Shared Savings Program ACOs, all are invited to use the sandbox and share their feedback.

This API follows the workflow outlined by the [FHIR Bulk Data Export Proposal](https://github.com/smart-on-fhir/fhir-bulk-data-docs/blob/master/export.md){:target="_blank"}, using the [HL7 FHIR Standard](https://www.hl7.org/fhir/){:target="_blank"}. Claims data is provided as FHIR resources in [NDJSON](http://ndjson.org/){:target="_blank"} format.

This guide serves as a starting point for users to begin working with the API. [Comprehensive Swagger documentation about all BCDA endpoints is available in the sandbox environment.](./api/v1/swagger/)


## About APIs
Not familiar with APIs? Here are some great introductions:
* [Introduction to Web APIs](https://developer.mozilla.org/en-US/docs/Learn/JavaScript/Client-side_web_APIs/Introduction){:target="_blank"}
* [An Intro to APIs](https://www.codenewbie.org/blogs/an-intro-to-apis){:target="_blank"}

## Authentication and Authorization
The Beneficiary Claims Data API is currently accessible as an open sandbox environment, which returns sample NDJSON files with synthetic beneficiary data. You can use the generic credentials below to view our implementation of the API, write a process for decrypting the payload, and learn the shape of the data before working with production files that include PII and PHI. There is no beneficiary PII or PHI in the files you can access via the sandbox.

To get a token that can be used with protected endpoints, `POST` the following credentials using [Basic Authentication](https://en.wikipedia.org/wiki/Basic_access_authentication){:target="_blank"} to `https://sandbox.bcda.cms.gov/auth/token`:

Client ID: 
{%- capture client_id -%}
09869a7f-46ce-4908-a914-6129d080a2ae
{%- endcapture -%}

{% include copy_snippet.md code=client_id %}

Client Secret: 
{%- capture client_secret -%}
64916fe96f71adc79c5735e49f4e72f18ff941d0dd62cf43ee1ae0857e204f173ba10e4250c12c48
{%- endcapture -%}

{% include copy_snippet.md code=client_secret %}

Encoded Basic authentication:
{%- capture auth_header -%}
Basic MDk4NjlhN2YtNDZjZS00OTA4LWE5MTQtNjEyOWQwODBhMmFlOjY0OTE2ZmU5NmY3MWFkYzc5YzU3MzVlNDlmNGU3MmYxOGZmOTQxZDBkZDYyY2Y0M2VlMWFlMDg1N2UyMDRmMTczYmExMGU0MjUwYzEyYzQ4
{%- endcapture -%}

{% include copy_snippet.md code=auth_header %}

##### Request
`POST /auth/token`

###### Headers
* `Accept: application/json`
* `Authorization: <Encoded Basic authentication>`



###### cURL command
``` sh
curl -X POST "http://sandbox.bcda.cms.gov/auth/token" -H "accept: application/json" -H "authorization: Basic MDk4NjlhN2YtNDZjZS00OTA4LWE5MTQtNjEyOWQwODBhMmFlOj\
Y0OTE2ZmU5NmY3MWFkYzc5YzU3MzVlNDlmNGU3MmYxOGZmOTQx\
ZDBkZDYyY2Y0M2VlMWFlMDg1N2UyMDRmMTczYmExMGU0MjUwYzEyYzQ4"
```

##### Response
You will receive a `200 OK` response and an access token if your credentials were accepted.


## Encryption
All data files are encrypted. Learn more about the [encryption strategy](./encryption.html).

## Environment
The examples below include [cURL](https://curl.haxx.se/){:target="_blank"} commands, but may be followed using any tool that can make HTTP GET requests with headers, such as [Postman](https://www.getpostman.com/){:target="_blank"}.

# Examples
Examples are shown as requests to the BCDA sandbox environment.

## BCDA Metadata
Metadata about the Beneficiary Claims Data API is available as a FHIR [CapabilityStatement](https://www.hl7.org/fhir/capabilitystatement.html){:target="_blank"} resource. A token is not required to access this information.

#### 1. Request the metadata

##### Request
`GET /api/v1/metadata`

###### cURL command
{%- capture code -%}
curl https://sandbox.bcda.cms.gov/api/v1/metadata
{%- endcapture -%}

{% include copy_snippet.md code=code %}


##### Response
```json
{
	"resourceType": "CapabilityStatement",
	"status": "active",
	"date": "2018-11-26",
	"publisher": "Centers for Medicare \u0026 Medicaid Services",
	"kind": "capability",
	"instantiates": ["https://fhir.backend.bluebutton.hhsdevcloud.us/baseDstu3/metadata/"],
	"software": {
		"name": "Beneficiary Claims Data API",
		"version": "latest",
		"releaseDate": "2018-11-26"
	},
	"implementation": {
		"url": "https://sandbox.bcda.cms.gov"
	},
	"fhirVersion": "3.0.1",
	"acceptUnknown": "extensions",
	"format": ["application/json", "application/fhir+json"],
	"rest": [{
		"mode": "server",
		"security": {
			"cors": true,
			"service": [{
				"coding": [{
					"system": "http://hl7.org/fhir/ValueSet/restful-security-service",
					"code": "OAuth",
					"display": "OAuth"
				}],
				"text": "OAuth"
			}, {
				"coding": [{
					"system": "http://hl7.org/fhir/ValueSet/restful-security-service",
					"code": "SMART-on-FHIR",
					"display": "SMART-on-FHIR"
				}],
				"text": "SMART-on-FHIR"
			}]
		},
		"interaction": [{
			"code": "batch"
		}, {
			"code": "search-system"
		}],
		"operation": [{
			"name": "export",
			"definition": {
				"reference": "https://sandbox.bcda.cms.gov/api/v1/ExplanationOfBenefit/$export"
			}
		}, {
			"name": "jobs",
			"definition": {
				"reference": "https://sandbox.bcda.cms.gov/api/v1/jobs/[jobID]"
			}
		}, {
			"name": "metadata",
			"definition": {
				"reference": "https://sandbox.bcda.cms.gov/api/v1/metadata"
			}
		}, {
			"name": "version",
			"definition": {
				"reference": "https://sandbox.bcda.cms.gov/_version"
			}
		}, {
			"name": "data",
			"definition": {
				"reference": "https://sandbox.bcda.cms.gov/data/[jobID]/[random_UUID].ndjson"
			}
		}]
	}]
}
```

## Beneficiary Explanation of Benefit Data

#### 1. Obtain an access token
See [Authentication and Authorization](#authentication-and-authorization){:target="_blank"} above.

#### 2. Initiate an export job

##### Request
`GET /api/v1/ExplanationOfBenefit/$export`

To start an explanation of benefit data export job, a GET request is made to the ExplanationOfBenefit export endpoint. An access token as well as `Accept` and `Prefer` headers are required.

The dollar sign ('$') before the word "export" in the URL indicates that the endpoint is an action rather than a resource. The format is defined by the [FHIR Bulk Data Export spec](https://github.com/smart-on-fhir/fhir-bulk-data-docs/blob/master/export.md){:target="_blank"}.

###### Headers
* `Authorization: Bearer {token}`
* `Accept: application/fhir+json`
* `Prefer: respond-async`

###### cURL command
```sh
curl -v https://sandbox.bcda.cms.gov/api/v1/ExplanationOfBenefit/\$export \
-H 'Authorization: Bearer {token}' \
-H 'Accept: application/fhir+json' \
-H 'Prefer: respond-async'
```

##### Response
If the request was successful, a `202 Accepted` response code will be returned and the response will include a `Content-Location` header. The value of this header indicates the location to check for job status and outcome. In the example header below, the number 42 in the URL represents the ID of the export job.

###### Headers
* `Content-Location: https://sandbox.bcda.cms.gov/api/v1/jobs/42`

#### 3. Check the status of the export job

##### Request
`GET https://sandbox.bcda.cms.gov/api/v1/jobs/42`

Using the `Content-Location` header value from the ExplanationOfBenefit data export response, you can check the status of the export job. The status will change from `202 Accepted` to `200 OK` when the export job is complete and the data is ready to be downloaded.

###### Headers
* `Authorization: Bearer {token}`

###### cURL Command
```sh
curl -v https://sandbox.bcda.cms.gov/api/v1/jobs/42 \
-H 'Authorization: Bearer {token}'
```

##### Responses
* `202 Accepted` indicates that the job is processing. Headers will include `X-Progress: In Progress`
* `200 OK` indicates that the job is complete. Below is an example of the format of the response body.
```json
{
  "transactionTime": "2018-10-19T14:47:33.975024Z",
  "request": "https://sandbox.bcda.cms.gov/api/v1/ExplanationOfBenefit/$export",
  "requiresAccessToken": true,
  "output": [
    {
      "type": "ExplanationOfBenefit",
      "url": "https://sandbox.bcda.cms.gov/data/42/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson"
    }
  ],
  "error": [
    {
      "type": "OperationOutcome",
      "url": "https://sandbox.bcda.cms.gov/data/42/DBBD1CE1-AE24-435C-807D-ED45953077D3-error.ndjson"
    }
  ]
}
```
Claims data can be found at the URLs within the `output` field. The number 42 in the data file URLs is the same job ID from the `Content-Location` header URL in previous step. If some of the data cannot be exported due to errors, details of the errors can be found at the URLs in the `error` field. The errors are provided in an NDJSON file as FHIR [OperationOutcome](https://www.hl7.org/fhir/operationoutcome.html){:target="_blank"} resources.

#### 4. Retrieve the NDJSON output file(s)
To obtain the exported explanation of benefit data, a GET request is made to the output URLs in the job status response when the job reaches the Completed state. The data will be presented as an NDJSON file of [ExplanationOfBenefit](https://www.hl7.org/fhir/explanationofbenefit.html){:target="_blank"} resources.

##### Request
`GET https://sandbox.bcda.cms.gov/data/42/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson`

###### Headers
* `Authorization: Bearer {token}`

###### cURL command
```sh
curl https://sandbox.bcda.cms.gov/data/42/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson \
-H 'Authorization: Bearer {token}'
```

##### Response
The response will be the requested data as FHIR ExplanationOfBenefit resources in NDJSON format, encrypted. An unencrypted example of one such resource is shown below.
```json
{
  "status":"active",
  "diagnosis":[
    {
      "diagnosisCodeableConcept":{
        "coding":[
          {
            "system":"http://hl7.org/fhir/sid/icd-9-cm",
            "code":"2113"
          }
        ]
      },
      "sequence":1,
      "type":[
        {
          "coding":[
            {
              "system":"https://bluebutton.cms.gov/resources/codesystem/diagnosis-type",
              "code":"principal",
              "display":"The single medical diagnosis that is most relevant to the patient's chief complaint or need for treatment."
            }
          ]
        }
      ]
    }
  ],
  "id":"carrier-10300336722",
  "payment":{
    "amount":{
      "system":"urn:iso:std:iso:4217",
      "code":"USD",
      "value":250.0
    }
  },
  "resourceType":"ExplanationOfBenefit",
  "billablePeriod":{
    "start":"2000-10-01",
    "end":"2000-10-01"
  },
  "extension":[
    {
      "valueMoney":{
        "system":"urn:iso:std:iso:4217",
        "code":"USD",
        "value":0.0
      },
      "url":"https://bluebutton.cms.gov/resources/variables/prpayamt"
    },
    {
      "valueIdentifier":{
        "system":"https://bluebutton.cms.gov/resources/variables/carr_num",
        "value":"99999"
      },
      "url":"https://bluebutton.cms.gov/resources/variables/carr_num"
    },
    {
      "valueCoding":{
        "system":"https://bluebutton.cms.gov/resources/variables/carr_clm_pmt_dnl_cd",
        "code":"1",
        "display":"Physician/supplier"
      },
      "url":"https://bluebutton.cms.gov/resources/variables/carr_clm_pmt_dnl_cd"
    }
  ],
  "type":{
    "coding":[
      {
        "system":"https://bluebutton.cms.gov/resources/variables/nch_clm_type_cd",
        "code":"71",
        "display":"Local carrier non-durable medical equipment, prosthetics, orthotics, and supplies (DMEPOS) claim"
      },
      {
        "system":"https://bluebutton.cms.gov/resources/codesystem/eob-type",
        "code":"CARRIER"
      },
      {
        "system":"http://hl7.org/fhir/ex-claimtype",
        "code":"professional",
        "display":"Professional"
      },
      {
        "system":"https://bluebutton.cms.gov/resources/variables/nch_near_line_rec_ident_cd",
        "code":"O",
        "display":"Part B physician/supplier claim record (processed by local carriers; can include DMEPOS services)"
      }
    ]
  },
  "patient":{
    "reference":"Patient/20000000000001"
  },
  "identifier":[
    {
      "system":"https://bluebutton.cms.gov/resources/variables/clm_id",
      "value":"10300336722"
    },
    {
      "system":"https://bluebutton.cms.gov/resources/identifier/claim-group",
      "value":"44077735787"
    }
  ],
  "insurance":{
    "coverage":{
      "reference":"Coverage/part-b-20000000000001"
    }
  },
  "item":[
    {
      "locationCodeableConcept":{
        "extension":[
          {
            "valueCoding":{
              "system":"https://bluebutton.cms.gov/resources/variables/prvdr_state_cd",
              "code":"99",
              "display":"With 000 county code is American Samoa; otherwise unknown"
            },
            "url":"https://bluebutton.cms.gov/resources/variables/prvdr_state_cd"
          },
          {
            "valueCoding":{
              "system":"https://bluebutton.cms.gov/resources/variables/prvdr_zip",
              "code":"999999999"
            },
            "url":"https://bluebutton.cms.gov/resources/variables/prvdr_zip"
          },
          {
            "valueCoding":{
              "system":"https://bluebutton.cms.gov/resources/variables/carr_line_prcng_lclty_cd",
              "code":"99"
            },
            "url":"https://bluebutton.cms.gov/resources/variables/carr_line_prcng_lclty_cd"
          }
        ],
        "coding":[
          {
            "system":"https://bluebutton.cms.gov/resources/variables/line_place_of_srvc_cd",
            "code":"99",
            "display":"Other Place of Service. Other place of service not identified above."
          }
        ]
      },
      "service":{
        "coding":[
          {
            "system":"https://bluebutton.cms.gov/resources/codesystem/hcpcs",
            "code":"45384",
            "version":"0"
          }
        ]
      },
      "extension":[
        {
          "valueCoding":{
            "system":"https://bluebutton.cms.gov/resources/variables/carr_line_mtus_cd",
            "code":"3",
            "display":"Services"
          },
          "url":"https://bluebutton.cms.gov/resources/variables/carr_line_mtus_cd"
        },
        {
          "valueQuantity":{
            "value":1
          },
          "url":"https://bluebutton.cms.gov/resources/variables/carr_line_mtus_cnt"
        }
      ],
      "servicedPeriod":{
        "start":"2000-10-01",
        "end":"2000-10-01"
      },
      "quantity":{
        "value":1
      },
      "category":{
        "coding":[
          {
            "system":"https://bluebutton.cms.gov/resources/variables/line_cms_type_srvc_cd",
            "code":"2",
            "display":"Surgery"
          }
        ]
      },
      "sequence":1,
      "diagnosisLinkId":[
        2
      ],
      "adjudication":[
        {
          "category":{
            "coding":[
              {
                "system":"https://bluebutton.cms.gov/resources/codesystem/adjudication",
                "code":"https://bluebutton.cms.gov/resources/variables/carr_line_rdcd_pmt_phys_astn_c",
                "display":"Carrier Line Reduced Payment Physician Assistant Code"
              }
            ]
          },
          "reason":{
            "coding":[
              {
                "system":"https://bluebutton.cms.gov/resources/variables/carr_line_rdcd_pmt_phys_astn_c",
                "code":"0",
                "display":"N/A"
              }
            ]
          }
        },
        {
          "extension":[
            {
              "valueCoding":{
                "system":"https://bluebutton.cms.gov/resources/variables/line_pmt_80_100_cd",
                "code":"0",
                "display":"80%"
              },
              "url":"https://bluebutton.cms.gov/resources/variables/line_pmt_80_100_cd"
            }
          ],
          "amount":{
            "system":"urn:iso:std:iso:4217",
            "code":"USD",
            "value":250.0
          },
          "category":{
            "coding":[
              {
                "system":"https://bluebutton.cms.gov/resources/codesystem/adjudication",
                "code":"https://bluebutton.cms.gov/resources/variables/line_nch_pmt_amt",
                "display":"Line NCH Medicare Payment Amount"
              }
            ]
          }
        },
        {
          "category":{
            "coding":[
              {
                "system":"https://bluebutton.cms.gov/resources/codesystem/adjudication",
                "code":"https://bluebutton.cms.gov/resources/variables/line_bene_pmt_amt",
                "display":"Line Payment Amount to Beneficiary"
              }
            ]
          },
          "amount":{
            "system":"urn:iso:std:iso:4217",
            "code":"USD",
            "value":0.0
          }
        }
      ]
    }
  ]
}
```

## Beneficiary Patient Data
The process of retrieving patient data is very similar to exporting explanation of benefit data.

#### 1. Obtain an access token
See [Authentication and Authorization](#authentication-and-authorization) above.

#### 2. Initiate an export job

##### Request
`GET /api/v1/Patient/$export`

To start a patient data export job, a GET request is made to the Patient export endpoint. An access token as well as `Accept` and `Prefer` headers are required.

The dollar sign ('$') before the word "export" in the URL indicates that the endpoint is an action rather than a resource. The format is defined by the [FHIR Bulk Data Export spec](https://github.com/smart-on-fhir/fhir-bulk-data-docs/blob/master/export.md){:target="_blank"}.

###### Headers
* `Authorization: Bearer {token}`
* `Accept: application/fhir+json`
* `Prefer: respond-async`

###### cURL command
```sh
curl -v https://sandbox.bcda.cms.gov/api/v1/Patient/\$export \
-H 'Authorization: Bearer {token}' \
-H 'Accept: application/fhir+json' \
-H 'Prefer: respond-async'
```

##### Response
If the request was successful, a `202 Accepted` response code will be returned and the response will include a `Content-Location` header. The value of this header indicates the location to check for job status and outcome. In the example header below, the number 43 in the URL represents the ID of the export job.

###### Headers
* `Content-Location: https://sandbox.bcda.cms.gov/api/v1/jobs/43`

#### 3. Check the status of the export job

##### Request
`GET https://sandbox.bcda.cms.gov/api/v1/jobs/43`

Using the `Content-Location` header value from the Patient data export response, you can check the status of the export job. The status will change from `202 Accepted` to `200 OK` when the export job is complete and the data is ready to be downloaded.

###### Headers
* `Authorization: Bearer {token}`

###### cURL Command
```sh
curl -v https://sandbox.bcda.cms.gov/api/v1/jobs/43 \
-H 'Authorization: Bearer {token}'
```

##### Responses
* `202 Accepted` indicates that the job is processing. Headers will include `X-Progress: In Progress`
* `200 OK` indicates that the job is complete. Below is an example of the format of the response body.
```json
{
  "transactionTime": "2018-10-19T14:47:33.975024Z",
  "request": "https://sandbox.bcda.cms.gov/api/v1/Patient/$export",
  "requiresAccessToken": true,
  "output": [
    {
      "type": "Patient",
      "url": "https://sandbox.bcda.cms.gov/data/43/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson"
    }
  ],
  "error": [
    {
      "type": "OperationOutcome",
      "url": "https://sandbox.bcda.cms.gov/data/43/DBBD1CE1-AE24-435C-807D-ED45953077D3-error.ndjson"
    }
  ]
}
```
Patient demographic data can be found at the URLs within the `output` field. The number 43 in the data file URLs is the same job ID from the `Content-Location` header URL in previous step. If some of the data cannot be exported due to errors, details of the errors can be found at the URLs in the `error` field. The errors are provided in an NDJSON file as FHIR [OperationOutcome](https://www.hl7.org/fhir/operationoutcome.html){:target="_blank"} resources.

#### 4. Retrieve the NDJSON output file(s)
To obtain the exported explanation of benefit data, a GET request is made to the output URLs in the job status response when the job reaches the Completed state. The data will be presented as an NDJSON file of [Patient](https://www.hl7.org/fhir/patient.html){:target="_blank"} resources.

##### Request
`GET https://sandbox.bcda.cms.gov/data/43/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson`

###### Headers
* `Authorization: Bearer {token}`

###### cURL command
```sh
curl https://sandbox.bcda.cms.gov/data/43/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson \
-H 'Authorization: Bearer {token}'
```

##### Response
The response will be the requested data as FHIR Patient resources in NDJSON format, encrypted. An unencrypted example of one such resource is shown below.

```json
{
	"fullUrl": "https:///v1/fhir/Patient/19990000002901",
	"resource": {
		"address": [{
			"district": "999",
			"postalCode": "99999",
			"state": "34"
		}],
		"birthDate": "1999-06-01",
		"extension": [{
			"url": "https://bluebutton.cms.gov/resources/variables/race",
			"valueCoding": {
				"code": "1",
				"display": "White",
				"system": "https://bluebutton.cms.gov/resources/variables/race"
			}
		}],
		"gender": "unknown",
		"id": "19990000002901",
		"identifier": [{
			"system": "https://bluebutton.cms.gov/resources/variables/bene_id",
			"value": "19990000002901"
		}, {
			"system": "https://bluebutton.cms.gov/resources/identifier/hicn-hash",
			"value": "77174c6546668151f741cca47739271baf364d19825a387101d39fc303d91f2c"
		}],
		"name": [{
			"family": "Doe",
			"given": ["Jane", "X"],
			"use": "usual"
		}],
		"resourceType": "Patient"
	}
}
```
## Beneficiary Coverage Data
The process of retrieving coverage data is very similar to all of the other exporting operations supported by this API.

#### 1. Obtain an access token
See [Authentication and Authorization](#authentication-and-authorization) above.

#### 2. Initiate an export job

##### Request
`GET /api/v1/Coverage/$export`

To start a coverage data export job, a GET request is made to the Coverage export endpoint. An access token as well as `Accept` and `Prefer` headers are required.

The dollar sign ('$') before the word "export" in the URL indicates that the endpoint is an action rather than a resource. The format is defined by the [FHIR Bulk Data Export spec](https://github.com/smart-on-fhir/fhir-bulk-data-docs/blob/master/export.md){:target="_blank"}.

###### Headers
* `Authorization: Bearer {token}`
* `Accept: application/fhir+json`
* `Prefer: respond-async`

###### cURL command
```sh
curl -v https://sandbox.bcda.cms.gov/api/v1/Coverage/\$export \
-H 'Authorization: Bearer {token}' \
-H 'Accept: application/fhir+json' \
-H 'Prefer: respond-async'
```

##### Response
If the request was successful, a `202 Accepted` response code will be returned and the response will include a `Content-Location` header. The value of this header indicates the location to check for job status and outcome. In the example header below, the number 44 in the URL represents the ID of the export job.

###### Headers
* `Content-Location: https://sandbox.bcda.cms.gov/api/v1/jobs/44`

#### 3. Check the status of the export job

##### Request
`GET https://sandbox.bcda.cms.gov/api/v1/jobs/44`

Using the `Content-Location` header value from the Coverage data export response, you can check the status of the export job. The status will change from `202 Accepted` to `200 OK` when the export job is complete and the data is ready to be downloaded.

###### Headers
* `Authorization: Bearer {token}`

###### cURL Command
```sh
curl -v https://sandbox.bcda.cms.gov/api/v1/jobs/44 \
-H 'Authorization: Bearer {token}'
```

##### Responses
* `202 Accepted` indicates that the job is processing. Headers will include `X-Progress: In Progress`
* `200 OK` indicates that the job is complete. Below is an example of the format of the response body.
```json
{
  "transactionTime": "2018-10-19T14:47:33.975024Z",
  "request": "https://sandbox.bcda.cms.gov/api/v1/Coverage/$export",
  "requiresAccessToken": true,
  "output": [
    {
      "type": "Coverage",
      "url": "https://sandbox.bcda.cms.gov/data/44/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson"
    }
  ],
  "error": [
    {
      "type": "OperationOutcome",
      "url": "https://sandbox.bcda.cms.gov/data/44/DBBD1CE1-AE24-435C-807D-ED45953077D3-error.ndjson"
    }
  ]
}
```
Coverage demographic data can be found at the URLs within the `output` field. The number 44 in the data file URLs is the same job ID from the `Content-Location` header URL in previous step. If some of the data cannot be exported due to errors, details of the errors can be found at the URLs in the `error` field. The errors are provided in an NDJSON file as FHIR [OperationOutcome](https://www.hl7.org/fhir/operationoutcome.html) resources.

#### 4. Retrieve the NDJSON output file(s)
To obtain the exported coverage data, a GET request is made to the output URLs in the job status response when the job reaches the Completed state. The data will be presented as an NDJSON file of [Coverage](https://www.hl7.org/fhir/coverage.html) resources.

##### Request
`GET https://sandbox.bcda.cms.gov/data/44/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson`

###### Headers
* `Authorization: Bearer {token}`

###### cURL command
```sh
curl https://sandbox.bcda.cms.gov/data/44/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson \
-H 'Authorization: Bearer {token}'
```

##### Response
The response will be the requested data as FHIR Coverage resources in NDJSON format. An example of one such resource is shown below.

  ```json
  {
  "fullUrl": "https:///v1/fhir/Coverage/part-a-19990000002901",
  "resource": {
    "beneficiary": {
      "reference": "Patient/19990000002901"
    },
    "extension": [
      {
        "url": "https://bluebutton.cms.gov/resources/variables/ms_cd",
        "valueCoding": {
          "code": "10",
          "display": "Aged without end-stage renal disease (ESRD)",
          "system": "https://bluebutton.cms.gov/resources/variables/ms_cd"
        }
      },
      {
        "url": "https://bluebutton.cms.gov/resources/variables/orec",
        "valueCoding": {
          "code": "0",
          "display": "Old age and survivor’s insurance (OASI)",
          "system": "https://bluebutton.cms.gov/resources/variables/orec"
        }
      },
      {
        "url": "https://bluebutton.cms.gov/resources/variables/crec",
        "valueCoding": {
          "code": "0",
          "display": "Old age and survivor’s insurance (OASI)",
          "system": "https://bluebutton.cms.gov/resources/variables/crec"
        }
      },
      {
        "url": "https://bluebutton.cms.gov/resources/variables/esrd_ind",
        "valueCoding": {
          "code": "0",
          "display": "the beneficiary does not have ESRD",
          "system": "https://bluebutton.cms.gov/resources/variables/esrd_ind"
        }
      },
      {
        "url": "https://bluebutton.cms.gov/resources/variables/a_trm_cd",
        "valueCoding": {
          "code": "0",
          "display": "Not Terminated",
          "system": "https://bluebutton.cms.gov/resources/variables/a_trm_cd"
        }
      }
    ],
    "grouping": {
      "subGroup": "Medicare",
      "subPlan": "Part A"
    },
    "id": "part-a-19990000002901",
    "resourceType": "Coverage",
    "status": "active",
    "type": {
      "coding": [
        {
          "code": "Part A",
          "system": "Medicare"
        }
      ]
    }
  }
}
```

