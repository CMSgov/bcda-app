# /usr/bin/python3

import csv
import json

from tqdm import tqdm
import pandas as pd


def process_ndjson(string_data: str):
    if string_data[-1:] == '\n':  # Handle possible trailing new line
        string_data = string_data[0:-1]
    string_data = '[' + string_data + ']'
    string_data = string_data.replace('\n', ',')
    return json.loads(string_data)

def build_exec_string(obj, fhir_path):
    return "row.append({}['{}'])".format(obj, "']['".join(fhir_path))


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
    coverage = []

    # eob = open('benefits.txt', 'r').read()
    # eob = process_ndjson(eob)
    eob = []

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
    cclf8 = [cclf_headers]

    # Using the patients as the primary reference
    for beneficiary in tqdm(patient):
        row = []
        patient_reference = beneficiary['id']
        for column in cclf_headers:
            try:
                fhir_path = cclf8_map[column].split('.')

                if column == 'BENE_MBI_ID':
                    import ipdb; ipdb.set_trace()



                if fhir_path[0] == 'Patient':
                    exec(build_exec_string(beneficiary, fhir_path[1:]))
                elif fhir_path[0] == 'Eob':
                    exec(build_exec_string(eob, fhir_path[1:]))
                elif fhir_path[0] == 'Coverage':
                    exec(build_exec_string(coverage, fhir_path[1:]))
                else:
                    row.append('Not Mapped')
            except:
                row.append('')
        cclf8.append(row)
        break  # TODO: delete this
    
    # Write CSV
    report_file = open('cclf8.csv', 'w')
    csvwriter = csv.writer(report_file)
    for item in cclf8:
        csvwriter.writerow(item)
    report_file.close()
