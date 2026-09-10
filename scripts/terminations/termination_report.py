import argparse
import datetime
import os
import pandas as pd
import pathvalidate

from argparse import RawTextHelpFormatter

DESCRIPTION = """
Supports BCDA operations by summarizing the upcoming terminations based on each model's termination file. The output will be written to an xlsx file.

Example:
    python3 ./termination_report.py \\
        --ssp './path/to/ssp/termination_file.xlsx' \\
        --kcc './path/to/kcc/termination_file.xlsx' \\
        --reach './path/to/reach/termination_file.xlsx' \\
        --iota './path/to/iota/termination_file.xlsx' \\
"""

def main(ssp_filepath: str, kcc_filepath: str, reach_filepath: str, iota_filepath: str, output_filepath: str):
    today = datetime.datetime.today()
    models = []
    cols = ['Entity ID', 'Legal Business Name', 'Termination Date']

    # Read in and prepare the files for each model

    ## Shared Savings Program
    if ssp_filepath:
        ssp = pd.read_excel(ssp_filepath, sheet_name='Termination Tracker Report')[
                                ['ACO_ID', 'Legal Entity Name', 'Termination Date']]
        ssp.columns = cols
        models.append(find_terminations_after(ssp, today))
        print("Added upcoming terminations from the Shared Savings Program")

    ## Kidney Care Choices
    if kcc_filepath:
        # We use the index of 'My Entities Agreement Details' sheet here because .read_excel fails to find it by name for KCC files
        kcc = pd.read_excel(kcc_filepath, sheet_name=1)[ 
                                ['Entity ID', 'Entity Legal Business Name', 'Effective Termination Date']]
        kcc.columns = cols
        models.append(find_terminations_after(kcc, today))
        print("Added upcoming terminations from the Kidney Care Choices model")

    ## ACO REACH
    if reach_filepath:
        reach = pd.read_excel(reach_filepath, sheet_name='My Entities Agreement Details')[
                                ['Entity ID', 'Entity Legal Business Name', 'Effective Termination Date']]
        reach.columns = cols
        models.append(find_terminations_after(reach, today))
        print("Added upcoming terminations from the ACO REACH model")

    ## IOTA
    if iota_filepath:
        iota = pd.read_excel(iota_filepath)[
                                ['Participant ID', 'Participant Legal Business Name', 'Effective Termination Date']]
        iota.columns = cols
        models.append(find_terminations_after(iota, today))
        print("Added upcoming terminations from the IOTA model")

    pd.concat(models, ignore_index=True).sort_values(by=['Termination Date','Entity ID']).to_excel(output_filepath, index=False)
    print(f"Wrote the upcoming terminations to '{output_filepath}'")

def find_terminations_after(model: pd.DataFrame, after_dt: datetime):
    model['Termination Date'] = pd.to_datetime(model['Termination Date'])
    model = model.loc[model['Termination Date'] >= after_dt]
    return model

def validate_filetype(parser, arg, ext: str):
    try:
        pathvalidate.validate_filepath(arg)
    except pathvalidate.ValidationError as e:
        parser.error(e)

    if not os.path.exists(arg):
        parser.error(f"The file '{arg}' does not exist")
        return

    _, arg_ext = os.path.splitext(arg)
    if arg_ext.lower() != ext.lower():
        parser.error(f"Expected a {ext.lower()} file, received '{arg}'")
        return

    return arg

def sanitize_validate_destination(parser, arg):
    sanitized_arg = pathvalidate.sanitize_filepath(arg)

    if not os.path.exists(os.path.dirname(os.path.abspath(sanitized_arg))):
        parser.error(f"The directory of the destination file does not exits, recieved '{arg}'")
        return
    
    if os.getcwd() != os.path.commonpath([os.getcwd(), os.path.abspath(sanitized_arg)]):
        parser.error(f"The destination must be in the current directory or sub-directory, received '{arg}'")
        return
    
    return sanitized_arg

if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description=DESCRIPTION,
        formatter_class=RawTextHelpFormatter
    )

    parser.add_argument(
        '--ssp', dest='ssp_filepath', type=lambda x: validate_filetype(parser, x, ".xlsx"),
        help='The path to the Termination Tracker Report (xlsx) for the Shared Savings Program. This file can be downloaded from the "Reporting" tab in ACO-MS.'
    )

    parser.add_argument(
        '--kcc', dest='kcc_filepath', type=lambda x: validate_filetype(parser, x, ".xlsx"),
        help='The path to the KCC Agreement Management Data (xlsx) for the Kidney Care Choices Model. This file can be downloaded from the "Dashboards & Reports" tab in 4i.'
    )

    parser.add_argument(
        '--reach', dest='reach_filepath', type=lambda x: validate_filetype(parser, x, ".xlsx"),
        help='The path to the ACO REACH Agreement Management (xlsx) Data for the ACO REACH Model. This file can be downloaded from the "Dashboards & Reports" tab in 4i.'
    )

    parser.add_argument(
        '--iota', dest='iota_filepath', type=lambda x: validate_filetype(parser, x, ".xlsx"),
        help='The path to the IOTA Participant List (xlsx) Data for the Increasing Organ Transplant Access (IOTA) Model. This file can be downloaded from the "Dashboards" tab in 4i.'
    )

    parser.add_argument(
        '--output', dest='output_filepath', default='upcoming terminations.xlsx', 
        type=lambda x: sanitize_validate_destination(parser, x),
        help='The path of a file to write the output to. If not specified, this will be "upcoming terminations.xlsx"'
    )

    args = parser.parse_args()

    main(ssp_filepath=args.ssp_filepath, kcc_filepath=args.kcc_filepath, reach_filepath=args.reach_filepath, iota_filepath=args.iota_filepath, output_filepath=args.output_filepath)