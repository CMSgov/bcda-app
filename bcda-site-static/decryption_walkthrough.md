---
layout: home
title:  "File Decryption in BCDA"
date:   2019-03-13 09:21:12 -0500
description: Step-by-step instructions to assist with decrypting the NDJSON payload.
landing-page: live
gradient: "blueberry-lime-background"
subnav-link-gradient: "blueberry-lime-link"
sections:
  - Gathering the tools
  - Exploring the API with Swagger
  - Getting a token
  - Requesting a file
  - Decrypting a file
  - Reading further
  - Troubleshooting
ctas:
  - title: Visit our encryption overview
    link: ./encryption.html
  - title: Visit the BCDA Google Group
    link: https://groups.google.com/forum/#!forum/bc-api
    target: _blank
---

# BCDA File Decryption: An Example

## Gathering the tools
To complete this decryption example, you will need:

- Credentials [from the user guide](./user_guide.html#authentication-and-authorization)
- A file containing the example RSA `private key` ([example RSA private key](https://github.com/CMSgov/bcda-app/blob/master/shared_files/ATO_private.pem){:target="_blank"}) download
- You can write your own decryption code later based on the [documentation](./user_guide.html). This example uses the [python example decryption code](https://github.com/CMSgov/bcda-app/blob/master/encryption_utils/Python/decrypt.py){:target="_blank"}, which requires:
    - [Python](https://www.python.org/downloads/){:target="_blank"} installed
    - [decrypt.py](https://github.com/CMSgov/bcda-app/blob/master/encryption_utils/Python/decrypt.py){:target="_blank"} and [requirements.txt](https://github.com/CMSgov/bcda-app/blob/master/encryption_utils/Python/requirements.txt){:target="_blank"} downloaded to the same directory
    - Run `pip install -r requirements.txt` from that same directory to download any required libraries


## Exploring the API with Swagger
You will be interacting with the [BCDA API](./user_guide.html) in your browser using [Swagger](https://swagger.io){:target="_blank"}. This will enable you to authenticate, make requests, and everything else the API provides.

- Open your browser and navigate to:

### [https://sandbox.bcda.cms.gov/api/v1/swagger](./api/v1/swagger/){:target="_blank"}
<img src="assets/img/decrypt_walkthrough_01.png" alt="Screenshot of swagger" width="400" />

## Getting a token

[Use your credentials from the previous step](./user_guide.html#authentication-and-authorization){:target="_blank"} to get an access token in this step.

- Click the `Authorize` button
- In the Basic authorization section
    - Enter your `Client ID` in the `Username` box
    - Enter your `Client Secret` in the `Password` box
    - Click `Authorize`

<img src="assets/img/decrypt_walkthrough_02.png" alt="Available authorizations" width="350" />

- Click the `/auth/token` link

<img src="assets/img/decrypt_walkthrough_03.png" alt="Token endpoint" width="600" />

- Click `Try it out`, then `Execute`

<img src="assets/img/decrypt_walkthrough_04.png" alt="Getting a token" width="600" />

If you're successful, your token will appear as pictured in the `response body`.  You can now use this token to get full access to the API.

- Copy your new token to your clipboard
- Click the `Authorize` button
- In the `bearer_token` box:
    - Type "Bearer"
    - Add a space
    - Paste your `token` (Ctrl + V)

<img src="assets/img/decrypt_walkthrough_05.png" alt="Authorization dialog" width="350" />

- Click `Authorize`, then `Close`

<img src="assets/img/decrypt_walkthrough_06.png" alt="API menu" width="600" />

## Requesting a file
There are three types of encrypted files that can be downloaded: Coverage, Explanation of Benefit (EoB), and Patient. The example below id is for downloading the EoB.

- Click the `/api/v1/ExplanationOfBenefit/$export` link

Notice that there's quite a bit of documentation here, from the parameter types to possible kinds of server responses.

<img src="assets/img/decrypt_walkthrough_07.png" alt="Starting Explanation of Benefits job" width="600" />

- Click `Try it out`, then `Execute`

<img src="assets/img/decrypt_walkthrough_08.png" alt="Explanation of Benefits response" width="600" />

If you'd like to repeat this from the command line or implement this API call in code, look in the `Curl` section for the request you just made.  Not far below that under `Server response` you can see the response: an `HTTP` 202 success giving a link in the `content-location` header for status information on our EoB job.

- Note the job ID number at the end of this link.
- Open the job status section in Swagger (click `/api/v1/jobs/{jobID}`)

<img src="assets/img/decrypt_walkthrough_09.png" alt="Requesting job status" width="600" />

- Type the job ID you received
- Click `Execute`

<img src="assets/img/decrypt_walkthrough_10.png" alt="Job status response" width="600" />

1. Depending on the size of the file, the job may take some time.  If the job is not yet complete, status information will be shown.  Simply wait a few seconds and click execute again until the job completes. You will then get a result for a completed job as shown below.  You can download the file from the URL provided.
1. Your token will expire after a few minutes, and you may need to [get another from `/auth/token`](./decryption_walkthrough.html#getting-a-token) if it expires before you are finished interacting with the API.
1. Take special note of the new `KeyMap` section of the response.  To decrypt the file, you will need the filename (the first part of the keymap) and the [symmetric key](./encryption.html#how-we-encrypt) (the second part of the keymap), as shown above.  There are no spaces in either one.
1. Sometimes one or more data points are unavailable.  When this happens, the `error` section will contain a separate filename and symmetric key with a list of the patients involved.

- Copy these values from the `KeyMap` (filename and symmetric key) for later.

Your last API task is to download the encrypted file.

- Open the data file section in Swagger (click `/api/v1/jobs/{jobID}/{filename}`)
- Paste the job ID and filename into the appropriate boxes
- Click `Execute`

<img src="assets/img/decrypt_walkthrough_11.png" alt="Get data file (Swagger)" width="600" />

- Click the `Download file` link that appeared in the response section.  Note that a large sample file may take a while to download.

<img src="assets/img/decrypt_walkthrough_12.png" alt="Download file" width="600" />

## Decrypting a file
After downloading the file, move to the command line.  Navigate to the directory you saved `decrypt.py` and `requirements.txt` from the [Gathering the tools](#gathering-the-tools) section.

<img src="assets/img/decrypt_walkthrough_13.png" alt="Directory with decryption tool" width="450" />

Verify that Python is running properly.
- Run `decrypt.py` with the help argument (`python decrypt.py -h`).  You should get the response shown below.

<img src="assets/img/decrypt_walkthrough_14.png" alt="Decrypt.py syntax" width="600" />

- Rename the downloaded file with the filename you saved earlier. **This is extremely important as the file name is used as part of the file decryption process and using a different file name will cause decryption to fail.**

<img src="assets/img/decrypt_walkthrough_15.png" alt="Rename downloaded file" width="600" />

You are now ready to decrypt the file!  Your sample decryption tool will print the decrypted contents to the console, so you can send the output to a file.  Make sure to use the following syntax, with the entire command on the same line:

    python decrypt.py 
        --pk   [location_of_private_key] 
        --file [location_of_encrypted_file] 
        --key  [symmetric_key_value]
        > filename.txt

<img src="assets/img/decrypt_walkthrough_16.png" alt="Running decrypt.py" width="900" />

Take a look at the result.  If you do not see unencrypted [NDJSON](http://ndjson.org/){:target="_blank"} (two example lines shown below), then skip ahead to the [troubleshooting](#troubleshooting) section.

<img src="assets/img/decrypt_walkthrough_17.png" alt="Parsing decrypted file contents" width="900" />

## Reading further
* [Encryption documentation](./encryption.html)
* [API documentation](./user_guide.html)

## Troubleshooting
### Authentication problems in Swagger
- Did you use the credentials (Client ID and Client Secret) from the [user guide](./user_guide.html) with Basic authorization?
- After entering your credentials, did you get an access token from `/auth/token`?
- Did you put your access token in the `bearer_token` section of the authorization dialog?
- Has your token expired?  Use your credentials to get a new token from `/auth/token`.
- Is it possible you clicked on `Logout`?  Is the lock on the `Authorize` icon not closed?  Click it again, and after pasting your token in the `bearer_token` box, make sure to click the `Authorize` button.
- Are there any spaces or newlines in your token?  Remove them and paste it as a single line.
- Do you get an HTTP 504 `GATEWAY_TIMEOUT` error?  Make sure to add the word "Bearer" (and a space) before the token, as demonstrated in the [exploring the API](#exploring-the-api-with-swagger) section.

### Python not installed
- Is this your first time running Python on your system?  You might be interested in [this Windows installation guide](https://www.howtogeek.com/197947/how-to-install-python-on-windows/){:target="_blank"}

### Encryption issues
- The best practice would be to keep your private key in a separate, secured directory.  While you're testing the encryption feature for the first time, however, you may find it useful to have all the files in the same directory.
- Have you saved the encrypted file with exactly the filename provided by the API?  If not, rename it and try again.
- Is the symmetric key value provided with no spaces or newlines?  Double-check that no characters are missing from the beginning or end of the key.  