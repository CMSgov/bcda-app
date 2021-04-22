# Shared Files

What's in here and why?

* cclf directory
  * sample cclf files related to figuring out attribution of beneficiary data
* decrypted
  * git-ignored, decrypted files sourced from the `encrypted/` directory
* encrypted
  * sensitive configuration values that should be encrypted in the repository
    * `bfd-dev-test-cert.pem`: a certificate identifying and authorizing this application to retrieve claims data; encrypted
    * `bfd-dev-test-key.pem`: the private key for the above certificate; encrypted
    * `local.env`: a .env file with sensitive environmental variables used by Docker (see `docker-compose.yml` and `docker-compose.test.yml`); encrypted
* synthetic_beneficiary_data
  * see README.md in this directory
* localhost.crt
* localhost.key
  * our identity for testing ssl locally
