#!/bin/bash

docker build . -t dell-r730-044.dsal.lab.eng.rdu2.redhat.com:5000/multus-cni-release:v3.1
docker run -it -v /tmp/multus:/opt/cni/bin:z dell-r730-044.dsal.lab.eng.rdu2.redhat.com:5000/multus-cni-release:v3.1

