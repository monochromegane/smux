#!/bin/sh

delay=$1
cert=$2
key=$3

go build -o bench cmd/bench/main.go

for proto in http http2 smux
do
  ./bench -mode server -proto $proto -delay $delay -cert $cert -key $key &
  pid=$!

  for concurrent in 10 50 100 150 200 250 300
  do
    for i in `seq 5`
    do
      sleep 10
      ./bench -mode client -concurrent $concurrent -proto $proto -delay $delay
    done
  done

  kill -9 $pid
done
