---
layout: home
title:  "Beneficiary Claims Data API"
date:   2017-10-30 09:21:12 -0500
description: Shared Savings Program ACOs can use this FHIR-based API to retrieve bulk Medicare claims data related to their assignable or prospectively assigned beneficiaries. Under construction; feedback invited.
landing-page: live
gradient: "blueberry-lime-background"
subnav-link-gradient: "blueberry-lime-link"
sections:
  - Getting Started
  - What We've Learned
  - About the Data
ctas:

  - title: Visit the BCDA User Guide
    link: ./user_guide.html
  - title: Join the BCDA Google Group
    link: https://groups.google.com/forum/#!forum/bc-api
    target: _blank

---


# Overview

  The Beneficiary Claims Data API (BCDA) will enable Accountable Care Organizations (ACOs) participating in the Shared Savings Program to retrieve Medicare Part A, Part B, and Part D claims data for their assigned or assignable beneficiaries. This includes Medicare claims data for instances in which beneficiaries receive care outside of the ACO, allowing a full picture of patient care. When it is in production, the API will provide similar data to Claim and Claim Line Feed (CCLF) files, currently provided monthly to Shared Savings Program ACOs by CMS.

   Developers, analysts, and administrators at Shared Savings Program ACOs are invited to try out the BCDA sandbox experience. Learn more and obtain credentials by visiting the BCDA [user guide](./user_guide.html).
   
   * We’re currently providing synthetic data that resembles the kinds of data Shared Savings Program ACOs will receive by connecting with this endpoint, so that they can try out different ways of interacting with the API before receiving live data.

   * We’re providing this test experience and documentation early so we can learn from Shared Savings Program ACOs and their vendor partners who need to access this information, about what works best for them. Through conversations and test drives, we strive to learn what ACOs need from bulk beneficiary claims data, and create a process that meets their needs.

   * While the initial iteration of the sandbox focuses on Shared Savings Program ACOs, all ACOs and their vendor partners are invited to explore the sandbox and give their feedback.

## Getting Started

   * [User Guide](./user_guide.html)
   * [Encryption Overview](./encryption.html)
   * [Swagger Documentation](./api/v1/swagger)

## What has been learned so far from ACOs in the pilot?
{:#what-we-ve-learned}

   Developers, analysts, and administrators at ACOs have been instrumental in shaping CMS’ approach to this API. With their feedback, the team is implementing the following elements:

   * Providing clear, human-readable narrative documentation to aid all users’ use of the API and the data that is shared
   * Using resilient NDJSON ([New Line Delimited JSON](http://ndjson.org){:target="_blank"}) rather than fixed-width files in response to requests for delimited information
   * Providing bulk beneficiary claims data through an automated retrieval process that requires minimal hands-on intervention to receive
   * Formatting data in accordance with robust Fast Health Interoperability Resource ([FHIR](https://www.hl7.org/fhir/overview.html){:target="_blank"}) specifications

   BCDA will continue to take an iterative approach to testing and learning from its users.


## About the Data

   If you're used to working with CCLF files, you'll want to know more about the data we're using and how to work with it.
   For data formatting, BCDA follows the workflow outlined by the [FHIR Bulk Data Export Implementation Guide](https://github.com/smart-on-fhir/fhir-bulk-data-docs/blob/master/export.md){:target="_blank"}, using the [HL7 FHIR Standard](https://www.hl7.org/fhir/){:target="_blank"}.
   Claims data is provided as FHIR resources in [NDJSON](http://ndjson.org/){:target="_blank"} format.

   What is FHIR ([Fast Healthcare Interoperability Resources](https://www.hl7.org/fhir/){:target="_blank"})?   

   * FHIR is a specification for how servers that provide healthcare records should be set up.

   * FHIR provides a framework for the exchange of healthcare-related data, allowing any system to access and consume this data to solve clinical and administrative problems around healthcare-related data.
   * BCDA will be using the following endpoints from the FHIR spec:
        * patient endpoint
        * explanation of benefits endpoint
        * coverage endpoint

   How is BCDA different from Blue Button 2.0? 

   * Blue Button 2.0 provides FHIR-formatted data for one individual Medicare beneficiary at a time, to registered applications with beneficiary authorization. See [https://bluebutton.cms.gov/](https://bluebutton.cms.gov/){:target="_blank"}.
   * BCDA provides FHIR-formatted bulk data files to an ACO for all of the beneficiaries eligible to a given Shared Savings Program ACO. BCDA does not require individual beneficiary authorization. 
