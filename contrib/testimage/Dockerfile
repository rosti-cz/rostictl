FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get upgrade -y && apt-get install -y dropbear
RUN mkdir -p /root/.ssh
ADD test_ssh_keys/id_ed25519.pub /root/.ssh/
ADD test_ssh_keys/id_rsa_secured.pub /root/.ssh/
ADD test_ssh_keys/id_rsa.pub /root/.ssh/
RUN cat /root/.ssh/id_ed25519.pub /root/.ssh/id_rsa_secured.pub /root/.ssh/id_rsa.pub > /root/.ssh/authorized_keys

ENTRYPOINT [ "/usr/sbin/dropbear", "-F" ]
