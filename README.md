# aze

a proxy to reduce bandwidth.

# usage

```sh
aze <dst> <src> <cap>

rm rcv.data;netlisten -t 2s -k 1 localhost:9090 - > rcv.data && pkill -u ${USER} aze &
aze -t 2s localhost:9090 localhost:9091 888K || rm rcv.data && pkill -u ${USER} watch &
aze gen 6.66M "a" | nc --send-only localhost 9091 &
watch -n 1 ls -lah rcv.data

```

# install

```sh
go get github.com/mh-cbon/aze
```
