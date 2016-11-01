FROM ubuntu:16.04

ADD bin/voyager-ipam-service voyager-ipam-service

CMD ./voyager-ipam-service
