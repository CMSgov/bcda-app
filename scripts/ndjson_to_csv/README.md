# NDJSON to CSV Script

## Overview
This python script can be used to generate a CSV from NDJSON data. It is understood that users of this script may not have software development experience and these instructions were written in an attempt to be understood by all audiences.

## Setup
### Install Python
Running this script requires that you have Python 3 installed on your machine. Mac OS ships with Python 2 and 3, but Windows users will need to manually install.

You can download the most recent version of Python at [python.org/downloads](https://www.python.org/downloads/).

You can verify that Python has been installed correctly by opening a Command Prompt or Terminal and running the following:

```
python3 --version
```

Your maching should output the active version of Python.

```
>> python3 --version
Python 3.8.2
```

### Build a Virtual Environment
These steps will only need to be completed once. If you already have a virtual environment and just need to run the script, proceed to the next section.

The purpose of the virtual environment is to keep the tools used here separate from other system resources. Everything done in the following steps can be undone by simply deleting the folder you're using as a workspace.

*Note: It is highly recommended that you create a new folder for this work on your computer to isolate your working files.*
- Add this script to your working folder.
- Using a Command Prompt or Terminal, navigate to your working folder.
```
cd path/of/your/folder
```
- Create the virtual environment. 
```
python3 -m venv venv
```
- Activate the virtual environment.
```
source ./env/bin/activate
```
- Install the Python library Pandas.
```
pip install pandas
```

### Running the Script
These instructions assume that you already have NDJSON data on your computer from the BCDA ACO API that you would like to convert to a CSV format.

- Move the NDJSON files you would like to convert into your working folder.
- Using a Command Prompt or Terminal, navigate to your working folder.
```
cd path/of/your/folder
```
- Activate the virtual environment.
```
source ./env/bin/activate
```
- Run the script using the following command.
```
python3 ndjson_to_csv.py
```
The script will provide prompts to ingest a TXT or JSON file to convert to a flat CSV. The CSV will be output in the same place the script was run from. 