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

def get_matching_rows(filename: str, key: str, reference: str) -> list:
    with open(filename, 'r') as source:
        matches = []
        for line in source:
            obj = json.loads(line)
            if obj[key]['reference'] == reference:
                return obj
    #             matches.append(obj)
    # return matches

def build_cclf(
        cclf_name: str,
        cclf_map: dict,
        patient_filename: str,
        coverage_filename: str,
        eob_filename: str,
        output_filename:str,
    ):
    """
    This function maps beneficiary data to the CCLF8 using objects from the Patient Resource as
    the primary reference.
    """
    print(f'Building {cclf_name.upper()}...')
    headers = [k for k in cclf_map]
    counter = get_file_length(patient_filename)
    with open(patient_filename, 'r') as patient, open(f'{cclf_name}_{output_filename}.csv', 'w') as cclf:
        csvwriter = csv.writer(cclf)
        csvwriter.writerow(headers)
        for x in tqdm(range(counter)):
            line = patient.readline()
            row = []
            bene = json.loads(line)
            bene_id = bene['id']

            for column in headers:
                try:
                    if (nm := cclf_map[column]) == 'Not mapped': row.append(nm)

                    elif cclf_map[column][0] == 'patient':
                        row.append(cclf_map[column][1](bene))

                    elif cclf_map[column][0] == 'eob':
                        # TODO: Regex is not actually needed. '-' is a symptom of seed data.
                        pattern = re.compile('[\W_]+')
                        patient_ref = f"Patient/{pattern.sub('', bene_id)}"
                        obj = get_matching_rows(eob_filename, 'patient', patient_ref)
                        row.append(cclf_map[column][1](obj))

                    elif cclf_map[column][0] == 'coverage':
                        patient_ref = f"Patient/{bene_id}"
                        # TODO: Need to confirm how to match Patient to other resources in all cases
                        # EX: BENE_MDCR_STUS_CD in CCLF8 has multiple matches

                        # matches = get_matching_rows(coverage_filename, 'beneficiary', patient_ref)
                        # for obj in matches:

                        obj = get_matching_rows(coverage_filename, 'beneficiary', patient_ref)
                        row.append(cclf_map[column][1](obj))

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


    build_cclf('cclf8', cclf_fhir3_maps.cclf8, patient, coverage, eob, output)
    build_cclf('cclf9', cclf_fhir3_maps.cclf9, patient, coverage, eob, output)
    build_cclf('cclf7', cclf_fhir3_maps.cclf7, patient, coverage, eob, output)
    # build_cclf('cclf1', cclf_fhir3_maps.cclf1, patient, coverage, eob, output)
    # build_cclf('cclf2', cclf_fhir3_maps.cclf2, patient, coverage, eob, output)
    # build_cclf('cclf3', cclf_fhir3_maps.cclf3, patient, coverage, eob, output)
    # build_cclf('cclf4', cclf_fhir3_maps.cclf4, patient, coverage, eob, output)
    # build_cclf('cclf5', cclf_fhir3_maps.cclf5, patient, coverage, eob, output)
    # build_cclf('cclf6', cclf_fhir3_maps.cclf6, patient, coverage, eob, output)
    # build_cclf('cclf_A', cclf_fhir3_maps.cclfA, patient, coverage, eob, output)
    # build_cclf('cclf_B', cclf_fhir3_maps.cclfB, patient, coverage, eob, output)
