# aze

a proxy to reduce bandwidth.

# usage

```sh
aze <dst> <src> <cap>

GLOBALSPEED="10M" SPEED="250M" SIZE="250M" # it works, as long as you have enough ram to hold
 # (SPEED*1.2) + (SPEED*1.2) + (GLOBALSPEED*CONNLEN)
netlisten -monitor :9079 -k 2 localhost:9090 - "${SPEED}" > /dev/null && pkill -u ${USER} aze &
aze -g -monitor :9080 localhost:9090 localhost:9091 "${GLOBALSPEED}" &
aze -monitor :9081 gen "${SIZE}" "abcdefghijk" "${SPEED}" | nc --send-only localhost 9091 &
aze -monitor :9082 gen "${SIZE}" "abcdefghijk" "${SPEED}" | nc --send-only localhost 9091 &

2018/06/23 13:05:52 [netlisten :9079] 127.0.0.1:9090 accepted 127.0.0.1:33736
2018/06/23 13:05:52 [netlisten :9079] 127.0.0.1:9090 accepted 127.0.0.1:33744
2018/06/23 13:06:51 [netlisten :9079] 127.0.0.1:33736 -> - copied 300M - 58.590601168s - 5.2M/s
2018/06/23 13:06:51 [aze :9080] 127.0.0.1:38634 -> 127.0.0.1:9090 copied 300M - 59.001753663s - 5.1M/s
2018/06/23 13:06:51 [netlisten :9079] 127.0.0.1:33744 -> - copied 300M - 59.001499202s - 5.1M/s

expvarmon -i 500ms -ports ":9081"
expvarmon -i 500ms -ports ":9082"
expvarmon -i 500ms -ports ":9080"
expvarmon -i 500ms -ports ":9079"

```

# install

```sh
go get github.com/mh-cbon/aze
```
