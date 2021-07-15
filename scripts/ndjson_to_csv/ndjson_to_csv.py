# /usr/bin/python3

import csv
import datetime
import json


def flatten(data, parent_key='', sep='.'):
    items = []
    for k, v in data.items():
        new_key = parent_key + sep + k if parent_key else k
        if isinstance(v, dict):
            items.extend(flatten(v, new_key, sep=sep).items())
        elif isinstance(v, str):
            items.append((new_key, v))
        elif isinstance(v, list):
            if not isinstance(v[0], str):
                for item in v:
                    items.extend(flatten(item, new_key, sep=sep).items())
            else:
                items.append((new_key, ' '.join(v)))
        else:
            items.append((new_key, v))
    return dict(items)

def convert_to_csv(template:dict, data:list):
    output = []
    
    # Build columns
    columns_dict = flatten(template)
    columns = [k for k in columns_dict]
    
    output.append(columns)

    # Flatten data
    data_dict = [flatten(obj) for obj in data]

    # Populate CSV
    for obj in data_dict:
        row = []
        for column in columns:
            try:
                row.append(obj[column])
            except:
                row.append('')
        output.append(row)

    return output


if __name__=='__main__':
    print("This tool can convert FHIR data stored in an NDJSON file to a CSV format.")
    while True:
        # Ingest the report results and format the data
        filename = input("Enter the name of the NDJSON file you wish to convert: ")
        f = open(filename, 'r')
        string_json = '[' + f.read()[0:-1] + ']'  # Drop the linebreaks
        string_json = string_json.replace('\n', ',')
        data = json.loads(string_json)
        template = None
        while template is None:
            resource_type = input("What kind of resource is this? (enter 1, 2, or 3)\
                \n1. Patient\n2. Coverage\n3. Benefits\n")
            if resource_type == '1':
                template = json.loads(open("templates/patient.json", "r").read())
            elif resource_type == '2':
                template = json.loads(open("templates/coverage.json", "r").read())
            elif resource_type == '3':
                template = json.loads(open("templates/benefits.json", "r").read())
        
        # Create the file and write results
        report_results = convert_to_csv(template, data)
        report_file = open('{}_{}.csv'.format(
            filename,
            datetime.datetime.today().date()
            ), 'w')
        csvwriter = csv.writer(report_file)
        for item in report_results:
            csvwriter.writerow(item)
        report_file.close()

        print("File converted successfully. Output file '{}'".format(report_file.name))
        if input("Would you like to convert another file? (y/n) ") == 'n':
            print("Goodbye.")
            exit()
