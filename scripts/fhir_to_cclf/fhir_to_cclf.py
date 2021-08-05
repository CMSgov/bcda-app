# /usr/bin/python3

import csv
import json

from tqdm import tqdm
# import pandas as pd

import cclf_fhir3_maps

def process_ndjson(string_data: str):
    if string_data[-1:] == '\n':  # Handle possible trailing new line
        string_data = string_data[0:-1]
    string_data = '[' + string_data + ']'
    string_data = string_data.replace('\n', ',')
    return json.loads(string_data)

def build_exec_string(obj, fhir_path):
    return "row.append({}['{}'])".format(obj, "']['".join(fhir_path))


if __name__=='__main__':
    # TODO: Prompt user to select which version of FHIR
    cclf8_map = cclf_fhir3_maps.cclf8  # FHIR 3 CCLF8

    # Read in data sources
    # TODO: Prompt user to select data sources
    print("Ingesting FHIR data.")
    patient = open('patient.txt', 'r').read()
    patient = process_ndjson(patient)
    
    coverage = open('coverage.txt', 'r').read()
    coverage = process_ndjson(coverage)

    # eob = open('benefits.txt', 'r').read()
    # eob = process_ndjson(eob)
    eob = []

    # Using the patients as the primary reference
    cclf8 = []
    cclf8_headers = [k for k in cclf8_map]
    cclf8.append(cclf8_headers)

    for beneficiary in tqdm(patient):
        # import ipdb; ipdb.set_trace()
        row = []
        patient_reference = beneficiary['id']
        for column in cclf8_headers:
            try:
                if cclf8_map[column] == 'Not mapped':
                    row.append('Not mapped')
                elif cclf8_map[column][0] == 'patient':
                    row.append(cclf8_map[column][1](beneficiary))
                elif cclf8_map[column][0] == 'eob':
                    pass
                elif cclf8_map[column][0] == 'coverage':
                    pass
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
