FROM davidwalter/debian-stretch-slim
COPY bin/loadbalancer /opt/sbin/loadbalancer
CMD [ "/opt/sbin/loadbalancer" ]
