# ToDo
[x] include TG by go.mod
[x] just dup something
[x] github actions
[x] do somethng with "github.com/ledgerwatch/turbo-geth/internal/ethapi" package
[] try ethash.NewFaker()
[x] move flags to package
[x] add way to include all TG eth_ and debug_ methods by default
[x] make project minimalistic
[] rename chaindata to datadir? (now datadir logic smashed across Config.instanceDir and utils.setDataDir)

Readme

To change TG version use exact git commit:
```
go get -u github.com/ledgerwatch/turbo-geth@7a64e9d4b65df614f3a62ad7a513cdcaea508fb0
```