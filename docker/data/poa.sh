hwclock -s
mkdir /dyndata
cp -fR /data/$1/* /dyndata
/geth --datadir /dyndata init /data/gomet-genesis.json
acc=$(cat /dyndata/account)
/geth --datadir /dyndata --rpc --rpcaddr 0.0.0.0 --rpcapi eth,web3,personal,clique,admin,net --mine --unlock $acc --password /dyndata/password
