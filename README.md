Readme

To upgrade TG version - use exact git commit. For example:
```
go get -u github.com/ledgerwatch/turbo-geth@213cf2cbec5792c9b23ab3c3ffa7f1b662d4188e
```

Run (this assumes turbo-geth is running on the same machine with option `--private.api.addr 127.0.0.1:9090`): 
```
go run . --private.api.addr 127.0.0.1:9090 --http.api example
```
alternatively
```
make
./project-1 --private.api.addr 127.0.0.1:9090 --http.api example
```

Docs: https://github.com/ledgerwatch/turbo-geth/blob/master/cmd/rpcdaemon/Readme.md

## Custom API methods
The custom method is called `example_localFork`.

### Input
The input consists of 3 paramters (`params`):
1. Block number (or "latest" to use the latest available block) after which the execution will be applied
2. Array of unsigned transactions. Each transaction has fields `from`, `to`, `gas` (hex number in quotes), `value` (in wei, hex number in quotes), `nonce` (hex value in quotes), `input` (hex value in quotes). The field `to` can be omitted, which means it is contract-creating transaction.
3. Array of queries (the same input as for `eth_call`). A query has the same fields, except for `nonce`.

### Output
The output is the object with two fields `txResults`, and `queryResults`:
1. The field `txResults` contains an array, with the same number of elements as the second input (transactions) has. Each element in the `txResults` corresponds to the transaction in the input. Each element has fields `UsedGas`, `Err`, and `ReturnData`. Fields `Err` and `ReturnData` can have `null` values.
2. The field `queryResults` contains an array, with the same number of elements as the second input (transactions) has. Each element in the `txResults` corresponds to the transaction in the input. Each element has fields `UsedGas`, `Err`, and `ReturnData`. Fields `Err` and `ReturnData` can have `null` values.

## Example
Input
```
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"example_localFork", "params": ["latest", [
{"from": "0x0000000000000000000000000000000000000000", "gas": "0x100000", "gasPrice": "0x1000", "value": "0x34000", "nonce": "0x0", "input": ""},
{"from": "0x0000000000000000000000000000000000000000", "gas": "0x100000", "gasPrice": "0x1000", "value": "0x34000", "nonce": "0x1", "input": ""}
], [
{"from": "0x0000000000000000000000000000000000000000", "gas": "0x100000", "gasPrice": "0x1000", "value": "0x34000", "input": ""},
{"from": "0x0000000000000000000000000000000000000000", "gas": "0x100000", "gasPrice": "0x1000", "value": "0x34000", "input": ""}
]], "id":1}' localhost:8545
```

Output
```
{"jsonrpc":"2.0","id":1,"result":{"txResults":[{"UsedGas":53000,"Err":null,"ReturnData":null},{"UsedGas":53000,"Err":null,"ReturnData":null}],"queryResults":[{"UsedGas":53000,"Err":null,"ReturnData":null},{"UsedGas":53000,"Err":null,"ReturnData":null}]}}
```
