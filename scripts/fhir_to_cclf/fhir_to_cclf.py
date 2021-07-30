# /usr/bin/python3

import json

from tqdm import tqdm
import pandas as pd


def process_ndjson(string_data: str):
    if string_data[-1:] == '\n':  # Handle possible trailing new line
        string_data = string_data[0:-1]
    string_data = '[' + string_data + ']'
    string_data = string_data.replace('\n', ',')
    return json.loads(string_data)


if __name__=='__main__':
    # Read in CCLF/FHIR map
    # TODO: Prompt user to select which version of FHIR
    df = pd.read_csv('CCLF_FHIR3_mapping.csv')

    # Read in data sources
    # TODO: Prompt user to select data sources
    print("Ingesting FHIR data.")
    patient = open('patient.txt', 'r').read()
    patient = process_ndjson(patient)
    
    # coverage = open('coverage.txt', 'r').read()
    # coverage = process_ndjson(coverage)

    # eob = open('benefits.txt', 'r').read()
    # eob = process_ndjson(eob)

    # Map CCLF 8
    cclf8_map = {}
    print("Building data dictionary.")
    for index, row in df.loc[df['CCLF File'] == 'CCLF8'].iterrows():
        # Clarify data
        cclf_field = row['CCLF Claim Field Label'].replace('\n', '')
        fhir_field = row['FHIR R3 - Mapping'].replace('\n', '')
        
        cclf8_map[cclf_field] = fhir_field

    # Get field labels that match this CCLF file type
    cclf_headers = df.loc[df['CCLF File'] == 'CCLF8']['CCLF Claim Field Label'].to_list()
    cclf8 = pd.DataFrame(columns=cclf_headers)

    for beneficiary in tqdm(patient):
        row = []
        patient_reference = beneficiary['id']
        for column in cclf8.columns.to_list():
            if column == 'BENE_SEX_CD':
                fhir_path = cclf8_map[column].split('.')
                if fhir_path[0] == 'Patient':
                    exec_string = "row.append(beneficiary['{}'])".format("']['".join(fhir_path[1:]))
                    exec(exec_string)
                elif fhir_path[0] == 'Eob':
                    patient_reference
                    pass
                elif fhir_path[0] == 'Coverage':
                    patient_reference
                    pass
                
            else:
                row.append('')
        print(row)  # TODO: add to cclf8
        break
