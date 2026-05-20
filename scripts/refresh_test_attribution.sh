# This file refreshes attribution data (CCLF & CSV) for all test ACOs.
# Output is deliberately suppressed to prevent sensitive data from being print to stdout during Github Action runs.
# psql must be installed to run commands.
#!/bin/bash

conn=""
force=""
env=""

usage() {
  echo "Usage: $0 -c <conn> [optional] -f <force>"
  exit 1
}

# Use getopts to parse command-line options
while getopts "c:e:f" opt; do
  case $opt in
    c)
      conn="$OPTARG"
      ;;
    f)
      force="true"
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      usage
      ;;
    :)
      echo "Option -$OPTARG requires an argument." >&2
      usage
      ;;
    *)
      echo "Option -$OPTARG requires an argument." >&2
      usage
      ;;
  esac
done

# Check if required options are set
if [ -z "$conn" ]; then
  echo "Error: Missing required options." >&2
  usage
fi

# Remove parsed options from the argument list
shift "$((OPTIND - 1))"

# list of test acos (same for all envs)
ACOS=("A9990" "A9991" "A9992" "A9993" "A9994" "A9998" "A9999" "TEST993" "TEST994" "SBXPS002" "SBXPL002")
PREV_PY_ACOS=("TEST995")
env=$(echo $conn | sed -e 's/.*bcda-\(.*\)-rds.*/\1/')

refresh_attribution () {
    echo "Updating test attribution for env: ${env}"
    for aco in ${ACOS[@]}; do
        echo "Updating: $aco"
            # attribution file
            psql -t $conn -c "UPDATE cclf_files SET timestamp = NOW()::date + timestamp::time, performance_year = (EXTRACT(YEAR FROM NOW())::int) % 100 WHERE id = (SELECT id FROM cclf_files WHERE name LIKE 'T.%' AND name LIKE '%ZC8Y%' AND aco_cms_id = '${aco}' ORDER BY timestamp DESC LIMIT 1);"
            # runout file
            psql -t $conn -c "UPDATE cclf_files SET timestamp = NOW()::date + timestamp::time, performance_year = (EXTRACT(YEAR FROM NOW())::int - 1) % 100 WHERE id = (SELECT id FROM cclf_files WHERE name LIKE 'T.%' AND name LIKE '%ZC8R%' AND aco_cms_id = '${aco}' ORDER BY timestamp DESC LIMIT 1);"
    done
    for aco in ${PREV_PY_ACOS[@]}; do
        echo "Updating: $aco with attribution for previous year"
            # attribution file and runout file
            psql -t $conn -c "UPDATE cclf_files SET timestamp = NOW()::date + timestamp::time, performance_year = (EXTRACT(YEAR FROM NOW())::int - 1) % 100 WHERE id = (SELECT id FROM cclf_files WHERE name LIKE 'T.%' AND name LIKE '%ZC8%' AND aco_cms_id = '${aco}' ORDER BY timestamp DESC LIMIT 1);"
    done
}

# allow automation to skip keyboard prompt
if [[ "$force" == "true" ]]
then
    refresh_attribution
else
    read -p "Refreshing attribution data for env: $env. Proceed? (y/n) " -n 1 -r
    if [[ $REPLY =~ ^[Yy]$ ]]
    then
        echo
        echo "Refreshing attribution..." 
        refresh_attribution
    else
        echo
        echo "Cancelled."
    fi
fi


