# /usr/bin/python3

import argparse, csv, datetime, json
from pathlib import Path

from tqdm import tqdm


def flatten(data, parent_key='', sep='.'):
    items = []
    for k, v in data.items():
        new_key = parent_key + sep + k if parent_key else k
        if isinstance(v, dict):
            items.extend(flatten(v, new_key, sep).items())
        elif isinstance(v, list):
            if not isinstance(v[0], (str, int)):
                if len(v) > 1:
                    for index, item in zip(range(len(v)), v):
                        if index > 0:
                            items.extend(flatten(
                                item,
                                '{}{}'.format(new_key, index+1),
                                sep
                            ).items())
                        else:
                            items.extend(flatten(item, new_key, sep).items())
                else:
                    for item in v:
                        items.extend(flatten(item, new_key, sep).items())
            else:
                list_strings = [str(item) for item in v]
                items.append((new_key, ' '.join(list_strings)))
        else:
            items.append((new_key, v))
    return dict(items)


if __name__=='__main__':
    parser = argparse.ArgumentParser(description='Convert FHIR data stored in an NDJSON file to a CSV format.')
    parser.add_argument('input', metavar='<inputfile>', type=str, nargs=1,
                    help='Source file to be converted (.json, .ndjson, .txt)')
    parser.add_argument('output', metavar='<output_filename>', type=str, nargs='?',
                    default=f'flat_fhir_output_{datetime.datetime.now().strftime("%Y-%m-%d-%H:%M:%S")}.csv',
                    help='Optional desired output filename')
    
    args = parser.parse_args()
    inputfile, outputfile = args.input[0], args.output
    
    if outputfile[-4:] != '.csv':
        outputfile = f'{outputfile}.csv'    

    # Error handling
    if not Path(inputfile).is_file():
        print(f'Source file "{inputfile}" does not exist.')
        exit()
    if Path(outputfile).is_file():
        print(f'A file with the name "{outputfile}" already exists in this directory.')
        exit()

    counter = 0

    print(f'Determining file size...')
    with open(inputfile, 'r') as source_file:
        for line in source_file:  # Count lines to build progress indicator
            counter += 1
        source_file.seek(0)
        
        print(f'Flattening "{inputfile}"...')
        col_headers = set()
        for x in tqdm(range(counter)):
            line = source_file.readline()
            if line[-1:] == '\n':
                line = line[0:-1]
            data = flatten(json.loads(line))
            col_headers.update(data.keys())

    # Merge data referencing headers
    print('Merging data and creating CSV...')
    list_headers = sorted(list(col_headers))
    with open(outputfile, "x") as output_file:
        with open(inputfile, 'r') as source_file:
            csvwriter = csv.writer(output_file)
            csvwriter.writerow(list_headers)
            for x in tqdm(range(counter)):
                row = []
                line = source_file.readline()
                if line[-1:] == '\n':
                    line = line[0:-1]
                data = flatten(json.loads(line))
                for col in list_headers:
                    try:
                        row.append(data[col])
                    except:
                        row.append('')
                csvwriter.writerow(row)

    print(f'File flattened successfully. Output file "{outputfile}"')
    exit()
