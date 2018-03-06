ip1=$(docker exec -ti `docker ps -qf "name=docker_poa1"` /bin/hostname -i) 
ip2=$(docker exec -ti `docker ps -qf "name=docker_poa2"` /bin/hostname -i)
ip3=$(docker exec -ti `docker ps -qf "name=docker_poa3"` /bin/hostname -i)

id1=$(geth attach http://localhost:18545 --exec admin.nodeInfo.id)
id2=$(geth attach http://localhost:28545 --exec admin.nodeInfo.id)
id3=$(geth attach http://localhost:38545 --exec admin.nodeInfo.id)

echo enode: // $id1 "@" 111 $ip1 30303 222 
echo $id2 $ip2 
echo $id3 $ip3 
