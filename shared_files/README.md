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
* ATO_private.pem
* ATO_public.pem
  * a private/public key pair used as a standin for an ACO's key pair for encryption testing. The private key does not have a passphrase. Do not reuse for other purposes.
* api_unit_test_auth_private.pem
* api_unit_test_auth_public.pem
  * a private/public key pair used only for unit testing of api code that uses auth.Provider methods. The private key does not have a passphrase. The auth service will use these keys as its server signing key pair.
* localhost.crt
* localhost.key
  * our identity for testing ssl locally
