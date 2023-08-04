TOKEN=$2
USERNAME=$1

#todo: docker or path to docker?
echo "DEBUG: USER=$1 TOKEN=$2"

echo $TOKEN | docker login ghcr.io -u $USERNAME --password-stdin
 
