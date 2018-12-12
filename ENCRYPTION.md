# File Encryption in BCDA API

When we deliver files via our data endpoint, we encrypt the files before sending them. You may wonder why we encrypt the files, since they are already being delivered via an encrypted channel (https). We do so for additional protection of all interested parties --- ourselves, you, and beneficiaries --- in the event of accidental or malicious activity like:

* a malicious party steals your credentials or an access token. The party can make requests and download files, but can't read the data without being able to decrypt it.
* a malicious party intercepts a file in transit. Again, they can't read the data without being able to decrypt it.

These are just some examples of the ways in which data can go astray. In general, we should all err on the side of caution with protecting this data.

Some best practices you can observe include 
* guarding your client credentials and RSA private key assiduously 
* storing them separately from each other
* limiting access to only those who must use them.

## How We Encrypt

We encrypt the file as the last step in producing it, immediately before we return a final jobStatus (the one that has a body and no X-Progress header). Please see API.md for more on job status. 

1. Generate a random 32 byte / 256 bit symmetric encryption key
1. Generate a random nonce (also known as an Initialization Value, or IV)
1. Read the data from the file
1. Use the the nonce and encryption key to encrypt the data with the AES-GCM algorithm. We do not append additional data. The resulting cipher text output begins with a byte indicating the size of the nonce, the nonce itself, and the encrypted data, and the additional data.
1. Encrypt the symmetric key using an rsa public key you provided us. We use the filename as the label, which is appended to the end of the key and separated from it by '|'.
1. Write the encrypted file to the appropriate data directory
1. Return the encrypted keys as hex-encoded strings in the body of the final jobStatus method, along with file download urls and other information. There can be two files, a data file and an error file. Each file will be encrypted with a different symmetric key. An example body follows:

```
{
    "transactionTime": "2018-12-11T06:29:56.723792Z",
    "request": "http://localhost:3000/api/v1/ExplanationOfBenefit/$export?encrypt=true",
    "requiresAccessToken": true,
    "output": [
        {
            "type": "ExplanationOfBenefit",
            "url": "http://localhost:3000/data/1/0c527d2e-2e8a-4808-b11d-0fa06baf8254.ndjson"
        }
    ],
    "keys": [
        "6c498a997001592ac05ace691fcf4a81724936c78937e24f90242c4f3081759f5365bef70a79eb0a6e145d22190b1178acf9f819399d27a4261efedf027642ca37d3f50cc0b941b105e35fc5b21cc785b171acb0ed299be16ff86fb457ff00d6855fefc9d403efdecbaca81ebffc85f8dbf1574d791640d392c5523482578ed232f7554880fa52d3471a4d919ab1ae8687e0442697cad7326aeb6ad0ddecaaeccaf61f952ef0cde2a3f15167b8854f8620440d8f1d9e09a0a39f1d04a3acf8178e5b6b28d9a062f09ff5fece3d16d9aacf7d43f4b94932d4f3268d1029f2874f3542ba71c858586393a80f45cb92b0cff9d2857b960045d733183d15c3599377|0c527d2e-2e8a-4808-b11d-0fa06baf8254.ndjson                                                                                    ",
        "4fd09523856ff24b9505c921973847fd4b1daf02753b3979373e8be8ea7da5418faa091535003a097ba8013582707535d0f5ea60380036c8be318094092c1936d0a80981ee2465009871c2fe56312e65239fea3785753684de19599d3219c545c24ad12018be4b86a39e742035e2559dcbe6169b6a3354f34bd2fbd569f88b70d3d1d13f62521693e779d3d2479d36515e086518bfd1140655d3b6100b05377b3ccacdfc10772c6a58178fae70b3a6a6ef897f64ae4a60045247b02331930ee6f15db45271afb2a432a8084170469458eef87c3a96ff6c4664c53b4867842b8650b3105860d29e87f43aad2c528d635f0eb02dc2bc905bf43bb1d1dd7f2cad3d|0c527d2e-2e8a-4808-b11d-0fa06baf8254-error.ndjson                                                                              "
    ],
    "error": [
        {
            "type": "OperationOutcome",
            "url": "http://localhost:3000/data/1/0c527d2e-2e8a-4808-b11d-0fa06baf8254-error.ndjson"
        }
    ],
    "KeyMap": {
        "0c527d2e-2e8a-4808-b11d-0fa06baf8254-error.ndjson                                                                              ": "4fd09523856ff24b9505c921973847fd4b1daf02753b3979373e8be8ea7da5418faa091535003a097ba8013582707535d0f5ea60380036c8be318094092c1936d0a80981ee2465009871c2fe56312e65239fea3785753684de19599d3219c545c24ad12018be4b86a39e742035e2559dcbe6169b6a3354f34bd2fbd569f88b70d3d1d13f62521693e779d3d2479d36515e086518bfd1140655d3b6100b05377b3ccacdfc10772c6a58178fae70b3a6a6ef897f64ae4a60045247b02331930ee6f15db45271afb2a432a8084170469458eef87c3a96ff6c4664c53b4867842b8650b3105860d29e87f43aad2c528d635f0eb02dc2bc905bf43bb1d1dd7f2cad3d",
        "0c527d2e-2e8a-4808-b11d-0fa06baf8254.ndjson                                                                                    ": "6c498a997001592ac05ace691fcf4a81724936c78937e24f90242c4f3081759f5365bef70a79eb0a6e145d22190b1178acf9f819399d27a4261efedf027642ca37d3f50cc0b941b105e35fc5b21cc785b171acb0ed299be16ff86fb457ff00d6855fefc9d403efdecbaca81ebffc85f8dbf1574d791640d392c5523482578ed232f7554880fa52d3471a4d919ab1ae8687e0442697cad7326aeb6ad0ddecaaeccaf61f952ef0cde2a3f15167b8854f8620440d8f1d9e09a0a39f1d04a3acf8178e5b6b28d9a062f09ff5fece3d16d9aacf7d43f4b94932d4f3268d1029f2874f3542ba71c858586393a80f45cb92b0cff9d2857b960045d733183d15c3599377"
    }
}
```

When you receive the final job status response, you should save the keys associated with the files so that they are available to you when you are ready to decrypt the file[s]. You should also save the output.url and the error.url.

When you are ready to decrypt the files, you make a request to output.url for the data file, and to error.url for the error file. These are protected endpoints, so you must obtain and use a token.

To decrypt the files, you must use the same algorithm (AES-GCM), and follow these steps:

1. Decrypt the key you saved from the final jobStatus body, using your rsa private key that is the mate to the public key we have.
2. Initialize the AES-GCM cipher with the symmetric key decoded in the previous step
3. Decode the cipher text
4. Do something useful with the data

Exactly how these steps are accomplished in code will vary with language and platform. We have some examples, implemented using commonly used languages, for you to consult.

## Show me the code

### We assume you have
* saved the encrypted symmetric key 
* downloaded the file
* access to the RSA private key
* sufficient memory to decode the file in one go
  (should provide a formula here for calculating memory needs)

### These examples all do the following

1. load the encrypted symmetric key, decoding it from the hex string
1. load the RSA private key
1. decrypt the symmetric key with the RSA private key
1. load the encrypted file
1. configure the cipher with the symmetric key
1. decode the encrypted file with the cypher
1. write the decoded data to a file

### Example Code

[go](https://gist.github.com/rnagle/f18099029460c7150cb5d68c3e06cb48)

[python](https://gist.github.com/rnagle/a2b8ecb7905337afaf00c060024d4fb4)


[java]() (in progress)


### Why this cipher

If you're interested in why we chose this algorithm, read [this page](https://proandroiddev.com/security-best-practices-symmetric-encryption-with-aes-in-java-7616beaaade9), which provides a high-level discussion and pointers to deeper references.

## PKI Key Pair

### You gave it to us or we gave it to you ... TKTK

document process here once we know what it is

## Limitations and Gotchas

* Encrypted data size is limited to 64GB
* If you are using Java, you need to know [this](https://stackoverflow.com/questions/6481627/java-security-illegal-key-size-or-default-parameters)



