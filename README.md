# Partner API Signing Client

## Description
This is a simple client program for setting up, generating and sending of signed requests to the Qredo Partner API.

### Building

Download and install [Golang](https://golang.org/doc/install)

Build the project:
```
go build -o partner-api-sign
```

### Keys
* Put your RSA private key in `pem` format in a file called `private.pem`
* Put your base64-encoded API key in a file called `apikey`

### Signing
To use the tool only for signing requests, run:
```
./partner-api-sign sign
```
You will be prompted to enter url and body. Make sure that you body has a new line at the end.
**Note:** the body should be a valid JSON and will be compacted before signing.

#### Example
GET request (no body)
```
» ./partner-api-sign sign
url: https://api.qredo.network/api/v1/p/company/search
body (hit return to end):

x-sign: OJk5vbrUPku37o-SiRXhIEzCy5cqBK9VKVvGeS1DD_HmxcjOBsIumayJhyl3oWJejtKeaGE3-TvPBDUNNuwCnlwVgrZl3Qq8ejL_TNEWwA6bJ-ZqmhK8SLLelTT2r6yBB_3DMaIXcn1A2cz_EsqhJP6JT-0kMIFUcAT6AKPRKHH9Laf1jpoXlNUGi-wuoquh-AiJczSQN1j5SOSOP2EkEd2T5NdSgxHbdER8g-eWpUyeaO8z2HkmrfALhUz3okiWDS9gYXzo7HyRyIgfrD5hUFiUzZJbyjDkRTvFo8jXZ-A9LA4-q7Rj0EjWFmftrNYZ-sXVuIx2BDbiX0cHXdTWwA
x-timestamp: 1605779886
```

PUT request (with body)
```
» ./partner-api-sign sign
url: https://api.qredo.network/api/v1/p/company/1eJFur7EANNaDjcqbm1ZgYFF5Nz
body (hit return to end):
{
  "name": "ACME Corp17",
  "city": "Paris",
  "country": "FR17",
  "domain": "acme4.com",
  "ref": "9827feed-5eae-4e80-bda3-drtreteraa7c3b97add",
  "anon_IOI": false,
  "anon_RFQ": false,
  "tc_only": false,
  "dir_listed": false
}

x-sign: APqwoFF-WdtwG9YDkrEVJWPCTQa9oXcIYsBkpO6Cwp9FxLjmh5uQmKMwIPATS4GAOGuRDSn874cO1TN77h_UQavmR86RH4IxIWqaZapHWVdaCntQs6r0j_0BSxNfbm8hHYpByxIJrtcCseuZ_XAIP5fJ-_MKp7OtiaM3_EHq3wRk_wZybJdRUbOq593PjDq622RBC2MDZntNr7HM-maGjgcoYh5U6qQHg9L_HgRcv7OXhqQ0h-CgLYaB-WJ1fUjxlgKAzy2nu1xdMHyKWN_gJSOJDfnDgdZ3mPxjZwfBvEReEfdXiVQq56Nq3LeOMXWERJ7x9vgGRsMbpHIYLWNooA
x-timestamp: 1605779886
```

### Sending requests
In subdirectory `requests/` create a new directory with the name of the request for example `company_new`. 
Create two files in that directory named `uri` and `body` e.g.:

file `requests/company_new/uri`:
```
POST https://api.qredo.network/api/v1/p/company
```
file `requests/company_new/body`:
```
{
  "name": "ACME Corp",
  "city": "Paris",
  "country": "FR",
  "domain": "example.com",
  "ref": "9827feed-5eae-4e80-bda3-drtreteraa7c3b97add"
}

```

then run:
```
./partner-api-sign company_new
```
where `company_new` is the directory name

#### Full example
```
» ./partner-api-sign company_new
POST https://api.qredo.network/api/v1/p/company
x-api-key: eyJrZXlfaWQiOiI5cUJ1cGRrYTctU19kdyIsImtleSI6IndOTGlLSC1EMmRKall5YUV1V3hoS0RmaG9XZTVqUTNib3JKdWZjeERzcFUifQ
x-sign: X66PWIazZomn4Q-kzRZX71_6FqtcGS6QYNaZfPV6OAZkxNXg9dpdZUfX_svfCpJVXRtZ3wyIxT2PThfEM7l94ihowpwWzZ6zUZ0Dk1dJFaMxeRxVmT8AGIiR4GncEgnNStcAcPaIsFwarq43lOJKMJgppR3gkEqB5i7n6sWA-EghssqG4lZdzfAvdLfeXUZfe_poQS1sPMSy8gDqbAeo0UIyrtSVG3Duwsh2_UPIsyqKu9fdmllErfRNTXoFZe7i7Ulr4y7Ya45gyUYEzdqT8Gm3t0OttQqEyyvIwx7nrmy1ACaZwmg-SQmWkJevug9xXMozLvgqgHw3_erOm_Kenw
x-timestamp: 1605779886
---
200 OK
{"company_id":"1eJFur7EANNaDjcqbm1ZgYFF5Nz","ref":"9827feed-5eae-4e80-bda3-drtreteraa7c3b97add"}
```

### Core client websocket

Connect to core client websocket feed endpoint:
```
./partner-api-sign websocket -url wss://api.qredo.network/api/v1/p/coreclient/ZupenzfrjAixU7G5AoDTpH113mwBa6enNfnhkEETqWix/feed
```

