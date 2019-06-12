from __future__ import print_function

import argparse
import binascii
import os
import re
import sys
import codecs

from argparse import RawTextHelpFormatter

try:
    from Crypto.Cipher import PKCS1_OAEP, AES
    from Crypto.PublicKey import RSA
    from Crypto.Hash import SHA256
except ImportError:
    print("Install requirements first: pip install -r requirements.txt", file=sys.stderr)
    raise SystemExit

# Match nonce and tag sizes used by BCDA
# Golang defaults: https://golang.org/src/crypto/cipher/gcm.go#L146
GCM_NONCE_SIZE = 12
GCM_TAG_SIZE = 16


def init():
    parser = argparse.ArgumentParser(
        description='Decrypt a BCDA payload.',
        formatter_class=RawTextHelpFormatter
    )
    parser.add_argument(
        '--key', dest='key', type=str,
        help="encrypted symmetric key used for file decryption (hex-encoded string)"
    )
    parser.add_argument(
        '--file', dest='file', type=str,
        help="location of encrypted file"
    )
    parser.add_argument(
        '--pk', dest='pk', type=str,
        help="location of private key to use for decryption of symmetric key"
    )

    args = parser.parse_args()

    if not args.key or not args.file or not args.pk:
        print("missing argument(s)", file=sys.stderr)
        raise SystemExit

    if not valid_uuid(args.file):
        print("""File name does not appear to be valid.
Please use the exact file name from the job status endpoint (i.e., of the format: <UUID>.ndjson).""",
              file=sys.stderr)
        raise SystemExit

    return args


def decrypt_cipher(ct, key):
    nonce = ct.read(GCM_NONCE_SIZE)
    cipher = AES.new(key, AES.MODE_GCM, nonce=nonce, mac_len=GCM_TAG_SIZE)
    ciphertext = ct.read()
    return cipher.decrypt_and_verify(
        ciphertext[:-GCM_TAG_SIZE],
        ciphertext[-GCM_TAG_SIZE:]
    )


def decrypt_file(private_key, encrypted_key, filepath):
    base = os.path.basename(filepath)
    cipher = PKCS1_OAEP.new(key=private_key, hashAlgo=SHA256, label=base.encode('utf-8'))
    decrypted_key = cipher.decrypt(encrypted_key)

    with open(filepath, 'rb') as fh:
        result = decrypt_cipher(fh, decrypted_key).decode("utf-8")

    print(result)


def get_private_key(loc):
    with open(loc, 'r') as fh:
        return RSA.importKey(fh.read())

def valid_uuid(filename):
    path, name = os.path.split(filename)
    uuid = name.split(".")[0]
    regex = re.compile('^[a-f0-9]{8}-?[a-f0-9]{4}-?4[a-f0-9]{3}-?[89ab][a-f0-9]{3}-?[a-f0-9]{12}\Z', re.I)
    match = regex.match(uuid)
    return bool(match)

def main():
    args = init()
    if sys.version_info[0] < 3:
        sys.stdout = codecs.getwriter('utf-8')(sys.stdout)
    ek = binascii.unhexlify(args.key)
    pk = get_private_key(args.pk)
    decrypt_file(pk, ek, args.file)

if __name__ == "__main__":
    main()