# /usr/bin/python3

import datetime
import json

from functools import reduce

from tqdm import tqdm
import pandas as pd

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
    print("This tool can convert FHIR data stored in an NDJSON file to a CSV format.")
    while True:
        # Ingest the report results
        filename = input("Enter the name of the NDJSON file you wish to convert: ")
        f = open(filename, 'r')
        
        # Format data
        string_json = f.read()
        if string_json[-1:] == '\n':  # Handle possible trailing new line
            string_json = string_json[0:-1]
        string_json = '[' + string_json + ']'
        string_json = string_json.replace('\n', ',')
        data = json.loads(string_json)

        list_data_dict = [flatten(obj) for obj in data]  # Flatten data
        print("Creating DataFrames")
        list_dataframes = [  # Create DataFrames
            pd.DataFrame(obj, index=[list_data_dict.index(obj)])
            for obj in tqdm(list_data_dict)
        ]
        print("Merging DataFrames")
        output = reduce(lambda x, y: x.append(y), tqdm(list_dataframes))  # Merge DataFrames
        
        output_filename = '{}_{}.csv'.format(
            filename,
            datetime.datetime.today().date()
        )
        print("Creating CSV")
        output.to_csv(output_filename, index=False)

        print("File converted successfully. Output file '{}'".format(output_filename))
        if input("Would you like to convert another file? (y/n) ") == 'n':
            print("Goodbye.")
            exit()