# aze

a proxy to reduce bandwidth.

# usage

```sh
aze <dst> <src> <cap>

GLOBALSPEED="250M" SPEED="250M" SIZE="2500M"
netlisten -monitor :9079 -k 2 localhost:9090 to - "${SPEED}" > /dev/null && pkill -u ${USER} aze &
aze -g -monitor :9080 localhost:9090 localhost:9091 "${GLOBALSPEED}" &
aze -monitor :9081 gen "${SIZE}" "abcdefghijk" "${SPEED}" | nc --send-only localhost 9091 &
aze -monitor :9082 gen "${SIZE}" "abcdefghijk" "${SPEED}" | nc --send-only localhost 9091 &

2018/06/23 13:05:52 [netlisten :9079] 127.0.0.1:9090 accepted 127.0.0.1:33736
2018/06/23 13:05:52 [netlisten :9079] 127.0.0.1:9090 accepted 127.0.0.1:33744
2018/06/23 13:26:58 [aze gen :9081] written 2.6G
2018/06/23 13:26:58 [aze gen :9082] written 2.6G
2018/06/23 13:26:58 [aze :9080] 127.0.0.1:51628 -> 127.0.0.1:9090 copied 2.6G - 20.691467059s - 135M/s
2018/06/23 13:26:58 [netlisten :9079] 127.0.0.1:46738 -> - copied 2.6G - 20.691482663s - 135M/s
2018/06/23 13:26:58 [aze :9080] 127.0.0.1:51636 -> 127.0.0.1:9090 copied 2.6G - 20.687182277s - 135M/s
2018/06/23 13:26:58 [netlisten :9079] 127.0.0.1:46746 -> - copied 2.6G - 20.687222545s - 135M/s

expvarmon -i 500ms -ports ":9081"
expvarmon -i 500ms -ports ":9082"
expvarmon -i 500ms -ports ":9080"
expvarmon -i 500ms -ports ":9079"

```

# install

```sh
go get github.com/mh-cbon/aze
```
