# /usr/bin/python3

import csv, json, sys, tempfile

from functools import reduce
from pathlib import Path

import pandas as pd
from tqdm import tqdm


col_headers = set()
help_message = 'Convert FHIR data stored in an NDJSON file to a CSV format.\n\
\n\
Usage:\n\
\n\
    ndjson_to_csv.py <inputfile> <output_filename>\n\
\n\
\n\
<inputfile>        Source file to be converted (.json, .ndjson, .txt)\n\
<output_filename>  Optional desired output filename\n\
\n\
-h, --help         view help message\n'

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
                col_headers.add(new_key)
                items.append((new_key, ' '.join(list_strings)))
        else:
            col_headers.add(new_key)
            items.append((new_key, v))
    return dict(items)


if __name__=='__main__':
    """
    Convert FHIR data stored in an NDJSON file to a CSV format.

    Usage: ndjson_to_csv.py <inputfile> <output_filename>
    
    <inputfile>        Source file to be converted (.json, .ndjson, .txt)
    <output_filename>  Optional desired output filename
    """

    argv = sys.argv[1:]
    
    if '-h' in argv or '--help' in argv:
        print(help_message)
        exit()
    if len(argv) > 2:
        print('Too many arguments.')
        print(help_message)
        exit()

    inputfile, outputfile = argv[0], argv[1]
    
    if outputfile[-4:] != '.csv':
        outputfile = f'{outputfile}.csv'

    # Error handling
    if not Path(inputfile).is_file():
        print(f'Source file "{inputfile}" does not exist.')
        exit()
    if Path(outputfile).is_file():
        print(f'A file with the name "{outputfile}" already exists in this directory.')
        exit()

    # Create Tempfile
    tf = tempfile.NamedTemporaryFile(prefix=f'{outputfile}_temp_', mode="w+t", delete=True)
    counter = 0

    # Flatten data structures
    print(f'Flattening "{inputfile}"...')
    with open(inputfile, 'r') as source_file:
        for line in source_file:
            if line[-1:] == '\n':
                line = line[0:-1]
            data = json.loads(line)
            tf.write(f'{json.dumps(flatten(data))}\n')
            counter += 1

    # Merge data referencing headers
    print('Merging data and creating CSV...')
    list_headers = sorted(list(col_headers))  # Lists are ordered
    tf.seek(0)
    with open(outputfile, "x") as output_file:
        csvwriter = csv.writer(output_file)
        csvwriter.writerow(list_headers)
        for x in tqdm(range(counter)):
            row = []
            line = tf.readline()[0:-1]  # Every line has a trailing return
            data = json.loads(line)
            for col in list_headers:
                try:
                    row.append(data[col])
                except:
                    row.append('')
            csvwriter.writerow(row)

    print(f'File flattened successfully. Output file {outputfile}.csv')
    exit()
