# aze

a proxy to reduce bandwidth.

# usage

```sh
aze <dst> <src> <cap>

GLOBALSPEED="50M" SPEED="250M" SIZE="100M" # it works, as long as you have enough ram to hold
 # (SPEED*1.2) + (SIZE*1.2) + (GLOBALSPEED*CONNLEN)
netlisten -monitor :9079 -k 2 localhost:9090 - "${SPEED}" > /dev/null && pkill -u ${USER} aze &
aze -g -monitor :9080 localhost:9090 localhost:9091 "${GLOBALSPEED}" &
aze -monitor :9081 gen "${SIZE}" "a" "${SPEED}" | nc --send-only localhost 9091 &
aze -monitor :9082 gen "${SIZE}" "a" "${SPEED}" | nc --send-only localhost 9091 &

2018/06/23 13:05:52 [netlisten :9079] 127.0.0.1:9090 accepted 127.0.0.1:33736
2018/06/23 13:05:52 [netlisten :9079] 127.0.0.1:9090 accepted 127.0.0.1:33744
2018/06/23 13:19:14 [aze gen :9082] written 200M
2018/06/23 13:19:15 [aze :9080] 127.0.0.1:46584 -> 127.0.0.1:9090 copied 200M - 8.726957667s - 25M/s
2018/06/23 13:19:15 [netlisten :9079] 127.0.0.1:41694 -> - copied 200M - 8.727028177s - 25M/s
2018/06/23 13:19:16 [aze gen :9081] written 200M
2018/06/23 13:19:16 [aze :9080] 127.0.0.1:46576 -> 127.0.0.1:9090 copied 200M - 10.171754951s - 20M/s
2018/06/23 13:19:16 [netlisten :9079] 127.0.0.1:41686 -> - copied 200M - 10.171575107s - 20M/s

expvarmon -i 500ms -ports ":9081"
expvarmon -i 500ms -ports ":9082"
expvarmon -i 500ms -ports ":9080"
expvarmon -i 500ms -ports ":9079"

```

# install

```sh
go get github.com/mh-cbon/aze
```
