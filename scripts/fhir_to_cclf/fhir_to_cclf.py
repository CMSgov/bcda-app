# /usr/bin/python3

import argparse, csv, datetime, json, re
from pathlib import Path

from tqdm import tqdm

import cclf_fhir3_maps


def get_file_length(filename: str) -> int:
    """Return the number of lines in a file."""
    counter = 0
    with open(filename, 'r') as source:
        for line in source:
            counter += 1
    return counter

def get_matching_row(filename: str, key: str, reference: str) -> dict:
    with open(filename, 'r') as source:
        for line in source:
            obj = json.loads(line)
            if obj[key]['reference'] == reference:
                return obj


def build_cclf8(patient_filename: str, coverage_filename: str, eob_filename: str, output_filename:str):
    """
    This function maps beneficiary data to the CCLF8 using objects from the Patient Resource as
    the primary reference.
    
    The CCLF8 provides identifying information for beneficiaries and can be used as a
    unifier to help build the rest of the CCLF file structure.
    """
    print("Building CCLF8...")
    cclf8_map = cclf_fhir3_maps.cclf8
    headers = [k for k in cclf8_map]
    counter = get_file_length(patient_filename)

    with open(patient_filename, 'r') as patient, open(f'cclf8_{output_filename}.csv', 'w') as cclf8:
        csvwriter = csv.writer(cclf8)
        csvwriter.writerow(headers)
        for x in tqdm(range(counter)):
            line = patient.readline()
            row = []
            bene = json.loads(line)
            bene_id = bene['id']

            for column in headers:
                try:
                    if (nm := cclf8_map[column]) == 'Not mapped': row.append(nm)

                    elif cclf8_map[column][0] == 'patient':
                        row.append(cclf8_map[column][1](bene))

                    elif cclf8_map[column][0] == 'eob':
                        # TODO: Confirm regex is actually needed.
                        # Not sure if the '-' is a symptom of seed data.
                        pattern = re.compile('[\W_]+')
                        patient_ref = f"Patient/{pattern.sub('', bene_id)}"
                        obj = get_matching_row(eob_filename, 'patient', patient_ref)
                        row.append(cclf8_map[column][1](obj))

                    elif cclf8_map[column][0] == 'coverage':
                        patient_ref = f"Patient/{bene_id})"
                        obj = get_matching_row(coverage_filename, 'beneficiary', patient_ref)
                        row.append(cclf8_map[column][1](obj))

                except:  # Not every field will be present
                    row.append('')
            csvwriter.writerow(row)

def build_cclf9(patient_filename: str, eob_filename: str, output_filename:str):
    """
    This function maps FHIR Data to the CCLF9
    """
    print("Building CCLF9...")
    cclf9_map = cclf_fhir3_maps.cclf9
    headers = [k for k in cclf9_map]
    counter = get_file_length(patient_filename)
    with open(patient_filename, 'r') as patient, open(f'cclf9_{output_filename}.csv', 'w') as cclf9:
        csvwriter = csv.writer(cclf9)
        csvwriter.writerow(headers)
        for x in tqdm(range(counter)):
            line = patient.readline()
            row = []
            bene = json.loads(line)
            bene_id = bene['id']

            for column in headers:
                try:
                    if (nm := cclf9_map[column]) == 'Not mapped': row.append(nm)
                except:  # Not every field will be present
                    row.append('')
            csvwriter.writerow(row)

def error_handling(patient, coverage, eob, output):
    # Confirm all data sources are valid
    if not Path(patient).is_file():
        print(f'Source file "{patient}" does not exist.')
        exit()
    if not Path(coverage).is_file():
        print(f'Source file "{coverage}" does not exist.')
        exit()
    if not Path(eob).is_file():
        print(f'Source file "{eob}" does not exist.')
        exit()
    
    # Verify no output filename conflicts will exist
    if Path(output).is_file():
        print(f'A file with the name "{output}" already exists in this directory.')
        exit()


if __name__=='__main__':
    parser = argparse.ArgumentParser(description='Convert FHIR data stored in an NDJSON file to CCLF format.')
    parser.add_argument('patient', metavar='<patient>', type=str, nargs=1,
                    help='Patient Resource source file to be converted (.json, .ndjson, .txt)')
    parser.add_argument('coverage', metavar='<coverage>', type=str, nargs=1,
                    help='Coverage Resource source file to be converted (.json, .ndjson, .txt)')
    parser.add_argument('eob', metavar='<eob>', type=str, nargs=1,
                    help='Explanatino of Benefits source file to be converted (.json, .ndjson, .txt)')
    parser.add_argument('-f', '--fhir', metavar='<version>', type=str, nargs=1, required=True,
                    help='The version of FHIR represented in your data (3, 4)')
    parser.add_argument('output', metavar='<output_filename>', type=str, nargs='?',
                    default=f'{datetime.datetime.now().strftime("%Y-%m-%d-%H:%M:%S")}',
                    help='Optional desired output filename. Each report name will be prepended. Ex: cclf8_my_file_name.csv')
    
    args = parser.parse_args()
    patient, coverage, eob, output = args.patient[0], args.coverage[0], args.eob[0], args.output
    error_handling(patient, coverage, eob, output)

    # TODO: FHIR Version indication should impact conversion
    build_cclf8(patient, coverage, eob, output)
    build_cclf9(patient, eob, output)
