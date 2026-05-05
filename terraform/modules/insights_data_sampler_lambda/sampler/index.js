const { SSMClient, GetParameterCommand } = require("@aws-sdk/client-ssm");
const { Client } = require('pg')
const process = require('process')
const winston = require('winston');
const ssmClient = new SSMClient();

// logger that prints plainly to std out without the formatting of console.log, which enables easier data parsing
const plainLogger = winston.createLogger({
  level: 'info',
  format: winston.format.printf(({ message }) => message),
  transports: [
    new winston.transports.Console()
  ]
});

module.exports.handler = async (event, context) => {
  if (!event.query) {
    console.error("Insights Data Sampler: Query is undefined")
    return;
  }
  if (!event.name) {
    console.error("Insights Data Sampler: Event name is undefined")
    return;
  }
  if (!event.db_conn_string_env_var) {
    console.error("Insights Data Sampler: DB connection string env var is undefined")
    return;
  }

  try {
    const paramName = `/bcda/${process.env.ENV.toLowerCase()}/insights/${event.db_conn_string_env_var}`;
    const param = await getParameter(paramName);
    const connectionString = param.Parameter.Value

    const queryString = event.query;
    const queryResult = await executeQuery(connectionString, queryString);

    const contextRecord = {
      name: event.name,
      timestamp: Date.now(),
      requestId: context.awsRequestId
    }
    outputToStdOut(contextRecord, queryResult);
  } catch (error) {
    console.error(`Error executing data sampler lambda for query name ${event.name}:`, error)
  }
};

const getParameter = async (paramName) => {
  const input = {
    Name: paramName,
    WithDecryption: true
  };

  const command = new GetParameterCommand(input);
  return ssmClient.send(command);
};

const executeQuery = async (connectionString, queryString) => {
  const client = new Client({
    connectionString: connectionString,
    connectionTimeoutMillis: 5000,
    ssl: {
      rejectUnauthorized: false,
    }
  });

  await client.connect();
  const queryResult = await client.query(queryString);
  await client.end();
  return queryResult;
};

const outputToStdOut = (contextRecord, queryResult) => {
  const output = JSON.stringify({
    ...contextRecord,
    result: queryResult
  });
  plainLogger.info(output);
};
