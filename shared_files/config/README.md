# Sensitive Docker Configuration Files

The files committed in this directory hold secret information, and are encrypted with [Ansible Vault](https://docs.ansible.com/ansible/2.4/vault.html).  The directory should look something like this:

- `.gitignore`: set to ignore all filenames (except this one) that do not end with .encrypted. 
- `bfd-dev-test-cert.pem`: an unencrypted copy of the following file
- `bfd-dev-test-cert.pem.encrypted`: a certificate identifying and authorizing this application to retrieve claims data; encrypted
- `bfd-dev-test-key.pem`: an unencrypted copy of the following file
- `bfd-dev-test-key.pem.encrypted`: the private key for the above certificate; encrypted
- `local.env`: an unencrypted copy of the following file
- `local.env.encrypted`: a .env file with sensitive environmental variables used by Docker (see `docker-compose.yml` and `docker-compose.test.yml`); encrypted
- `README.md`: this file

## Setup
### Password
- See a team member for the Ansible Vault password
- Create a file named `.vault_password` in the root directory of the repository
- Place the Ansible Vault password in this file

### Git hook
**IMPORTANT:** Files containing sensitive information are enumerated in the `.secrets` file in the root of the repository. If you want to protect the contents of a file using the `./scripts/secrets` helper script, it must match a pattern listed in `.secrets`.

To avoid committing and pushing unencrypted secret files, use the included `scripts/pre-commit` git pre-commit hook by running the following script from the repository root directory:
```
cp ops/pre-commit .git/hooks
```

## Managing secrets
* Temporarily decrypt files by running the following command from the repository root directory: 
```
./ops/secrets --decrypt
```
* While files are decrypted, make a local copy with:
```
./ops/copy.encrypted
```
* Encrypt changed files with:
```
./ops/secrets --encrypt <filename>
```