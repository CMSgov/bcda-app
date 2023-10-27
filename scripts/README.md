# ACO API Component Scripts
These are a series of scripts that may be helpful in working with data provided from the ACO API.

## Setup
There are a few prerequistes in order to work with these scripts. It is understood that users will have varrying levels of experience and these instructions were written in an attempt to be understood by all audiences.

### Install Python
All of these scripts require that you have a version of Python 3 installed on your machine. Mac OS ships with Python 3, but Windows users may need to manually install.

You can download the most recent version of Python at [python.org/downloads](https://www.python.org/downloads/).

You can verify that Python has been installed correctly by opening a Command Prompt or Terminal and running the following:

```
python3 --version
```

Your machine should output the active version of Python.

```
>> python3 --version
Python 3.8.2
```

### Build a Virtual Environment
These steps will only need to be completed once. The purpose of the virtual environment is to keep the tools used here separate from other system resources. Everything done in the following steps can be undone by simply deleting the folder you're using as a workspace.

TODO: clarify setting up a python workspace

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
source ./venv/bin/activate
```

- Install Python dependencies/.
```
pip install -r requirements.txt
```

## NDJSON to CSV Data Flattener
This script can be used to generate a single CSV from NDJSON data.

### Using the Script
These instructions assume that you already have NDJSON data on your computer from the BCDA ACO API that you would like to convert to a CSV format. You can use synthetic data from the ACO Sandbox API or real data from the live API.

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
