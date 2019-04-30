# Shared Files

What's in here and why?

* cclf directory
  * sample cclf files related to figuring out attribution of beneficiary data
* synthetic_beneficiary_data
  * see README.md in this directory
* ATO_private.pem
* ATO_public.pem
  * a private/public key pair used as a standin for an ACO's key pair for encryption testing. The private key does not have a passphrase. Do not reuse for other purposes.
* api_unit_test_auth_private.pem
* api_unit_test_auth_public.pem
  * a private/public key pair used only for unit testing of api code that uses auth.Provider methods. The private key does not have a passphrase. The auth service will use these keys as its server signing key pair.
* bb-dev-test-cert.pem
* bb-dev-test-key.pem
  * our identity for mutual tls to blue button's testing environment
* localhost.crt
* localhost.key
  * our identity for testing ssl locally
