"""
Dictionaries that map FHIR 3 to CCLF
"""

cclf1 = {
    'CUR_CLM_UNIQ_ID'                 : 'Not mapped',
    'PRVDR_OSCAR_NUM'                 : 'Eob.provider.identifier.value',
    'BENE_MBI_ID'                     : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'                    : 'Not mapped',
    'CLM_TYPE_CD'                     : 'Eob.type.coding[N].code',
    'CLM_FROM_DT'                     : 'Eob.billablePeriod.start',
    'CLM_THRU_DT'                     : 'Eob.billablePeriod.end',
    'CLM_BILL_FAC_TYPE_CD'            : 'Eob.facility.extension[N].valueCoding.code',
    'CLM_BILL_CLSFCTN_CD'             : 'Eob.type.coding[N].code',
    'PRNCPL_DGNS_CD'                  : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding[X].code',
    'ADMTG_DGNS_CD'                   : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding[X].code',
    'CLM_MDCR_NPMT_RSN_CD'            : 'Eob.extension[N].valueCoding.code',
    'CLM_PMT_AMT'                     : 'Eob.payment.amount.value',
    'CLM_NCH_PRMRY_PYR_CD'            : 'Eob.information[N].code.coding.code',
    'PRVDR_FAC_FIPS_ST_CD'            : 'Eob.item[N].locationAddress.state',
    'BENE_PTNT_STUS_CD'               : 'Eob.information[N].code.coding.code',
    'DGNS_DRG_CD'                     : 'Eob.diagnosis[N].packageCode.coding.value',
    'CLM_OP_SRVC_TYPE_CD'             : 'Eob.type.coding[N].code',
    'FAC_PRVDR_NPI_NUM'               : 'Eob.facility.identifier.value',
    'OPRTG_PRVDR_NPI_NUM'             : 'Eob.careTeam[N].provider.identifier.value',
    'ATNDG_PRVDR_NPI_NUM'             : 'Eob.careTeam[N].provider.identifier.value',
    'OTHR_PRVDR_NPI_NUM'              : 'Eob.careTeam[N].provider.identifier.value',
    'CLM_ADJSMT_TYPE_CD'              : 'Eob.status',
    'CLM_EFCTV_DT'                    : 'Eob.information[N].timingDate',
    'CLM_IDR_LD_DT'                   : 'Not mapped',
    'BENE_EQTBL_BIC_HICN_NUM'         : 'Not mapped',
    'CLM_ADMSN_TYPE_CD'               : 'Eob.information[N].code.coding.code',
    'CLM_ADMSN_SRC_CD'                : 'Eob.information[N].code.coding.code',
    'CLM_BILL_FREQ_CD'                : 'Eob.information[N].code.coding.code',
    'CLM_QUERY_CD'                    : 'Eob.billablePeriod.extension[1].valueCoding.code',
    'DGNS_PRCDR_ICD_IND'              : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding[X].system',
    'CLM_MDCR_INSTNL_TOT_CHRG_AMT'    : 'Eob.totalCost.value',
    'CLM_MDCR_IP_PPS_CPTL_IME_AMT'    : 'Eob.extension[N].valueMoney.value',
    'CLM_OPRTNL_IME_AMT'              : 'Eob.extension[N].valueMoney.value',
    'CLM_MDCR_IP_PPS_DSPRPRTNT_AMT'   : 'Eob.extension[N].valueMoney.value',
    'CLM_HIPPS_UNCOMPD_CARE_AMT'      : 'Eob.extension[N].valueMoney.value',
    'CLM_OPRTNL_DSPRTNT_AMT'          : 'Eob.extension[N].valueMoney.value',
}

cclf2 = {
    'CUR_CLM_UNIQ_ID'            : 'Not mapped',
    'CLM_LINE_NUM'               : 'Eob.item.sequence',
    'BENE_MBI_ID'                : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'               : 'Not mapped',
    'CLM_TYPE_CD'                : 'Eob.type.coding[N].code',
    'CLM_LINE_FROM_DT'           : 'Eob.item[N].servicedPeriod.start',
    'CLM_LINE_THRU_DT'           : 'Eob.item[N].servicedPeriod.end',
    'CLM_LINE_PROD_REV_CTR_CD'   : 'Eob.item[N].revenue.coding.code',
    'CLM_LINE_INSTNL_REV_CTR_DT' : 'Eob.item.servicedDate',
    'CLM_LINE_HCPCS_CD'          : 'Eob.item[N].service.coding.code',
    'BENE_EQTBL_BIC_HICN_NUM'    : 'Not mapped',
    'PRVDR_OSCAR_NUM'            : 'Eob.provider.identifier.value',
    'CLM_FROM_DT'                : 'Eob.billablePeriod.start',
    'CLM_THRU_DT'                : 'Eob.billablePeriod.end',
    'CLM_LINE_SRVC_UNIT_QTY'     : 'Eob.item[N].quantity.value',
    'CLM_LINE_CVRD_PD_AMT'       : 'Eob.item[N].adjudication.amount.value',
    'HCPCS_1_MDFR_CD'            : 'Eob.item[N].modifier.coding.code',
    'HCPCS_2_MDFR_CD'            : 'See HCPCS_1_MDFR_CD above',
    'HCPCS_3_MDFR_CD'            : 'See HCPCS_1_MDFR_CD above',
    'HCPCS_4_MDFR_CD'            : 'See HCPCS_1_MDFR_CD above',
    'HCPCS_5_MDFR_CD'            : 'See HCPCS_1_MDFR_CD above',
    'CLM_REV_APC_HIPPS_CD'       : 'Eob.item[N].revenue.coding.code',
}

cclf3 = {
    'CUR_CLM_UNIQ_ID'          : 'Not mapped',
    'BENE_MBI_ID'              : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'             : 'Not mapped',
    'CLM_TYPE_CD'              : 'Eob.type.coding[N].code',
    'CLM_VAL_SQNC_NUM'         : 'Eob.procedure[N].sequence',
    'CLM_PRCDR_CD'             : 'Eob.procedure[N].procedureCodeableConcept.coding.code',
    'CLM_PRCDR_PRFRM_DT'       : 'Eob.procedure[N].procedure.date',
    'BENE_EQTBL_BIC_HICN_NUM'  : 'Not mapped',
    'PRVDR_OSCAR_NUM'          : 'Eob.provider.identifier.value',
    'CLM_FROM_DT'              : 'Eob.billablePeriod.start',
    'CLM_THRU_DT'              : 'Eob.billablePeriod.end',
    'DGNS_PRCDR_ICD_IND'       : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding[X].system',
}

cclf4 = {
    'CUR_CLM_UNIQ_ID'           : 'Not mapped',
    'BENE_MBI_ID'               : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'              : 'Not mapped',
    'CLM_TYPE_CD'               : 'Eob.type.coding[N].code',
    'CLM_PROD_TYPE_CD'          : 'Eob.diagnosis[N].type.coding.code',
    'CLM_VAL_SQNC_NUM'          : 'Eob.diagnosis[N].sequence',
    'CLM_DGNS_CD'               : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding[X].code',
    'BENE_EQTBL_BIC_HICN_NUM'   : 'Not mapped',
    'PRVDR_OSCAR_NUM'           : 'Eob.careTeam[N].provider.identifier.value',
    'CLM_FROM_DT'               : 'Eob.billablePeriod.start',
    'CLM_THRU_DT'               : 'Eob.billablePeriod.end',
    'CLM_POA_IND'               : 'Eob.diagnosis[N].extension.valueCoding.code',
    'DGNS_PRCDR_ICD_IND'        : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding[X].system',
}

cclf5 = {
    'CUR_CLM_UNIQ_ID'           : 'Not mapped',
    'CLM_LINE_NUM'              : 'Eob.item.sequence',
    'BENE_MBI_ID'               : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'              : 'Not mapped',
    'CLM_TYPE_CD'               : 'Eob.type.coding[N].code',
    'CLM_FROM_DT'               : 'Eob.billablePeriod.start',
    'CLM_THRU_DT'               : 'Eob.billablePeriod.end',
    'RNDRG_PRVDR_TYPE_CD'       : 'Eob.careTeam.extension[N].valueCoding.code',
    'RNDRG_PRVDR_FIPS_ST_CD'    : 'Eob.item[N].locationCodeableConcept.extension[N].valueCoding.code',
    'CLM_PRVDR_SPCLTY_CD'       : 'Eob.careTeam[N].qualification.coding.code',
    'CLM_FED_TYPE_SRVC_CD'      : 'Eob.item[N].category.coding.code',
    'CLM_POS_CD'                : 'Eob.item[N].locationCodeableConcept.coding.code',
    'CLM_LINE_FROM_DT'          : 'Eob.item[N].servicedPeriod.start',
    'CLM_LINE_THRU_DT'          : 'Eob.item[N].servicedPeriod.end',
    'CLM_LINE_HCPCS_CD'         : 'Eob.item[N].service.coding.code',
    'CLM_LINE_CVRD_PD_AMT'      : 'Eob.item[N].adjudication[N].amount',
    'CLM_LINE_PRMRY_PYR_CD'     : 'Eob.item[N].extension[N].valueCoding.code',
    'CLM_LINE_DGNS_CD'          : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_RNDRG_PRVDR_TAX_NUM'   : 'Eob.careTeam.provider.identifier.value',
    'RNDRG_PRVDR_NPI_NUM'       : 'Eob.careTeam[N].provider.identifier.value',
    'CLM_CARR_PMT_DNL_CD'       : 'Eob.extension[N].valueCoding.code',
    'CLM_PRCSG_IND_CD'          : 'Eob.item[N].adjudication[N].reason.coding.code',
    'CLM_ADJSMT_TYPE_CD'        : 'Eob.status',
    'CLM_EFCTV_DT'              : 'Eob.information[N].timingDate',
    'CLM_IDR_LD_DT'             : 'Not mapped',
    'CLM_CNTL_NUM'              : 'Eob.identifier[N].value',
    'BENE_EQTBL_BIC_HICN_NUM'   : 'Not mapped',
    'CLM_LINE_ALOWD_CHRG_AMT'   : 'Eob.item[N].adjudication[N].amount.value',
    'CLM_LINE_SRVC_UNIT_QTY'    : 'Eob.item[N].quantity.value',
    'HCPCS_1_MDFR_CD'           : 'Eob.item[N].modifier[N].coding.code',
    'HCPCS_2_MDFR_CD'           : 'Eob.item[N].modifier[N].coding.code',
    'HCPCS_3_MDFR_CD'           : 'Eob.item[N].modifier[N].coding.code',
    'HCPCS_4_MDFR_CD'           : 'Eob.item[N].modifier[N].coding.code',
    'HCPCS_5_MDFR_CD'           : 'Eob.item[N].modifier[N].coding.code',
    'CLM_DISP_CD'               : 'Eob.disposition',
    'CLM_DGNS_1_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_2_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_3_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_4_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_5_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_6_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_7_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_8_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'DGNS_PRCDR_ICD_IND'        : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding[X].system',
    'CLM_DGNS_9_CD'             : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_10_CD'            : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_11_CD'            : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'CLM_DGNS_12_CD'            : 'Eob.diagnosis[N].diagnosisCodeableConcept.coding.code',
    'HCPCS_BETOS_CD'            : 'Eob.item[N].extension[N].valueCoding.code',
}

cclf6 = {
    'CUR_CLM_UNIQ_ID'           : 'Not mapped',
    'CLM_LINE_NUM'              : 'Eob.item.sequence',
    'BENE_MBI_ID'               : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'              : 'Not mapped',
    'CLM_TYPE_CD'               : 'Eob.type.coding[N].code',
    'CLM_FROM_DT'               : 'Eob.billablePeriod.start',
    'CLM_THRU_DT'               : 'Eob.billablePeriod.end',
    'CLM_FED_TYPE_SRVC_CD'      : 'Eob.item[N].category.coding.code',
    'CLM_POS_CD'                : 'Eob.item[N].locationCodeableConcept.coding.code',
    'CLM_LINE_FROM_DT'          : 'Eob.item[N].servicedPeriod.start',
    'CLM_LINE_THRU_DT'          : 'Eob.item.servicedPeriod.end',
    'CLM_LINE_HCPCS_CD'         : 'Eob.item[N].service.coding.code',
    'CLM_LINE_CVRD_PD_AMT'      : 'Eob.item.adjudication.amount.value',
    'CLM_PRMRY_PYR_CD'          : 'Eob.item.extension[N].valueCoding.code',
    'PAYTO_PRVDR_NPI_NUM'       : 'Eob.careTeam[N].provider.identifier.value',
    'ORDRG_PRVDR_NPI_NUM'       : 'Eob.careTeam[N].provider.identifier.value',
    'CLM_CARR_PMT_DNL_CD'       : 'Eob.extension[N].valueCoding.code',
    'CLM_PRCSG_IND_CD'          : 'Eob.item[N].adjudication[N].reason.coding.code',
    'CLM_ADJSMT_TYPE_CD'        : 'Not mapped',
    'CLM_EFCTV_DT'              : 'Eob.information[N].timingDate',
    'CLM_IDR_LD_DT'             : 'Not mapped',
    'CLM_CNTL_NUM'              : 'Eob.identifier[N].value',
    'BENE_EQTBL_BIC_HICN_NUM'   : 'Not mapped',
    'CLM_LINE_ALOWD_CHRG_AMT'   : 'Eob.item[N].adjudication[N].amount.value',
    'CLM_DISP_CD'               : 'Eob.disposition',
}

cclf7 = {
    'CUR_CLM_UNIQ_ID'               : 'Not mapped',
    'BENE_MBI_ID'                   : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'                  : 'Not mapped',
    'CLM_LINE_NDC_CD'               : 'Eob.item.service.coding.code',
    'CLM_TYPE_CD'                   : 'Eob.type.coding[N].code',
    'CLM_LINE_FROM_DT'              : 'Eob.item[N].servicedPeriod.start',
    'PRVDR_SRVC_ID_QLFYR_CD'        : 'Not mapped - see Additional info',
    'CLM_SRVC_PRVDR_GNRC_ID_NUM'    : 'Explanation of Benefit.careTeam[N].provider',
    'CLM_DSPNSNG_STUS_CD'           : 'Not mapped - see Additional Info',
    'CLM_DAW_PROD_SLCTN_CD'         : 'Eob.information[N].code.coding.code',
    'CLM_LINE_SRVC_UNIT_QTY'        : 'Eob.item[N].quantity.value',
    'CLM_LINE_DAYS_SUPLY_QTY'       : 'Eob.item[N].quantity.extension[N].valueQuantity.value',
    'PRVDR_PRSBNG_ID_QLFYR_CD'      : 'Not mapped',
    'CLM_PRSBNG_PRVDR_GNRC_ID_NUM'  : 'Eob.facility.identifer.value',
    'CLM_LINE_BENE_PMT_AMT'         : 'Eob.item[N].adjudication[N].amount.value',
    'CLM_ADJSMT_TYPE_CD'            : 'Eob.status',
    'CLM_EFCTV_DT'                  : 'Eob.information[N].timingDate',
    'CLM_IDR_LD_DT'                 : 'Not mapped',
    'CLM_LINE_RX_SRVC_RFRNC_NUM'    : 'Eob.identifier[N].value',
    'CLM_LINE_RX_FILL_NUM'          : 'Eob.item[N].extension[N].value',
    'CLM_PHRMCY_SRVC_TYPE_CD'       : 'Eob.facility.extension.valueCoding.code',
}

def get_current(obj):
    for item in obj:
        if item['ValueCoding'] == 'current':
            return item

cclf8 = {
    "BENE_MBI_ID"               : ('patient', lambda obj: get_current(obj)['value']),  # Patient.identifier[N].value
    "BENE_HIC_NUM"              : "Not mapped",
    "BENE_FIPS_STATE_CD"        : ('patient', lambda obj: obj['address'][0]['state']),  # Patient.address.state
    # "BENE_FIPS_CNTY_CD"         : "No longer Mapped, use Beneficiary mailing addresses to get county information",
    "BENE_FIPS_CNTY_CD"         : "Not mapped",
    "BENE_ZIP_CD"               : ('patient', lambda obj: obj['address'][0]['postalCode']),  # Patient.address.postalCode
    "BENE_DOB"                  : ('patient', lambda obj: obj['birthDate']),  # "patient.birthDate"
    "BENE_SEX_CD"               : ('patient', lambda obj: obj['gender']),  # "Patient.gender"
    "BENE_RACE_CD"              : "Patient.extension[N].valueCoding.code",
    "BENE_AGE"                  : "Not mapped",
    "BENE_MDCR_STUS_CD"         : "Coverage.extension[N].valueCoding.code",
    "BENE_DUAL_STUS_CD"         : "Coverage.extension[N].valueCoding.code",
    "BENE_DEATH_DT"             : ('patient', lambda obj: obj['deceased']),  # Patient.deceased[x]
    "BENE_RNG_BGN_DT"           : ('eob', lambda obj: obj['hospitalization'][0]['start']),  # Eob.hospitalization.start
    "BENE_RNG_END_DT"           : ('eob', lambda obj: obj['hospitalization'][0]['end']),  # Eob.hospitalization.end
    "BENE_1ST_NAME"             : ('patient', lambda obj: obj['name'][0]['given'][0]),  # Patient.name.given[N]
    "BENE_MIDL_NAME"            : ('patient', lambda obj: obj['name'][0]['given'][1]),  # Patient.name.given[N]
    "BENE_LAST_NAME"            : ('patient', lambda obj: obj['name'][0]['family']),  # Patient.name.family
    "BENE_ORGNL_ENTLMT_RSN_CD"  : "Coverage.extension[N].valueCoding.code",
    "BENE_ENTLMT_BUYIN_IND"     : "Coverage.extension[N].valueCoding.code",
    "BENE_PART_A_ENRLMT_BGN_DT" : "Coverage[N].period.start",
    "BENE_PART_B_ENRLMT_BGN_DT" : "Coverage[N].period.start",
    "BENE_LINE_1_ADR"           : ('patient', lambda obj: obj['address'][0]['line'][0]),  # Patient.address.line[N]
    "BENE_LINE_2_ADR"           : ('patient', lambda obj: obj['address'][0]['line'][1]),  # Patient.address.line[N]
    "BENE_LINE_3_ADR"           : ('patient', lambda obj: obj['address'][0]['line'][2]),  # Patient.address.line[N]
    "BENE_LINE_4_ADR"           : ('patient', lambda obj: obj['address'][0]['line'][3]),  # Patient.address.line[N]
    "BENE_LINE_5_ADR"           : ('patient', lambda obj: obj['address'][0]['line'][4]),  # Patient.address.line[N]
    "BENE_LINE_6_ADR"           : ('patient', lambda obj: obj['address'][0]['line'][5]),  # Patient.address.line[N]
    "GEO_ZIP_PLC_NAME"          : ('patient', lambda obj: obj['address'][0]['city']),  # Patient.address.city
    "GEO_USPS_STATE_CD"         : ('patient', lambda obj: obj['address'][0]['state']),  # Patient.address.state
    "GEO_ZIP5_CD"               : ('patient', lambda obj: obj['address'][0]['postalCode']),  # Patient.address.postalCode
    "GEO_ZIP4_CD"               : ('patient', lambda obj: obj['address'][0]['postalCode']),  # Patient.address.postalCode[N]=Patient.address.postalCode[1] - GEO_ZIP4_CD
}

cclf9 = {
    'HICN_MBI_XREF_IND' : 'Eob.patient.identifier.type',
    'CRNT_NUM'          : 'Patient.identifier[N].value',
    'PRVS_NUM'          : 'Patient.identifier[N].value',
    'PRVS_ID_EFCTV_DT'  : 'Not mapped',
    'PRVS_ID_OBSLT_DT'  : 'Not mapped',
    'BENE_RRB_NUM'      : 'Not mapped',
}

cclfA = {
    'CUR_CLM_UNIQ_ID'       : 'Not mapped',
    'BENE_MBI_ID'           : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'          : 'Not mapped',
    'CLM_TYPE_CD'           : 'Eob.type.coding[N].code',
    'CLM_ACTV_CARE_FROM_DT' : 'Eob.hospitalization.start',
}

cclfB = {
    'CUR_CLM_UNIQ_ID'   : 'Not mapped',
    'CLM_LINE_NUM'      : 'Eob.item.sequence',
    'BENE_MBI_ID'       : 'Patient.identifier[N].value',
    'BENE_HIC_NUM'      : 'Not mapped',
    'CLM_TYPE_CD'       : 'Eob.type.coding[N].code',
}
