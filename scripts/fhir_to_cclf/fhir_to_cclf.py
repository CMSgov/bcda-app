# /usr/bin/python3

import argparse, csv, datetime, json, re
from pathlib import Path

from tqdm import tqdm

import cclf_fhir3_maps

def process_ndjson(string_data: str):
    """
    Turn 
    """
    if string_data[-1:] == '\n':
        string_data = string_data[0:-1]
    string_data = '[' + string_data + ']'
    string_data = string_data.replace('\n', ',')
    return json.loads(string_data)


def build_cclf8(patient, coverage, eob):
    """
    This function maps bene data to the CCLF8 using objects from the Patient Resource as
    the primary reference.
    
    The CCLF8 provides identifying information for beneficiaries and can be used as a
    unifier to help build the rest of the CCLF file structure.
    """
    cclf8 = []
    cclf8_headers = [k for k in cclf8_map]
    cclf8.append(cclf8_headers)

    for beneficiary in tqdm(patient):
        row = []
        bene_id = beneficiary['id']
        for column in cclf8_headers:
            try:
                if cclf8_map[column] == 'Not mapped':
                    row.append('Not mapped')
                elif cclf8_map[column][0] == 'patient':
                    row.append(cclf8_map[column][1](beneficiary))
                elif cclf8_map[column][0] == 'eob':
                    # TODO: Confirm regex is actually needed.
                    # Not sure if the '-' is a symptom of seed data.
                    pattern = re.compile('[\W_]+')
                    patient_ref = f"Patient/{pattern.sub('', bene_id)}"
                    for obj in eob:
                        if obj['patient']['reference'] == patient_ref:
                            row.append(cclf8_map[column][1](obj))
                elif cclf8_map[column][0] == 'coverage':
                    for obj in coverage:
                        patient_ref = f"Patient/{bene_id})"
                        if obj['beneficiary']['reference'] == patient_ref:
                            row.append(cclf8_map[column][1](obj))
                else:
                    row.append(column)    
            except:
                row.append('')
        cclf8.append(row)
    
    # Write CSV
    report_file = open('cclf8.csv', 'w')
    csvwriter = csv.writer(report_file)
    for item in cclf8:
        csvwriter.writerow(item)
    report_file.close()



if __name__=='__main__':
    # parser = argparse.ArgumentParser(description='Convert FHIR data stored in an NDJSON file to CCLF format.')
    # parser.add_argument('input', metavar='<inputfile>', type=str, nargs=1,
    #                 help='Source file to be converted (.json, .ndjson, .txt)')
    # parser.add_argument('output', metavar='<output_filename>', type=str, nargs='?',
    #                 default=f'flat_fhir_output_{datetime.datetime.now().strftime("%Y-%m-%d-%H:%M:%S")}.csv',
    #                 help='Optional desired output filename')
    
    # args = parser.parse_args()
    # inputfile, outputfile = args.input[0], args.output

    # # Error handling
    # if not Path(inputfile).is_file():
    #     print(f'Source file "{inputfile}" does not exist.')
    #     exit()
    # if Path(outputfile).is_file():
    #     print(f'A file with the name "{outputfile}" already exists in this directory.')
    #     exit()

    # TODO: User must indicate which version of FHIR
    cclf8_map = cclf_fhir3_maps.cclf8  # FHIR 3 CCLF8

    # Read in data sources
    # TODO: User must indicate data sources
    print("Ingesting FHIR data.")
    patient = open('sources/patient.ndjson', 'r').read()
    patient = process_ndjson(patient)
    
    coverage = open('sources/coverage.ndjson', 'r').read()
    coverage = process_ndjson(coverage)

    eob = open('sources/benefit.ndjson', 'r').read()
    eob = process_ndjson(eob)

    build_cclf8(patient, coverage, eob)
